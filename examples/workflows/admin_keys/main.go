// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_keys
package main

import (
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	v, e := a.APIKeys(ctx)
	common.Must(e)
	common.Print("API keys (no raw secrets)", v)
}
