// prefix_keys: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/prefix_keys
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("prefix_keys", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "integer", int64(1), amaquet.WriteOptions{}); err != nil {
			return err
		}

		keys, err := c.Keys(ctx, key, 10)
		if err == nil {
			common.Print("prefix keys", keys)
		}

		return err
	})
}
