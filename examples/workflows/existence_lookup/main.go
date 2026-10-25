// existence_lookup: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/existence_lookup
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("existence_lookup", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "boolean", true, amaquet.WriteOptions{}); err != nil {
			return err
		}

		exists, err := c.Exists(ctx, key)
		if err == nil {
			common.Print("exists", exists)
		}

		return err
	})
}
