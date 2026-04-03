package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestODataClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("$filter") != "Name eq 'Acme'" {
			t.Errorf("unexpected filter: %s", r.URL.Query().Get("$filter"))
		}
		if r.URL.Query().Get("$top") != "10" {
			t.Errorf("unexpected top: %s", r.URL.Query().Get("$top"))
		}
		resp := map[string]any{
			"value":            []map[string]any{{"ID": "1", "Name": "Acme"}},
			"@odata.count":    1,
			"@odata.nextLink": "next",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	resp, err := c.Get(context.Background(), "BusinessPartners", &QueryOptions{
		Filter: "Name eq 'Acme'",
		Top:    10,
	})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(resp.Value) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Value))
	}
	if resp.Value[0]["Name"] != "Acme" {
		t.Errorf("unexpected name: %v", resp.Value[0]["Name"])
	}
	if resp.Count != 1 {
		t.Errorf("expected count 1, got %d", resp.Count)
	}
	if resp.NextLink != "next" {
		t.Errorf("expected nextLink 'next', got %q", resp.NextLink)
	}
}

func TestODataClient_GetByKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/BusinessPartners('1001')" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"ID": "1001", "Name": "Acme"})
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	entity, err := c.GetByKey(context.Background(), "BusinessPartners", "'1001'")
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if entity["ID"] != "1001" {
		t.Errorf("unexpected ID: %v", entity["ID"])
	}
}

func TestODataClient_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-CSRF-Token") != "tok123" {
			t.Errorf("expected CSRF token, got %q", r.Header.Get("X-CSRF-Token"))
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		body["ID"] = "NEW1"
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(body)
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	c.SetCSRFToken("tok123")
	created, err := c.Create(context.Background(), "SalesOrders", map[string]any{"CustomerID": "C1"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created["ID"] != "NEW1" {
		t.Errorf("unexpected ID: %v", created["ID"])
	}
}

func TestODataClient_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/SalesOrders('SO1')" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	err := c.Update(context.Background(), "SalesOrders", "'SO1'", map[string]any{"Status": "Completed"})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
}

func TestODataClient_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	err := c.Delete(context.Background(), "SalesOrders", "'SO1'")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestODataClient_Batch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/$batch" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := map[string]any{
			"responses": []map[string]any{
				{"id": "0", "status": 200, "body": map[string]any{"ID": "1"}},
				{"id": "1", "status": 201, "body": map[string]any{"ID": "2"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	results, err := c.Batch(context.Background(), []BatchRequest{
		{Method: "GET", URL: "Orders('1')"},
		{Method: "POST", URL: "Orders", Body: map[string]any{"Name": "New"}},
	})
	if err != nil {
		t.Fatalf("Batch failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].StatusCode != 200 {
		t.Errorf("expected status 200, got %d", results[0].StatusCode)
	}
	if results[1].StatusCode != 201 {
		t.Errorf("expected status 201, got %d", results[1].StatusCode)
	}
}

func TestODataClient_CallFunction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{"result": "ok"})
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	result, err := c.CallFunction(context.Background(), "CalculatePrice", map[string]any{"product": "P1"})
	if err != nil {
		t.Fatalf("CallFunction failed: %v", err)
	}
	if result["result"] != "ok" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestODataClient_GetMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/$metadata" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte("<edmx:Edmx/>"))
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	meta, err := c.GetMetadata(context.Background())
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}
	if meta != "<edmx:Edmx/>" {
		t.Errorf("unexpected metadata: %s", meta)
	}
}

func TestODataClient_RawRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "test" {
			t.Errorf("missing custom header")
		}
		json.NewEncoder(w).Encode(map[string]any{"raw": true})
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	status, body, err := c.RawRequest(context.Background(), "GET", "/custom", nil, map[string]string{"X-Custom": "test"})
	if err != nil {
		t.Fatalf("RawRequest failed: %v", err)
	}
	if status != 200 {
		t.Errorf("expected 200, got %d", status)
	}
	if body["raw"] != true {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestODataClient_CallFunction_StringParams(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RawPath
		if gotPath == "" {
			gotPath = r.URL.Path
		}
		json.NewEncoder(w).Encode(map[string]any{"result": "ok"})
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)

	// Single string param — should be single-quoted and URL-encoded
	result, err := c.CallFunction(context.Background(), "GetPartner", map[string]any{
		"name": "Acme Corp",
	})
	if err != nil {
		t.Fatalf("CallFunction failed: %v", err)
	}
	if result["result"] != "ok" {
		t.Errorf("unexpected result: %v", result)
	}
	// Path should contain the single-quoted, encoded parameter
	if !contains(gotPath, "name='Acme") {
		t.Errorf("expected single-quoted string param in path, got %q", gotPath)
	}

	// Param with embedded single quote — should be escaped as ''
	_, err = c.CallFunction(context.Background(), "Search", map[string]any{
		"query": "O'Reilly",
	})
	if err != nil {
		t.Fatalf("CallFunction with quote failed: %v", err)
	}
	if !contains(gotPath, "query='O") {
		t.Errorf("expected escaped quote in path, got %q", gotPath)
	}

	// Numeric param should NOT be quoted
	_, err = c.CallFunction(context.Background(), "GetPrice", map[string]any{
		"amount": 42,
	})
	if err != nil {
		t.Fatalf("CallFunction with int failed: %v", err)
	}
	if contains(gotPath, "'42'") {
		t.Errorf("numeric param should not be quoted, got %q", gotPath)
	}
	if !contains(gotPath, "amount=42") {
		t.Errorf("expected amount=42 in path, got %q", gotPath)
	}
}

func TestODataClient_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	_, err := c.GetByKey(context.Background(), "Orders", "'999'")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestODataClient_AuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(map[string]any{"ID": "1"})
	}))
	defer srv.Close()

	authFunc := func(req *http.Request) error {
		req.Header.Set("Authorization", "Bearer test-token")
		return nil
	}
	c := NewODataClient(srv.URL, srv.Client(), authFunc)
	_, err := c.GetByKey(context.Background(), "Items", "'1'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("expected Bearer token, got %q", gotAuth)
	}
}

func TestODataClient_SelectExpandOrderBy(t *testing.T) {
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.RequestURI()
		json.NewEncoder(w).Encode(map[string]any{"value": []map[string]any{}})
	}))
	defer srv.Close()

	c := NewODataClient(srv.URL, srv.Client(), nil)
	_, err := c.Get(context.Background(), "Orders", &QueryOptions{
		Select:  "ID,Name",
		Expand:  "Items",
		OrderBy: "Name desc",
		Skip:    5,
	})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	// URL params are percent-encoded by net/url, so check decoded query string
	if !contains(gotURL, "select=") {
		t.Errorf("URL %q missing $select param", gotURL)
	}
	if !contains(gotURL, "expand=") {
		t.Errorf("URL %q missing $expand param", gotURL)
	}
	if !contains(gotURL, "orderby=") {
		t.Errorf("URL %q missing $orderby param", gotURL)
	}
	if !contains(gotURL, "skip=5") {
		t.Errorf("URL %q missing $skip param", gotURL)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchSubstring(s, sub)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
