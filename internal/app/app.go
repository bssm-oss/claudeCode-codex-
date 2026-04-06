package app

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bssm-oss/claudeCode-codex-/internal/auth"
	"github.com/bssm-oss/claudeCode-codex-/internal/config"
	"github.com/bssm-oss/claudeCode-codex-/internal/extensions"
	openaiprovider "github.com/bssm-oss/claudeCode-codex-/internal/provider/openai"
	"github.com/bssm-oss/claudeCode-codex-/internal/session"
	"github.com/bssm-oss/claudeCode-codex-/internal/vcs"
	"github.com/bssm-oss/claudeCode-codex-/internal/workspace"
)

const developerPrompt = `You are a clean-room Go terminal coding agent. Work only with the provided tools. Always inspect before editing. Prefer concise answers. For shell, edits, branch creation, and commits, expect an approval gate outside the model.`

type App struct {
	in  io.Reader
	out io.Writer
	err io.Writer
}

func New(in io.Reader, out, err io.Writer) *App {
	return &App{in: in, out: out, err: err}
}

func (a *App) Run(ctx context.Context, args []string) error {
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	paths, err := config.ResolvePaths(homeDir, projectRoot)
	if err != nil {
		return err
	}

	cfg, err := config.Load(paths, projectRoot)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return a.runChat(ctx, cfg, paths, "")
	}

	switch args[0] {
	case "chat":
		prompt := ""
		if len(args) > 1 {
			prompt = strings.Join(args[1:], " ")
		}
		return a.runChat(ctx, cfg, paths, prompt)
	case "doctor":
		return a.runDoctor(ctx, cfg, paths)
	case "login":
		return a.runLogin(paths, args[1:])
	case "config":
		return a.runConfig(paths, cfg)
	case "help", "--help", "-h":
		fallthrough
	default:
		return a.printHelp()
	}
}

func (a *App) printHelp() error {
	_, err := fmt.Fprintln(a.out, `ccagent is a clean-room Go terminal coding agent.

Usage:
  ccagent help
  ccagent doctor
  ccagent login [--api-key KEY] [--device-auth] [--issuer URL] [--client-id ID]
  ccagent config
  ccagent chat [prompt]

Environment:
  OPENAI_API_KEY   API key for OpenAI-compatible requests`)
	return err
}

func (a *App) runDoctor(ctx context.Context, cfg config.Config, paths config.Paths) error {
	store := auth.NewStore(paths)
	creds, err := store.Load(environmentMap())
	if err != nil {
		return err
	}

	gitClient := vcs.New(cfg.Workspace)
	status := map[string]any{
		"workspace":      cfg.Workspace,
		"model":          cfg.Model,
		"approval_mode":  cfg.ApprovalMode,
		"config_file":    paths.ConfigFile,
		"auth_file":      paths.AuthFile,
		"transcript_dir": cfg.Transcripts,
		"auth_mode":      creds.Mode(),
		"has_api_key":    strings.TrimSpace(creds.OpenAIAPIKey) != "",
		"has_chatgpt":    creds.HasChatGPTAccessToken(),
		"account_id":     creds.AccountID(),
		"is_git_repo":    gitClient.IsRepository(ctx),
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(a.out, string(data))
	return err
}

func (a *App) runLogin(paths config.Paths, args []string) error {
	apiKey := ""
	deviceAuth := false
	issuerURL := auth.DefaultIssuerURL
	clientID := auth.DefaultClientID
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--api-key":
			if i+1 < len(args) {
				apiKey = args[i+1]
				i++
			}
		case "--device-auth":
			deviceAuth = true
		case "--issuer":
			if i+1 < len(args) {
				issuerURL = args[i+1]
				i++
			}
		case "--client-id":
			if i+1 < len(args) {
				clientID = args[i+1]
				i++
			}
		}
	}

	store := auth.NewStore(paths)
	if deviceAuth {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		deviceCode, err := auth.RequestDeviceCode(ctx, issuerURL, clientID)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(a.out, "Open this URL and enter the code:\n%s\n\nCode: %s\n", deviceCode.VerificationURL, deviceCode.UserCode)
		if err := store.CompleteDeviceCodeLogin(ctx, issuerURL, clientID, deviceCode); err != nil {
			return err
		}
		_, err = fmt.Fprintf(a.out, "Saved ChatGPT/Codex auth to %s\n", paths.AuthFile)
		return err
	}

	if strings.TrimSpace(apiKey) == "" {
		_, _ = fmt.Fprint(a.out, "Enter OPENAI_API_KEY: ")
		scanner := bufio.NewScanner(a.in)
		if !scanner.Scan() {
			return errors.New("no api key provided")
		}
		apiKey = strings.TrimSpace(scanner.Text())
	}

	if err := store.Save(auth.Credentials{OpenAIAPIKey: apiKey}); err != nil {
		return err
	}

	_, err := fmt.Fprintf(a.out, "Saved API key to %s\n", paths.AuthFile)
	return err
}

