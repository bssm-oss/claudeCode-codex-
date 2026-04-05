package workspace

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Workspace struct {
	Root string
}

type SearchResult struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Match  string `json:"match"`
	Column int    `json:"column"`
}

func New(root string) (Workspace, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Workspace{}, fmt.Errorf("resolve workspace root: %w", err)
	}
	return Workspace{Root: absRoot}, nil
}

func (w Workspace) Resolve(path string) (string, error) {
	clean := filepath.Clean(path)
	resolved := clean
	if !filepath.IsAbs(clean) {
		resolved = filepath.Join(w.Root, clean)
	}
	resolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	if resolved != w.Root && !strings.HasPrefix(resolved, w.Root+string(os.PathSeparator)) {
		return "", errors.New("path escapes workspace root")
	}
	return resolved, nil
}

func (w Workspace) List(maxEntries int) ([]string, error) {
	if maxEntries <= 0 {
		maxEntries = 200
	}

	entries := make([]string, 0, maxEntries)
	err := filepath.WalkDir(w.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(w.Root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkip(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			rel += "/"
		}
		entries = append(entries, rel)
		if len(entries) >= maxEntries {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.SkipAll) {
		return nil, fmt.Errorf("walk workspace: %w", err)
	}

	sort.Strings(entries)
	return entries, nil
}

func (w Workspace) Read(path string, offset, limit int) ([]string, error) {
	resolved, err := w.Resolve(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(resolved)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if offset < 1 {
		offset = 1
	}
	if limit <= 0 {
		limit = 200
	}

	lines := make([]string, 0, limit)
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if lineNo < offset {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d: %s", lineNo, scanner.Text()))
		if len(lines) >= limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	return lines, nil
}

func (w Workspace) Search(pattern string, maxMatches int) ([]SearchResult, error) {
	if maxMatches <= 0 {
		maxMatches = 50
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile pattern: %w", err)
	}

	results := make([]SearchResult, 0, maxMatches)
	err = filepath.WalkDir(w.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(w.Root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkip(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()

		scanner := bufio.NewScanner(file)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			loc := re.FindStringIndex(line)
			if loc == nil {
				continue
			}
			results = append(results, SearchResult{
				Path:   rel,
				Line:   lineNo,
				Match:  line,
				Column: loc[0] + 1,
			})
			if len(results) >= maxMatches {
				return fs.SkipAll
			}
		}
		return scanner.Err()
	})
	if err != nil && !errors.Is(err, fs.SkipAll) {
		return nil, fmt.Errorf("search workspace: %w", err)
	}

	return results, nil
}

func (w Workspace) Replace(path, oldText, newText string, createIfMissing bool) (string, error) {
	resolved, err := w.Resolve(path)
	if err != nil {
		return "", err
	}

	before, err := os.ReadFile(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !createIfMissing {
				return "", fmt.Errorf("file does not exist: %s", path)
			}
			if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
				return "", fmt.Errorf("create parent dir: %w", err)
			}
			if err := os.WriteFile(resolved, []byte(newText), 0o644); err != nil {
				return "", fmt.Errorf("create file: %w", err)
			}
			return previewDiff("", newText), nil
		}
		return "", fmt.Errorf("read file: %w", err)
	}

	updated := string(before)
	if oldText == "" {
		updated = newText
	} else {
		if !strings.Contains(updated, oldText) {
			return "", errors.New("old_text not found in file")
		}
		updated = strings.Replace(updated, oldText, newText, 1)
	}

	if err := os.WriteFile(resolved, []byte(updated), 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return previewDiff(string(before), updated), nil
}

func previewDiff(before, after string) string {
	if before == after {
		return "no changes"
	}
	return fmt.Sprintf("--- before\n%s\n+++ after\n%s", before, after)
}

func shouldSkip(rel string, isDir bool) bool {
	parts := strings.Split(rel, string(os.PathSeparator))
	for _, part := range parts {
		if part == ".git" || part == ".idea" || part == "node_modules" {
			return true
		}
	}
	if isDir && strings.HasPrefix(filepath.Base(rel), ".") && rel != ".claude" {
		return true
	}
	return false
}
