package internal

import (
	"context"
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// entityReadStep implements step.erp_entity_read
type entityReadStep struct{ providerName string }

func (s *entityReadStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	p, err := getProvider(s.providerName)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": err.Error()}}, nil
	}
	entitySet := stringFrom(current, "entity_set")
	key := stringFrom(current, "key")
	if entitySet == "" || key == "" {
		return &sdk.StepResult{Output: map[string]any{"error": "entity_set and key are required"}}, nil
	}
	entity, err := p.erp.ReadEntity(ctx, entitySet, key)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("read entity: %v", err)}}, nil
	}
	return &sdk.StepResult{Output: map[string]any{"entity": entity}}, nil
}

// entityQueryStep implements step.erp_entity_query
type entityQueryStep struct{ providerName string }

func (s *entityQueryStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	p, err := getProvider(s.providerName)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": err.Error()}}, nil
	}
	entitySet := stringFrom(current, "entity_set")
	if entitySet == "" {
		return &sdk.StepResult{Output: map[string]any{"error": "entity_set is required"}}, nil
	}
	opts := QueryOptions{
		Filter:  stringFrom(current, "filter"),
		Select:  stringFrom(current, "select"),
		Expand:  stringFrom(current, "expand"),
		OrderBy: stringFrom(current, "orderby"),
		Top:     intFrom(current, "top"),
		Skip:    intFrom(current, "skip"),
	}
	result, err := p.erp.QueryEntities(ctx, entitySet, opts)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("query entities: %v", err)}}, nil
	}
	out := map[string]any{
		"results":  result.Results,
		"count":    result.Count,
		"nextLink": result.NextLink,
	}
	return &sdk.StepResult{Output: out}, nil
}

// entityCreateStep implements step.erp_entity_create
type entityCreateStep struct{ providerName string }

func (s *entityCreateStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	p, err := getProvider(s.providerName)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": err.Error()}}, nil
	}
	entitySet := stringFrom(current, "entity_set")
	data := mapFrom(current, "data")
	if entitySet == "" {
		return &sdk.StepResult{Output: map[string]any{"error": "entity_set is required"}}, nil
	}
	if data == nil {
		return &sdk.StepResult{Output: map[string]any{"error": "data is required"}}, nil
	}
	created, err := p.erp.CreateEntity(ctx, entitySet, data)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("create entity: %v", err)}}, nil
	}
	return &sdk.StepResult{Output: map[string]any{"entity": created}}, nil
}

// entityUpdateStep implements step.erp_entity_update
type entityUpdateStep struct{ providerName string }

func (s *entityUpdateStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	p, err := getProvider(s.providerName)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": err.Error()}}, nil
	}
	entitySet := stringFrom(current, "entity_set")
	key := stringFrom(current, "key")
	data := mapFrom(current, "data")
	if entitySet == "" || key == "" {
		return &sdk.StepResult{Output: map[string]any{"error": "entity_set and key are required"}}, nil
	}
	if data == nil {
		return &sdk.StepResult{Output: map[string]any{"error": "data is required"}}, nil
	}
	if err := p.erp.UpdateEntity(ctx, entitySet, key, data); err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("update entity: %v", err)}}, nil
	}
	return &sdk.StepResult{Output: map[string]any{"ok": true}}, nil
}

// entityDeleteStep implements step.erp_entity_delete
type entityDeleteStep struct{ providerName string }

func (s *entityDeleteStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	p, err := getProvider(s.providerName)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": err.Error()}}, nil
	}
	entitySet := stringFrom(current, "entity_set")
	key := stringFrom(current, "key")
	if entitySet == "" || key == "" {
		return &sdk.StepResult{Output: map[string]any{"error": "entity_set and key are required"}}, nil
	}
	if err := p.erp.DeleteEntity(ctx, entitySet, key); err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("delete entity: %v", err)}}, nil
	}
	return &sdk.StepResult{Output: map[string]any{"ok": true}}, nil
}

// batchStep implements step.erp_batch
type batchStep struct{ providerName string }

func (s *batchStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	p, err := getProvider(s.providerName)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": err.Error()}}, nil
	}
	rawOps := sliceFrom(current, "operations")
	if len(rawOps) == 0 {
		return &sdk.StepResult{Output: map[string]any{"error": "operations array is required"}}, nil
	}
	ops := make([]BatchOp, 0, len(rawOps))
	for _, raw := range rawOps {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ops = append(ops, BatchOp{
			Method:    stringFrom(m, "method"),
			EntitySet: stringFrom(m, "entity_set"),
			Key:       stringFrom(m, "key"),
			Body:      mapFrom(m, "body"),
			ContentID: stringFrom(m, "content_id"),
		})
	}
	results, err := p.erp.BatchOperation(ctx, ops)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("batch: %v", err)}}, nil
	}
	out := make([]map[string]any, len(results))
	for i, r := range results {
		out[i] = map[string]any{
			"content_id":  r.ContentID,
			"status_code": r.StatusCode,
			"body":        r.Body,
		}
	}
	return &sdk.StepResult{Output: map[string]any{"results": out}}, nil
}

// functionCallStep implements step.erp_function_call
type functionCallStep struct{ providerName string }

func (s *functionCallStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	p, err := getProvider(s.providerName)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": err.Error()}}, nil
	}
	name := stringFrom(current, "function_name")
	if name == "" {
		return &sdk.StepResult{Output: map[string]any{"error": "function_name is required"}}, nil
	}
	params := mapFrom(current, "params")
	result, err := p.erp.CallFunction(ctx, name, params)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("function call: %v", err)}}, nil
	}
	return &sdk.StepResult{Output: map[string]any{"result": result}}, nil
}

// metadataStep implements step.erp_metadata
type metadataStep struct{ providerName string }

func (s *metadataStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, _ map[string]any, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	p, err := getProvider(s.providerName)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": err.Error()}}, nil
	}
	metadata, err := p.erp.GetMetadata(ctx)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("metadata: %v", err)}}, nil
	}
	return &sdk.StepResult{Output: map[string]any{"metadata": metadata}}, nil
}

// rawRequestStep implements step.erp_raw_request
type rawRequestStep struct{ providerName string }

func (s *rawRequestStep) Execute(ctx context.Context, _ map[string]any, _ map[string]map[string]any, current, _ map[string]any, _ map[string]any) (*sdk.StepResult, error) {
	p, err := getProvider(s.providerName)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": err.Error()}}, nil
	}
	method := strOr(current, "method", "GET")
	path := stringFrom(current, "path")
	if path == "" {
		return &sdk.StepResult{Output: map[string]any{"error": "path is required"}}, nil
	}
	body := mapFrom(current, "body")
	headers := stringMapFrom(current, "headers")
	status, respBody, err := p.erp.RawRequest(ctx, method, path, body, headers)
	if err != nil {
		return &sdk.StepResult{Output: map[string]any{"error": fmt.Sprintf("raw request: %v", err)}}, nil
	}
	return &sdk.StepResult{Output: map[string]any{
		"status_code": status,
		"body":        respBody,
	}}, nil
}
