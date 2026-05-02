package internal

import (
	"context"
	"fmt"
	"os"
	"sync"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

var (
	providersMu sync.RWMutex
	providers   = map[string]*erpProvider{}
)

func getProvider(name string) (*erpProvider, error) {
	providersMu.RLock()
	defer providersMu.RUnlock()
	p, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("erp provider %q not found", name)
	}
	return p, nil
}

// erpProvider is the erp.provider module instance.
type erpProvider struct {
	name   string
	config ProviderConfig
	erp    ERPProvider
}

func newERPProvider(name string, cfg map[string]any) *erpProvider {
	return &erpProvider{
		name: name,
		config: ProviderConfig{
			BaseURL:      expandEnvStr(cfg, "baseUrl"),
			AuthType:     strOr(cfg, "authType", "basic"),
			Username:     expandEnvStr(cfg, "username"),
			Password:     expandEnvStr(cfg, "password"),
			ClientID:     expandEnvStr(cfg, "clientId"),
			ClientSecret: expandEnvStr(cfg, "clientSecret"),
			TokenURL:     expandEnvStr(cfg, "tokenUrl"),
			APIKey:       expandEnvStr(cfg, "apiKey"),
			FetchCSRF:    boolFrom(cfg, "fetchCsrf"),
		},
	}
}

func (p *erpProvider) Init() error {
	if p.config.BaseURL == "" {
		return fmt.Errorf("erp provider %q: baseUrl is required", p.name)
	}

	adapter := NewSAPAdapter()
	if err := adapter.Connect(context.Background(), p.config); err != nil {
		return fmt.Errorf("erp provider %q: connect: %w", p.name, err)
	}
	p.erp = adapter

	providersMu.Lock()
	providers[p.name] = p
	providersMu.Unlock()
	return nil
}

func (p *erpProvider) Start(_ context.Context) error { return nil }

func (p *erpProvider) Stop(_ context.Context) error {
	providersMu.Lock()
	delete(providers, p.name)
	providersMu.Unlock()
	if p.erp != nil {
		return p.erp.Close()
	}
	return nil
}

// Ensure interface compliance.
var _ sdk.ModuleInstance = (*erpProvider)(nil)

// Config helpers

func stringFrom(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func expandEnvStr(m map[string]any, key string) string {
	return os.ExpandEnv(stringFrom(m, key))
}

func strOr(m map[string]any, key, fallback string) string {
	v := stringFrom(m, key)
	if v == "" {
		return fallback
	}
	return v
}

func boolFrom(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func intFrom(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json_number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
}

type json_number interface {
	Int64() (int64, error)
}

func mapFrom(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}

func sliceFrom(m map[string]any, key string) []any {
	v, _ := m[key].([]any)
	return v
}

func stringMapFrom(m map[string]any, key string) map[string]string {
	raw, _ := m[key].(map[string]any)
	if raw == nil {
		return nil
	}
	result := make(map[string]string, len(raw))
	for k, v := range raw {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

// strFromKeys returns the first non-empty string value found for any of the
// provided keys. It is used to accept both legacy snake_case keys (e.g.
// "entity_set") and the camelCase equivalents produced by proto JSON encoding
// (e.g. "entitySet") without breaking existing callers. The keys are tried in
// order; the first non-empty match wins. When only one key is set, the result
// is unambiguous. When both are set, the first key takes precedence.
func strFromKeys(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, _ := m[k].(string); v != "" {
			return v
		}
	}
	return ""
}

// Key-name constants for proto JSON (camelCase) and legacy (snake_case) forms.
// These are used with strFromKeys to accept either form at step input boundaries.
const (
	keyEntitySet      = "entity_set"
	keyEntitySetCC    = "entitySet"
	keyContentID      = "content_id"
	keyContentIDCC    = "contentId"
	keyFunctionName   = "function_name"
	keyFunctionNameCC = "functionName"
)
