// raw_typed_operation: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/raw_typed_operation
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("raw_typed_operation", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.Create(ctx, key, "atomic_counter", map[string]any{"initial": int64(7)}); err != nil {
			return err
		}

		result, err := c.RawOp(ctx, key, "ADD", map[string]any{"delta": 3})
		if err == nil {
			common.Print("raw OP result", result)
		}

		return err
	})
}
