package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// QueryOptions represents OData v4 system query options.
type QueryOptions struct {
	Filter  string
	Select  string
	Expand  string
	OrderBy string
	Top     int
	Skip    int
}

// ODataResponse is the envelope returned by OData v4 collection queries.
type ODataResponse struct {
	Value    []map[string]any `json:"value"`
	Count    int              `json:"count"`
	NextLink string           `json:"nextLink"`
}

// BatchRequest represents a single request in an OData $batch payload.
type BatchRequest struct {
	Method    string
	URL       string
	Body      map[string]any
	ContentID string
}

// BatchResponse represents a single response from an OData $batch.
type BatchResponse struct {
	ContentID  string
	StatusCode int
	Body       map[string]any
}

// AuthHeaderFunc provides authorization headers for requests.
type AuthHeaderFunc func(req *http.Request) error

// ODataClient is a generic OData v4 HTTP client.
type ODataClient struct {
	baseURL    string
	httpClient *http.Client
	authFunc   AuthHeaderFunc
	csrfToken  string
}

// NewODataClient creates a new OData v4 client.
func NewODataClient(baseURL string, httpClient *http.Client, authFunc AuthHeaderFunc) *ODataClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &ODataClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		authFunc:   authFunc,
	}
}

func (c *ODataClient) buildURL(entitySet string, opts *QueryOptions) string {
	u := c.baseURL + "/" + entitySet
	if opts == nil {
		return u
	}
	params := url.Values{}
	if opts.Filter != "" {
		params.Set("$filter", opts.Filter)
	}
	if opts.Select != "" {
		params.Set("$select", opts.Select)
	}
	if opts.Expand != "" {
		params.Set("$expand", opts.Expand)
	}
	if opts.OrderBy != "" {
		params.Set("$orderby", opts.OrderBy)
	}
	if opts.Top > 0 {
		params.Set("$top", strconv.Itoa(opts.Top))
	}
	if opts.Skip > 0 {
		params.Set("$skip", strconv.Itoa(opts.Skip))
	}
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return u
}

func (c *ODataClient) newRequest(ctx context.Context, method, rawURL string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, rawURL, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.authFunc != nil {
		if err := c.authFunc(req); err != nil {
			return nil, fmt.Errorf("auth: %w", err)
		}
	}
	return req, nil
}

func (c *ODataClient) do(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("odata: HTTP %d: %s", resp.StatusCode, string(errBody))
	}
	return resp, nil
}

func decodeJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

// Get queries an entity set with OData options.
func (c *ODataClient) Get(ctx context.Context, entitySet string, opts *QueryOptions) (*ODataResponse, error) {
	u := c.buildURL(entitySet, opts)
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Value    []map[string]any `json:"value"`
		Count    *int             `json:"@odata.count"`
		NextLink string           `json:"@odata.nextLink"`
	}
	if err := decodeJSON(resp.Body, &raw); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	result := &ODataResponse{
		Value:    raw.Value,
		NextLink: raw.NextLink,
	}
	if raw.Count != nil {
		result.Count = *raw.Count
	}
	return result, nil
}

// GetByKey retrieves a single entity by key.
func (c *ODataClient) GetByKey(ctx context.Context, entitySet, key string) (map[string]any, error) {
	u := c.baseURL + "/" + entitySet + "(" + key + ")"
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var entity map[string]any
	if err := decodeJSON(resp.Body, &entity); err != nil {
		return nil, fmt.Errorf("decode entity: %w", err)
	}
	return entity, nil
}

// Create posts a new entity to the entity set.
func (c *ODataClient) Create(ctx context.Context, entitySet string, entity map[string]any) (map[string]any, error) {
	u := c.baseURL + "/" + entitySet
	req, err := c.newRequest(ctx, http.MethodPost, u, entity)
	if err != nil {
		return nil, err
	}
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var created map[string]any
	if err := decodeJSON(resp.Body, &created); err != nil {
		return nil, fmt.Errorf("decode created: %w", err)
	}
	return created, nil
}

