// Command workflow-plugin-TEMPLATE is a workflow engine external plugin.
// It runs as a subprocess and communicates with the host workflow engine
// via the go-plugin gRPC protocol.
package main

import (
	"github.com/GoCodeAlone/workflow-plugin-TEMPLATE/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewPlugin())
}
