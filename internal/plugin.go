package internal

import (
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// Manifest returns the plugin metadata used by the workflow engine for
// discovery and capability negotiation.
var Manifest = sdk.PluginManifest{
	Name:        "workflow-plugin-TEMPLATE",
	Version:     "0.1.0",
	Description: "TEMPLATE plugin for the workflow engine",
	Author:      "GoCodeAlone",
	License:     "MIT",
	ModuleTypes: []string{
		// "example.module_type",
	},
	StepTypes: []string{
		// "step.example_action",
	},
}

type plugin struct{}

// NewPlugin creates a new plugin instance.
func NewPlugin() sdk.PluginProvider {
	return &plugin{}
}

func (p *plugin) Manifest() sdk.PluginManifest {
	return Manifest
}

func (p *plugin) ModuleFactories() map[string]sdk.ModuleFactory {
	return map[string]sdk.ModuleFactory{
		// "example.module_type": NewExampleModuleFactory(),
	}
}

func (p *plugin) StepFactories() map[string]sdk.StepFactory {
	return map[string]sdk.StepFactory{
		// "step.example_action": NewExampleStepFactory(),
	}
}