func (a *App) runConfig(paths config.Paths, cfg config.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.out, "config file: %s\n%s\n", paths.ConfigFile, data)
	return err
}

func (a *App) runChat(ctx context.Context, cfg config.Config, paths config.Paths, initialPrompt string) error {
	store := auth.NewStore(paths)
	creds, err := store.Load(environmentMap())
	if err != nil {
		return err
	}

	provider, err := openaiprovider.New(creds, cfg.Model, cfg.OpenAIBaseURL, cfg.ChatGPTBaseURL)
	if err != nil {
		return err
	}

	ws, err := workspace.New(cfg.Workspace)
	if err != nil {
		return err
	}

	transcript, err := session.New(cfg.Transcripts)
	if err != nil {
		return err
	}
	defer func() { _ = transcript.Close() }()
	hooks, err := extensions.LoadHooks(cfg)
	if err != nil {
		return err
	}

	gitClient := vcs.New(cfg.Workspace)
	reader := bufio.NewReader(a.in)
	previousResponseID := ""
	if err := a.runHooks(ctx, reader, ws, transcript, hooks, "session_start", map[string]string{
		"CCAGENT_TRANSCRIPT_PATH": transcript.Path(),
	}); err != nil {
		return err
	}

	if strings.TrimSpace(initialPrompt) != "" {
		return a.runTurn(ctx, reader, provider, ws, gitClient, transcript, hooks, &previousResponseID, initialPrompt)
	}

	_, _ = fmt.Fprintln(a.out, "ccagent chat started. Type /exit to leave.")
	_, _ = fmt.Fprintf(a.out, "Transcript: %s\n", transcript.Path())
	for {
		_, _ = fmt.Fprint(a.out, "> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read prompt: %w", err)
		}
		prompt := strings.TrimSpace(line)
		if prompt == "" {
			continue
		}
		if prompt == "/exit" || prompt == "/quit" {
			return nil
		}
		if err := a.runTurn(ctx, reader, provider, ws, gitClient, transcript, hooks, &previousResponseID, prompt); err != nil {
			return err
		}
	}
}

