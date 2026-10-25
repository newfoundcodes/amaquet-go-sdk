// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_data
package main

import (
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	v, e := a.DataKeys(ctx, "", "sdk:", 50)
	common.Must(e)
	common.Print("paged keys", v)
}
