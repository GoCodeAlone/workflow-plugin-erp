package internal

import (
	"context"
	"fmt"
	"net/http"
)

// ProviderConfig holds the configuration needed to connect to an ERP backend.
type ProviderConfig struct {
	BaseURL      string
	AuthType     string // basic, oauth2, apikey
	Username     string
	Password     string
	ClientID     string
	ClientSecret string
	TokenURL     string
	APIKey       string
	FetchCSRF    bool
}

// QueryResult is the return type for entity queries.
type QueryResult struct {
	Results  []map[string]any
	Count    int
	NextLink string
}

// BatchOp describes a single operation within a batch request.
type BatchOp struct {
	Method    string
	EntitySet string
	Key       string
	Body      map[string]any
	ContentID string
}

// BatchResult is the result of a single batch operation.
type BatchResult struct {
	ContentID  string
	StatusCode int
	Body       map[string]any
}

// ERPProvider defines the interface for ERP backend interactions.
type ERPProvider interface {
	Connect(ctx context.Context, config ProviderConfig) error
	ReadEntity(ctx context.Context, entitySet string, key string) (map[string]any, error)
	QueryEntities(ctx context.Context, entitySet string, opts QueryOptions) (*QueryResult, error)
	CreateEntity(ctx context.Context, entitySet string, data map[string]any) (map[string]any, error)
	UpdateEntity(ctx context.Context, entitySet, key string, data map[string]any) error
	DeleteEntity(ctx context.Context, entitySet, key string) error
	BatchOperation(ctx context.Context, ops []BatchOp) ([]BatchResult, error)
	CallFunction(ctx context.Context, name string, params map[string]any) (map[string]any, error)
	GetMetadata(ctx context.Context) (string, error)
	RawRequest(ctx context.Context, method, path string, body map[string]any, headers map[string]string) (int, map[string]any, error)
	Close() error
}

// SAPAdapter implements ERPProvider for SAP S/4HANA via OData v4.
type SAPAdapter struct {
	auth   *SAPAuth
	client *ODataClient
}

// NewSAPAdapter creates a new unconnected SAP adapter.
func NewSAPAdapter() *SAPAdapter {
	return &SAPAdapter{}
}

func (a *SAPAdapter) Connect(ctx context.Context, config ProviderConfig) error {
	authCfg := SAPAuthConfig{
		AuthType:     config.AuthType,
		Username:     config.Username,
		Password:     config.Password,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     config.TokenURL,
		APIKey:       config.APIKey,
	}
	httpClient := &http.Client{Timeout: 0}
	a.auth = NewSAPAuth(config.BaseURL, authCfg, httpClient)
	a.client = NewODataClient(config.BaseURL, httpClient, a.auth.AuthHeader())

	if config.FetchCSRF {
		token, err := a.auth.FetchCSRFToken(ctx)
		if err != nil {
			return fmt.Errorf("fetch csrf: %w", err)
		}
		a.client.SetCSRFToken(token)
	}
	return nil
}

func (a *SAPAdapter) ReadEntity(ctx context.Context, entitySet string, key string) (map[string]any, error) {
	return a.client.GetByKey(ctx, entitySet, key)
}

func (a *SAPAdapter) QueryEntities(ctx context.Context, entitySet string, opts QueryOptions) (*QueryResult, error) {
	resp, err := a.client.Get(ctx, entitySet, &opts)
	if err != nil {
		return nil, err
	}
	return &QueryResult{
		Results:  resp.Value,
		Count:    resp.Count,
		NextLink: resp.NextLink,
	}, nil
}

func (a *SAPAdapter) CreateEntity(ctx context.Context, entitySet string, data map[string]any) (map[string]any, error) {
	return a.client.Create(ctx, entitySet, data)
}

func (a *SAPAdapter) UpdateEntity(ctx context.Context, entitySet, key string, data map[string]any) error {
	return a.client.Update(ctx, entitySet, key, data)
}

func (a *SAPAdapter) DeleteEntity(ctx context.Context, entitySet, key string) error {
	return a.client.Delete(ctx, entitySet, key)
}

func (a *SAPAdapter) BatchOperation(ctx context.Context, ops []BatchOp) ([]BatchResult, error) {
	requests := make([]BatchRequest, len(ops))
	for i, op := range ops {
		u := op.EntitySet
		if op.Key != "" {
			u += "(" + op.Key + ")"
		}
		requests[i] = BatchRequest{
			Method:    op.Method,
			URL:       u,
			Body:      op.Body,
			ContentID: op.ContentID,
		}
	}
	responses, err := a.client.Batch(ctx, requests)
	if err != nil {
		return nil, err
	}
	results := make([]BatchResult, len(responses))
	for i, r := range responses {
		results[i] = BatchResult{
			ContentID:  r.ContentID,
			StatusCode: r.StatusCode,
			Body:       r.Body,
		}
	}
	return results, nil
}

func (a *SAPAdapter) CallFunction(ctx context.Context, name string, params map[string]any) (map[string]any, error) {
	return a.client.CallFunction(ctx, name, params)
}

func (a *SAPAdapter) GetMetadata(ctx context.Context) (string, error) {
	return a.client.GetMetadata(ctx)
}

func (a *SAPAdapter) RawRequest(ctx context.Context, method, path string, body map[string]any, headers map[string]string) (int, map[string]any, error) {
	return a.client.RawRequest(ctx, method, path, body, headers)
}

func (a *SAPAdapter) Close() error {
	return nil
}
