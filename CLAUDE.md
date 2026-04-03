# CLAUDE.md — Workflow Plugin Template

External gRPC plugin for the GoCodeAlone/workflow engine.

## Build & Test

```sh
go build ./...
go test ./... -v -race -count=1
```

## Cross-compile for deployment

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o workflow-plugin-TEMPLATE ./cmd/workflow-plugin-TEMPLATE/
```

## Structure

- `cmd/workflow-plugin-TEMPLATE/main.go` — Plugin entry point (calls `sdk.Serve`)
- `internal/plugin.go` — Plugin manifest, module factories, step factories
- `internal/` — All module and step implementations
- `plugin.json` — Capability manifest for the workflow registry
- `.goreleaser.yaml` — GoReleaser v2 config for cross-platform releases
- `.github/workflows/ci.yml` — CI on push/PR (build + test)
- `.github/workflows/release.yml` — Release on v* tag push (GoReleaser)

## Adding a Module Type

1. Create `internal/module_example.go` implementing the module
2. Register in `internal/plugin.go` ModuleFactories()
3. Add to `plugin.json` capabilities.moduleTypes
4. Add tests in `internal/module_example_test.go`

## Adding a Step Type

1. Create `internal/step_example.go` implementing the step
2. Register in `internal/plugin.go` StepFactories()
3. Add to `plugin.json` capabilities.stepTypes
4. Add tests in `internal/step_example_test.go`

## Releasing

```sh
git tag v0.1.0
git push origin v0.1.0
```
GoReleaser builds cross-platform binaries and creates a GitHub Release automatically.
