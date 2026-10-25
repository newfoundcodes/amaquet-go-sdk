// batch_read: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/batch_read
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("batch_read", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "integer", int64(10), amaquet.WriteOptions{}); err != nil {
			return err
		}

		rows, err := c.Batch(ctx, []amaquet.BatchRequest{
			{
				Command: "GET",
				Args:    map[string]any{"key": key},
			},
			{Command: "PING"},
		}, false)

		if err == nil {
			common.Print("batch responses", rows)
		}

		return err
	})
}
