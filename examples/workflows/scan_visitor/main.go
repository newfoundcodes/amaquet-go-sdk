// scan_visitor: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/scan_visitor
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("scan_visitor", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "integer", int64(1), amaquet.WriteOptions{}); err != nil {
			return err
		}

		var seen []string
		err := c.ScanAll(ctx, key, 10, func(k string) error {
			seen = append(seen, k)
			return nil
		})

		if err == nil {
			common.Print("scan", seen)
		}

		return err
	})
}
