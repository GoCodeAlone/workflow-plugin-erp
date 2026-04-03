package internal

import (
	"fmt"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// Manifest is the plugin manifest exported for tests.
var Manifest = sdk.PluginManifest{
	Name:        "workflow-plugin-erp",
	Version:     "0.1.0",
	Description: "Enterprise ERP integration (SAP S/4HANA via OData v4)",
	Author:      "GoCodeAlone",
}

type erpPlugin struct{}

// NewPlugin returns the ERP plugin provider.
func NewPlugin() *erpPlugin { return &erpPlugin{} }

func (p *erpPlugin) Manifest() sdk.PluginManifest { return Manifest }

// ModuleProvider

func (p *erpPlugin) ModuleTypes() []string { return []string{"erp.provider"} }

func (p *erpPlugin) CreateModule(typeName, name string, config map[string]any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "erp.provider":
		return newERPProvider(name, config), nil
	default:
		return nil, fmt.Errorf("unknown module type %q", typeName)
	}
}

// StepProvider

var stepTypes = []string{
	"step.erp_entity_read",
	"step.erp_entity_query",
	"step.erp_entity_create",
	"step.erp_entity_update",
	"step.erp_entity_delete",
	"step.erp_batch",
	"step.erp_function_call",
	"step.erp_metadata",
	"step.erp_raw_request",
}

func (p *erpPlugin) StepTypes() []string { return stepTypes }

func (p *erpPlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	providerName := strOr(config, "provider", "default")

	switch typeName {
	case "step.erp_entity_read":
		return &entityReadStep{providerName: providerName}, nil
	case "step.erp_entity_query":
		return &entityQueryStep{providerName: providerName}, nil
	case "step.erp_entity_create":
		return &entityCreateStep{providerName: providerName}, nil
	case "step.erp_entity_update":
		return &entityUpdateStep{providerName: providerName}, nil
	case "step.erp_entity_delete":
		return &entityDeleteStep{providerName: providerName}, nil
	case "step.erp_batch":
		return &batchStep{providerName: providerName}, nil
	case "step.erp_function_call":
		return &functionCallStep{providerName: providerName}, nil
	case "step.erp_metadata":
		return &metadataStep{providerName: providerName}, nil
	case "step.erp_raw_request":
		return &rawRequestStep{providerName: providerName}, nil
	default:
		return nil, fmt.Errorf("unknown step type %q", typeName)
	}
}

// Verify interface compliance at compile time.
var (
	_ sdk.PluginProvider = (*erpPlugin)(nil)
	_ sdk.ModuleProvider = (*erpPlugin)(nil)
	_ sdk.StepProvider   = (*erpPlugin)(nil)
)
