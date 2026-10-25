// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_audit
package main

import (
	"fmt"

	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	v, e := a.Audit(ctx, 20)
	common.Must(e)

	fmt.Println("audit response bytes:", len(v))
}
