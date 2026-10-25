// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_command
package main

import (
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	v, e := a.DataCommand(ctx, "INFO", nil)
	common.Must(e)
	common.Print("HTTP data command", v)
}
