# CLAUDE.md — workflow-plugin-erp

Enterprise ERP integration (SAP S/4HANA via OData v4) for the GoCodeAlone/workflow engine.

## Build & Test

```sh
go build ./...
go test ./... -v -race -count=1
```

## Cross-compile for deployment

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o workflow-plugin-erp ./cmd/workflow-plugin-erp/
```

## Structure

- `cmd/workflow-plugin-erp/main.go` — Plugin entry point (calls `sdk.Serve`)
- `internal/plugin.go` — Plugin manifest, module/step registration
- `internal/odata_client.go` — Generic OData v4 HTTP client
- `internal/sap_auth.go` — SAP CSRF token + OAuth2 auth
- `internal/sap_adapter.go` — SAP adapter implementing ERPProvider interface
- `internal/provider.go` — erp.provider module (global provider registry)
- `internal/steps.go` — All 9 step type implementations
- `plugin.json` — Capability manifest for the workflow registry

## Module: erp.provider

Config: baseUrl, authType (basic|oauth2|apikey), username, password, clientId, clientSecret, tokenUrl, apiKey

## Steps

step.erp_entity_read, step.erp_entity_query, step.erp_entity_create, step.erp_entity_update,
step.erp_entity_delete, step.erp_batch, step.erp_function_call, step.erp_metadata, step.erp_raw_request

## Releasing

```sh
git tag v0.1.0
git push origin v0.1.0
```
