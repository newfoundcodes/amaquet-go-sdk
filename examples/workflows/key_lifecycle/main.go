// key_lifecycle: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/key_lifecycle
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("key_lifecycle", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "utf8_string", "hello", amaquet.WriteOptions{}); err != nil {
			return err
		}

		snapshot, err := c.GetSnapshot(ctx, key)
		if err == nil {
			common.Print("snapshot", snapshot)
		}

		return err
	})
}
