// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_metrics
package main

import (
	"fmt"
	"strings"

	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	v, e := a.Metrics(ctx)
	common.Must(e)

	lines := strings.Split(v, "\n")
	for i, line := range lines {
		if i >= 8 {
			break
		}

		fmt.Println(line)
	}
}