func (a *App) runTurn(
	ctx context.Context,
	reader *bufio.Reader,
	provider *openaiprovider.Client,
	ws workspace.Workspace,
	gitClient vcs.Git,
	transcript *session.Transcript,
	hooks []extensions.Hook,
	previousResponseID *string,
	prompt string,
) error {
	_ = transcript.Append("user", map[string]string{"prompt": prompt})
	if err := a.runHooks(ctx, reader, ws, transcript, hooks, "before_model", map[string]string{
		"CCAGENT_PROMPT": prompt,
	}); err != nil {
		return err
	}
	tools := toolDefinitions()
	input := openaiprovider.TurnInput{PreviousResponseID: *previousResponseID, Prompt: prompt}

	for i := 0; i < 8; i++ {
		turnCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		result, err := provider.Complete(turnCtx, input, developerPrompt, tools)
		cancel()
		if err != nil {
			return err
		}
		*previousResponseID = result.ResponseID

		if len(result.ToolCalls) == 0 {
			if result.Text != "" {
				_, _ = fmt.Fprintln(a.out, result.Text)
				_ = transcript.Append("assistant", map[string]string{"content": result.Text})
				if err := a.runHooks(ctx, reader, ws, transcript, hooks, "after_model", map[string]string{
					"CCAGENT_PROMPT":    prompt,
					"CCAGENT_ASSISTANT": result.Text,
				}); err != nil {
					return err
				}
			}
			return nil
		}

		toolOutputs := make([]openaiprovider.ToolOutput, 0, len(result.ToolCalls))
		for _, call := range result.ToolCalls {
			if err := a.runHooks(ctx, reader, ws, transcript, hooks, "before_tool", map[string]string{
				"CCAGENT_TOOL_NAME": call.Name,
				"CCAGENT_PROMPT":    prompt,
			}); err != nil {
				return err
			}
			toolOutput, err := a.executeTool(ctx, reader, ws, gitClient, call)
			if err != nil {
				toolOutput = fmt.Sprintf("tool error: %v", err)
			}
			_, _ = fmt.Fprintf(a.out, "[tool:%s]\n%s\n", call.Name, toolOutput)
			_ = transcript.Append("tool", map[string]string{"name": call.Name, "output": toolOutput})
			if err := a.runHooks(ctx, reader, ws, transcript, hooks, "after_tool", map[string]string{
				"CCAGENT_TOOL_NAME":   call.Name,
				"CCAGENT_TOOL_OUTPUT": toolOutput,
				"CCAGENT_PROMPT":      prompt,
			}); err != nil {
				return err
			}
			toolOutputs = append(toolOutputs, openaiprovider.ToolOutput{CallID: call.ID, Output: toolOutput})
		}
		input = openaiprovider.TurnInput{PreviousResponseID: result.ResponseID, ToolOutputs: toolOutputs}
	}

	return errors.New("model exceeded tool loop limit")
}

func (a *App) runHooks(ctx context.Context, reader *bufio.Reader, ws workspace.Workspace, transcript *session.Transcript, hooks []extensions.Hook, event string, values map[string]string) error {
	for _, hook := range hooks {
		if hook.Event != event {
			continue
		}
		if err := a.confirm(reader, fmt.Sprintf("Run hook %s from %s? %s", hook.Event, hook.Source, hook.Command)); err != nil {
			_ = transcript.Append("hook", map[string]string{
				"event":   hook.Event,
				"source":  hook.Source,
				"command": hook.Command,
				"status":  "rejected",
				"error":   err.Error(),
			})
			return err
		}
		env := os.Environ()
		env = append(env,
			"CCAGENT_HOOK_EVENT="+hook.Event,
			"CCAGENT_HOOK_SOURCE="+hook.Source,
			"CCAGENT_WORKSPACE="+ws.Root,
		)
		for key, value := range values {
			env = append(env, key+"="+value)
		}
		hookCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		cmd := exec.CommandContext(hookCtx, "sh", "-lc", hook.Command)
		cmd.Dir = ws.Root
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		cancel()
		text := string(out)
		_, _ = fmt.Fprintf(a.out, "[hook:%s:%s]\n%s\n", hook.Event, hook.Source, text)
		status := "completed"
		if err != nil {
			status = "failed"
		}
		_ = transcript.Append("hook", map[string]string{
			"event":   hook.Event,
			"source":  hook.Source,
			"command": hook.Command,
			"output":  text,
			"status":  status,
		})
		if err != nil {
			return fmt.Errorf("hook %s (%s) failed: %w", hook.Event, hook.Source, err)
		}
	}
	return nil
}

