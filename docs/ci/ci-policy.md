# CI policy

CI must run on every push and pull request. The baseline checks are:

- `go test ./...`
- `go build ./...`
- `gofmt -w` cleanliness check through `gofmt -l .`
- linting through `golangci-lint`

If CI fails, the branch is not ready for merge.
