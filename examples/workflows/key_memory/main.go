// key_memory: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/key_memory
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("key_memory", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "utf8_string", "memory", amaquet.WriteOptions{}); err != nil {
			return err
		}

		info, err := c.Memory(ctx, key)
		if err == nil {
			common.Print("key memory", info)
		}

		return err
	})
}
