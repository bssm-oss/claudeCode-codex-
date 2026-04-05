# CI policy

CI must run on every push and pull request. The baseline checks are:

- `go test ./...`
- `go build ./...`
- `gofmt -w` cleanliness check through `gofmt -l .`
- linting through `golangci-lint`

CI is expected to validate the same merged branch that users consume. A PR is not considered complete if it is merged while the primary CI workflow is red.

If CI fails, the branch is not ready for merge.
