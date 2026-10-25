// Standalone Amaquet SDK example. Run: go run ./examples/workflows/scan
package main

import (
	"fmt"

	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	c, e := common.Open(ctx)
	common.Must(e)
	defer c.Close()

	count := 0
	common.Must(c.ScanAll(ctx, "sdk:", 50, func(k string) error {
		count++
		fmt.Println(k)
		return nil
	}))

	fmt.Println("observed keys:", count)
}