func (a *App) executeTool(ctx context.Context, reader *bufio.Reader, ws workspace.Workspace, gitClient vcs.Git, call openaiprovider.ToolCall) (string, error) {
	switch call.Name {
	case "list_files":
		var args struct {
			MaxEntries int `json:"max_entries"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		entries, err := ws.List(args.MaxEntries)
		if err != nil {
			return "", err
		}
		return strings.Join(entries, "\n"), nil
	case "read_file":
		var args struct {
			Path   string `json:"path"`
			Offset int    `json:"offset"`
			Limit  int    `json:"limit"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		lines, err := ws.Read(args.Path, args.Offset, args.Limit)
		if err != nil {
			return "", err
		}
		return strings.Join(lines, "\n"), nil
	case "search_files":
		var args struct {
			Pattern    string `json:"pattern"`
			MaxMatches int    `json:"max_matches"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		results, err := ws.Search(args.Pattern, args.MaxMatches)
		if err != nil {
			return "", err
		}
		data, _ := json.MarshalIndent(results, "", "  ")
		return string(data), nil
	case "run_shell":
		var args struct {
			Command string `json:"command"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		if err := a.confirm(reader, fmt.Sprintf("Run shell command? %s", args.Command)); err != nil {
			return "", err
		}
		commandCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		cmd := exec.CommandContext(commandCtx, "sh", "-lc", args.Command)
		cmd.Dir = ws.Root
		out, err := cmd.CombinedOutput()
		if err != nil {
			return string(out), fmt.Errorf("shell command failed: %w", err)
		}
		return string(out), nil
	case "edit_file":
		var args struct {
			Path            string `json:"path"`
			OldText         string `json:"old_text"`
			NewText         string `json:"new_text"`
			CreateIfMissing bool   `json:"create_if_missing"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		if err := a.confirm(reader, fmt.Sprintf("Edit file %s?", args.Path)); err != nil {
			return "", err
		}
		return ws.Replace(args.Path, args.OldText, args.NewText, args.CreateIfMissing)
	case "git_status":
		return gitClient.Status(ctx)
	case "git_diff":
		return gitClient.Diff(ctx)
	case "git_branch":
		var args struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		if args.Name == "" {
			return gitClient.Branch(ctx)
		}
		if err := a.confirm(reader, fmt.Sprintf("Create git branch %s?", args.Name)); err != nil {
			return "", err
		}
		return gitClient.CreateBranch(ctx, args.Name)
	case "git_commit":
		var args struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(call.Arguments, &args)
		if err := a.confirm(reader, fmt.Sprintf("Create git commit? %s", args.Message)); err != nil {
			return "", err
		}
		return gitClient.Commit(ctx, args.Message)
	default:
		return "", fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func (a *App) confirm(reader *bufio.Reader, prompt string) error {
	_, _ = fmt.Fprintf(a.out, "%s [y/N]: ", prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read approval: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		return errors.New("operation rejected")
	}
	return nil
}

func environmentMap() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

func toolDefinitions() []openaiprovider.Tool {
	return []openaiprovider.Tool{
		{Name: "list_files", Description: "List files under the current workspace.", Schema: objectSchema(map[string]any{"max_entries": intSchema("Maximum number of entries to return")}, nil)},
		{Name: "read_file", Description: "Read a text file with line offsets.", Schema: objectSchema(map[string]any{"path": stringSchema("Workspace-relative file path"), "offset": intSchema("1-based line offset"), "limit": intSchema("Maximum number of lines")}, []string{"path"})},
		{Name: "search_files", Description: "Search workspace text files using a regular expression.", Schema: objectSchema(map[string]any{"pattern": stringSchema("Regular expression to search for"), "max_matches": intSchema("Maximum number of matches")}, []string{"pattern"})},
		{Name: "run_shell", Description: "Run a shell command in the workspace after explicit approval.", Schema: objectSchema(map[string]any{"command": stringSchema("Shell command to execute")}, []string{"command"})},
		{Name: "edit_file", Description: "Create or update a file after explicit approval.", Schema: objectSchema(map[string]any{"path": stringSchema("Workspace-relative file path"), "old_text": stringSchema("Existing text to replace. Leave empty to overwrite the whole file."), "new_text": stringSchema("Replacement file content or text fragment"), "create_if_missing": boolSchema("Whether to create a new file if it does not exist")}, []string{"path", "new_text"})},
		{Name: "git_status", Description: "Read git status for the current workspace.", Schema: objectSchema(map[string]any{}, nil)},
		{Name: "git_diff", Description: "Read the current git diff summary.", Schema: objectSchema(map[string]any{}, nil)},
		{Name: "git_branch", Description: "Read the current branch or create a new branch after approval.", Schema: objectSchema(map[string]any{"name": stringSchema("New branch name. Leave empty to read the current branch.")}, nil)},
		{Name: "git_commit", Description: "Create a git commit after approval.", Schema: objectSchema(map[string]any{"message": stringSchema("Commit message")}, []string{"message"})},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func stringSchema(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func intSchema(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func boolSchema(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}
