package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// testERPProvider is a mock ERPProvider for step tests.
type testERPProvider struct {
	readEntity   func(ctx context.Context, entitySet, key string) (map[string]any, error)
	queryEntites func(ctx context.Context, entitySet string, opts QueryOptions) (*QueryResult, error)
	createEntity func(ctx context.Context, entitySet string, data map[string]any) (map[string]any, error)
	updateEntity func(ctx context.Context, entitySet, key string, data map[string]any) error
	deleteEntity func(ctx context.Context, entitySet, key string) error
	batchOp      func(ctx context.Context, ops []BatchOp) ([]BatchResult, error)
	callFunc     func(ctx context.Context, name string, params map[string]any) (map[string]any, error)
	getMetadata  func(ctx context.Context) (string, error)
	rawRequest   func(ctx context.Context, method, path string, body map[string]any, headers map[string]string) (int, map[string]any, error)
}

func (m *testERPProvider) Connect(context.Context, ProviderConfig) error { return nil }
func (m *testERPProvider) Close() error                                  { return nil }

func (m *testERPProvider) ReadEntity(ctx context.Context, es, key string) (map[string]any, error) {
	return m.readEntity(ctx, es, key)
}
func (m *testERPProvider) QueryEntities(ctx context.Context, es string, opts QueryOptions) (*QueryResult, error) {
	return m.queryEntites(ctx, es, opts)
}
func (m *testERPProvider) CreateEntity(ctx context.Context, es string, data map[string]any) (map[string]any, error) {
	return m.createEntity(ctx, es, data)
}
func (m *testERPProvider) UpdateEntity(ctx context.Context, es, key string, data map[string]any) error {
	return m.updateEntity(ctx, es, key, data)
}
func (m *testERPProvider) DeleteEntity(ctx context.Context, es, key string) error {
	return m.deleteEntity(ctx, es, key)
}
func (m *testERPProvider) BatchOperation(ctx context.Context, ops []BatchOp) ([]BatchResult, error) {
	return m.batchOp(ctx, ops)
}
func (m *testERPProvider) CallFunction(ctx context.Context, name string, params map[string]any) (map[string]any, error) {
	return m.callFunc(ctx, name, params)
}
func (m *testERPProvider) GetMetadata(ctx context.Context) (string, error) {
	return m.getMetadata(ctx)
}
func (m *testERPProvider) RawRequest(ctx context.Context, method, path string, body map[string]any, headers map[string]string) (int, map[string]any, error) {
	return m.rawRequest(ctx, method, path, body, headers)
}

func registerTestProvider(t *testing.T, name string, mock ERPProvider) {
	t.Helper()
	providersMu.Lock()
	providers[name] = &erpProvider{name: name, erp: mock}
	providersMu.Unlock()
	t.Cleanup(func() {
		providersMu.Lock()
		delete(providers, name)
		providersMu.Unlock()
	})
}

func execStep(t *testing.T, step sdk.StepInstance, current map[string]any) *sdk.StepResult {
	t.Helper()
	result, err := step.Execute(context.Background(), nil, nil, current, nil, nil)
	if err != nil {
		t.Fatalf("step execute: %v", err)
	}
	return result
}

func TestEntityReadStep(t *testing.T) {
	mock := &testERPProvider{
		readEntity: func(_ context.Context, es, key string) (map[string]any, error) {
			return map[string]any{"ID": key, "EntitySet": es}, nil
		},
	}
	registerTestProvider(t, "test-read", mock)

	step := &entityReadStep{providerName: "test-read"}
	result := execStep(t, step, map[string]any{
		"entity_set": "Orders",
		"key":        "'123'",
	})
	entity := result.Output["entity"].(map[string]any)
	if entity["ID"] != "'123'" {
		t.Errorf("unexpected ID: %v", entity["ID"])
	}
}

func TestEntityReadStep_MissingFields(t *testing.T) {
	mock := &testERPProvider{}
	registerTestProvider(t, "test-read-err", mock)

	step := &entityReadStep{providerName: "test-read-err"}
	_, err := step.Execute(context.Background(), nil, nil, map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing fields")
	}
}

func TestEntityQueryStep(t *testing.T) {
	mock := &testERPProvider{
		queryEntites: func(_ context.Context, es string, opts QueryOptions) (*QueryResult, error) {
			return &QueryResult{
				Results:  []map[string]any{{"ID": "1"}, {"ID": "2"}},
				Count:    2,
				NextLink: "next-page",
			}, nil
		},
	}
	registerTestProvider(t, "test-query", mock)

	step := &entityQueryStep{providerName: "test-query"}
	result := execStep(t, step, map[string]any{
		"entity_set": "Products",
		"filter":     "Price gt 100",
		"top":        float64(10),
	})
	results := result.Output["results"].([]map[string]any)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if result.Output["count"] != 2 {
		t.Errorf("expected count 2, got %v", result.Output["count"])
	}
}

func TestEntityCreateStep(t *testing.T) {
	mock := &testERPProvider{
		createEntity: func(_ context.Context, es string, data map[string]any) (map[string]any, error) {
			data["ID"] = "NEW"
			return data, nil
		},
	}
	registerTestProvider(t, "test-create", mock)

	step := &entityCreateStep{providerName: "test-create"}
	result := execStep(t, step, map[string]any{
		"entity_set": "Orders",
		"data":       map[string]any{"CustomerID": "C1"},
	})
	entity := result.Output["entity"].(map[string]any)
	if entity["ID"] != "NEW" {
		t.Errorf("unexpected ID: %v", entity["ID"])
	}
}

