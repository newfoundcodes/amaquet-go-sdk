// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_system
package main

import (
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	v, e := a.System(ctx)
	common.Must(e)
	common.Print("public server config", v)
}