// Update patches an existing entity.
func (c *ODataClient) Update(ctx context.Context, entitySet, key string, entity map[string]any) error {
	u := c.baseURL + "/" + entitySet + "(" + key + ")"
	req, err := c.newRequest(ctx, http.MethodPatch, u, entity)
	if err != nil {
		return err
	}
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Delete removes an entity by key.
func (c *ODataClient) Delete(ctx context.Context, entitySet, key string) error {
	u := c.baseURL + "/" + entitySet + "(" + key + ")"
	req, err := c.newRequest(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Batch executes an OData $batch request with JSON format.
func (c *ODataClient) Batch(ctx context.Context, requests []BatchRequest) ([]BatchResponse, error) {
	type batchReqPayload struct {
		ID     string         `json:"id"`
		Method string         `json:"method"`
		URL    string         `json:"url"`
		Body   map[string]any `json:"body,omitempty"`
	}
	type batchEnvelope struct {
		Requests []batchReqPayload `json:"requests"`
	}

	payload := batchEnvelope{Requests: make([]batchReqPayload, len(requests))}
	for i, r := range requests {
		cid := r.ContentID
		if cid == "" {
			cid = strconv.Itoa(i)
		}
		payload.Requests[i] = batchReqPayload{
			ID:     cid,
			Method: r.Method,
			URL:    r.URL,
			Body:   r.Body,
		}
	}

	u := c.baseURL + "/$batch"
	req, err := c.newRequest(ctx, http.MethodPost, u, payload)
	if err != nil {
		return nil, err
	}
	if c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	type batchRespPayload struct {
		ID     string         `json:"id"`
		Status int            `json:"status"`
		Body   map[string]any `json:"body"`
	}
	type batchRespEnvelope struct {
		Responses []batchRespPayload `json:"responses"`
	}

	var batchResp batchRespEnvelope
	if err := decodeJSON(resp.Body, &batchResp); err != nil {
		return nil, fmt.Errorf("decode batch response: %w", err)
	}

	results := make([]BatchResponse, len(batchResp.Responses))
	for i, r := range batchResp.Responses {
		results[i] = BatchResponse{
			ContentID:  r.ID,
			StatusCode: r.Status,
			Body:       r.Body,
		}
	}
	return results, nil
}

// CallFunction invokes an OData function import.
func (c *ODataClient) CallFunction(ctx context.Context, name string, params map[string]any) (map[string]any, error) {
	parts := make([]string, 0, len(params))
	for k, v := range params {
		switch val := v.(type) {
		case string:
			escaped := strings.ReplaceAll(val, "'", "''")
			parts = append(parts, fmt.Sprintf("%s='%s'", k, url.PathEscape(escaped)))
		default:
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	paramStr := ""
	if len(parts) > 0 {
		paramStr = "(" + strings.Join(parts, ",") + ")"
	}
	u := c.baseURL + "/" + name + paramStr
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]any
	if err := decodeJSON(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("decode function result: %w", err)
	}
	return result, nil
}

// GetMetadata retrieves the OData $metadata document.
func (c *ODataClient) GetMetadata(ctx context.Context) (string, error) {
	u := c.baseURL + "/$metadata"
	req, err := c.newRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/xml")
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RawRequest executes an arbitrary HTTP request against the OData service.
func (c *ODataClient) RawRequest(ctx context.Context, method, path string, body map[string]any, headers map[string]string) (int, map[string]any, error) {
	u := c.baseURL + "/" + strings.TrimLeft(path, "/")

	var isMutating bool
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		isMutating = true
	}

	req, err := c.newRequest(ctx, method, u, body)
	if err != nil {
		return 0, nil, err
	}
	if isMutating && c.csrfToken != "" {
		req.Header.Set("X-CSRF-Token", c.csrfToken)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	var result map[string]any
	if len(respBody) > 0 {
		_ = json.Unmarshal(respBody, &result)
	}
	return resp.StatusCode, result, nil
}

// SetCSRFToken sets the CSRF token for mutating requests.
func (c *ODataClient) SetCSRFToken(token string) {
	c.csrfToken = token
}
