# CI policy

CI must run on every push and pull request. The baseline checks are:

- `go test ./...`
- `go build ./...`
- `gofmt -w` cleanliness check through `gofmt -l .`
- linting through `golangci-lint`

CI is expected to validate the same merged branch that users consume. The repository is considered release-ready only when the current `main` branch is green. If a PR is merged while the primary CI workflow is red, a follow-up fix and change note are required before the work can be considered complete.

If CI fails, the branch is not ready for merge.
