// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_auth
package main

import (
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	v, e := a.CheckAuth(ctx)
	common.Must(e)
	common.Print("session/token identity", v)
}
