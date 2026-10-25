// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_members
package main

import (
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	v, e := a.Members(ctx)
	common.Must(e)
	common.Print("members", v)

	v, e = a.RBAC(ctx)
	common.Must(e)
	common.Print("permissions", v)
}
