// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_health
package main

import (
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	v, e := a.Health(ctx)
	common.Must(e)
	common.Print("health", v)

	v, e = a.Ready(ctx)
	common.Must(e)
	common.Print("readiness", v)
}
