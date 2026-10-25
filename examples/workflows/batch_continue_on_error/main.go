// batch_continue_on_error: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/batch_continue_on_error
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("batch_continue_on_error", func(ctx context.Context, c *amaquet.Client, key string) error {
		rows, err := c.Batch(ctx, []amaquet.BatchRequest{
			{
				Command: "GET",
				Args:    map[string]any{"key": key + ":missing"},
			},
			{Command: "PING"},
		}, true)

		if err == nil {
			common.Print("independent item statuses", rows)
		}

		return err
	})
}