func TestEntityUpdateStep(t *testing.T) {
	called := false
	mock := &testERPProvider{
		updateEntity: func(_ context.Context, es, key string, data map[string]any) error {
			called = true
			return nil
		},
	}
	registerTestProvider(t, "test-update", mock)

	step := &entityUpdateStep{providerName: "test-update"}
	result := execStep(t, step, map[string]any{
		"entity_set": "Orders",
		"key":        "'1'",
		"data":       map[string]any{"Status": "Done"},
	})
	if !called {
		t.Error("update was not called")
	}
	if result.Output["ok"] != true {
		t.Errorf("expected ok=true")
	}
}

func TestEntityDeleteStep(t *testing.T) {
	called := false
	mock := &testERPProvider{
		deleteEntity: func(_ context.Context, es, key string) error {
			called = true
			return nil
		},
	}
	registerTestProvider(t, "test-delete", mock)

	step := &entityDeleteStep{providerName: "test-delete"}
	result := execStep(t, step, map[string]any{
		"entity_set": "Orders",
		"key":        "'1'",
	})
	if !called {
		t.Error("delete was not called")
	}
	if result.Output["ok"] != true {
		t.Errorf("expected ok=true")
	}
}

func TestBatchStep(t *testing.T) {
	mock := &testERPProvider{
		batchOp: func(_ context.Context, ops []BatchOp) ([]BatchResult, error) {
			results := make([]BatchResult, len(ops))
			for i, op := range ops {
				results[i] = BatchResult{ContentID: op.ContentID, StatusCode: 200}
			}
			return results, nil
		},
	}
	registerTestProvider(t, "test-batch", mock)

	step := &batchStep{providerName: "test-batch"}
	result := execStep(t, step, map[string]any{
		"operations": []any{
			map[string]any{"method": "GET", "entity_set": "Orders", "key": "'1'", "content_id": "op1"},
			map[string]any{"method": "DELETE", "entity_set": "Orders", "key": "'2'", "content_id": "op2"},
		},
	})
	results := result.Output["results"].([]map[string]any)
	if len(results) != 2 {
		t.Errorf("expected 2 batch results, got %d", len(results))
	}
}

func TestFunctionCallStep(t *testing.T) {
	mock := &testERPProvider{
		callFunc: func(_ context.Context, name string, params map[string]any) (map[string]any, error) {
			return map[string]any{"price": 42.5, "function": name}, nil
		},
	}
	registerTestProvider(t, "test-func", mock)

	step := &functionCallStep{providerName: "test-func"}
	result := execStep(t, step, map[string]any{
		"function_name": "CalculatePrice",
		"params":        map[string]any{"product": "P1"},
	})
	fnResult := result.Output["result"].(map[string]any)
	if fnResult["price"] != 42.5 {
		t.Errorf("unexpected price: %v", fnResult["price"])
	}
}

func TestMetadataStep(t *testing.T) {
	mock := &testERPProvider{
		getMetadata: func(_ context.Context) (string, error) {
			return "<edmx:Edmx/>", nil
		},
	}
	registerTestProvider(t, "test-meta", mock)

	step := &metadataStep{providerName: "test-meta"}
	result := execStep(t, step, nil)
	if result.Output["metadata"] != "<edmx:Edmx/>" {
		t.Errorf("unexpected metadata: %v", result.Output["metadata"])
	}
}

func TestRawRequestStep(t *testing.T) {
	mock := &testERPProvider{
		rawRequest: func(_ context.Context, method, path string, body map[string]any, headers map[string]string) (int, map[string]any, error) {
			return 200, map[string]any{"raw": true, "method": method}, nil
		},
	}
	registerTestProvider(t, "test-raw", mock)

	step := &rawRequestStep{providerName: "test-raw"}
	result := execStep(t, step, map[string]any{
		"method":  "POST",
		"path":    "/custom/endpoint",
		"body":    map[string]any{"key": "val"},
		"headers": map[string]any{"X-Custom": "test"},
	})
	if result.Output["status_code"] != 200 {
		t.Errorf("unexpected status: %v", result.Output["status_code"])
	}
}

func TestStepWithMissingProvider(t *testing.T) {
	step := &entityReadStep{providerName: "nonexistent-provider"}
	_, err := step.Execute(context.Background(), nil, nil, map[string]any{
		"entity_set": "X",
		"key":        "Y",
	}, nil, nil)
	if err == nil {
		t.Error("expected error for missing provider")
	}
}

func TestProviderIntegration_InitAndLookup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ID": "1"})
	}))
	defer srv.Close()

	p := newERPProvider("integration-test", map[string]any{
		"baseUrl":  srv.URL,
		"authType": "basic",
		"username": "u",
		"password": "p",
	})

	if err := p.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer p.Stop(context.Background())

	found, err := getProvider("integration-test")
	if err != nil {
		t.Fatalf("getProvider: %v", err)
	}
	if found.name != "integration-test" {
		t.Errorf("unexpected name: %s", found.name)
	}

	entity, err := found.erp.ReadEntity(context.Background(), "Items", "'1'")
	if err != nil {
		t.Fatalf("ReadEntity: %v", err)
	}
	if entity["ID"] != "1" {
		t.Errorf("unexpected ID: %v", entity["ID"])
	}
}

func TestProviderConcurrency(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"value": []map[string]any{{"ID": "1"}}})
	}))
	defer srv.Close()

	p := newERPProvider("concurrent-test", map[string]any{
		"baseUrl":  srv.URL,
		"authType": "basic",
		"username": "u",
		"password": "p",
	})
	if err := p.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer p.Stop(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			prov, err := getProvider("concurrent-test")
			if err != nil {
				t.Errorf("getProvider: %v", err)
				return
			}
			_, err = prov.erp.QueryEntities(context.Background(), "Items", QueryOptions{})
			if err != nil {
				t.Errorf("QueryEntities: %v", err)
			}
		}()
	}
	wg.Wait()
}
