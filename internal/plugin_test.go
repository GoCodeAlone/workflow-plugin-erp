package internal_test

import (
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-erp/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func TestNewPlugin_ImplementsPluginProvider(t *testing.T) {
	var _ sdk.PluginProvider = internal.NewPlugin()
}

func TestNewPlugin_ImplementsModuleProvider(t *testing.T) {
	var _ sdk.ModuleProvider = internal.NewPlugin()
}

func TestNewPlugin_ImplementsStepProvider(t *testing.T) {
	var _ sdk.StepProvider = internal.NewPlugin()
}

func TestManifest_HasRequiredFields(t *testing.T) {
	m := internal.Manifest
	if m.Name == "" {
		t.Error("manifest Name is empty")
	}
	if m.Version == "" {
		t.Error("manifest Version is empty")
	}
	if m.Description == "" {
		t.Error("manifest Description is empty")
	}
}

func TestModuleTypes(t *testing.T) {
	p := internal.NewPlugin()
	types := p.ModuleTypes()
	if len(types) != 1 || types[0] != "erp.provider" {
		t.Errorf("expected [erp.provider], got %v", types)
	}
}

func TestStepTypes(t *testing.T) {
	p := internal.NewPlugin()
	types := p.StepTypes()
	if len(types) != 9 {
		t.Errorf("expected 9 step types, got %d", len(types))
	}
}

func TestCreateModule_UnknownType(t *testing.T) {
	p := internal.NewPlugin()
	_, err := p.CreateModule("bogus.type", "test", nil)
	if err == nil {
		t.Error("expected error for unknown module type")
	}
}

func TestCreateStep_UnknownType(t *testing.T) {
	p := internal.NewPlugin()
	_, err := p.CreateStep("bogus.type", "test", nil)
	if err == nil {
		t.Error("expected error for unknown step type")
	}
}

func TestCreateStep_AllTypes(t *testing.T) {
	p := internal.NewPlugin()
	for _, st := range p.StepTypes() {
		_, err := p.CreateStep(st, "test-"+st, map[string]any{"provider": "default"})
		if err != nil {
			t.Errorf("CreateStep(%q) failed: %v", st, err)
		}
	}
}
