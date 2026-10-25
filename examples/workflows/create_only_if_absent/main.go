// create_only_if_absent: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/create_only_if_absent
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("create_only_if_absent", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "utf8_string", "first", amaquet.WriteOptions{NX: true}); err != nil {
			return err
		}

		snap, err := c.GetSnapshot(ctx, key)
		if err == nil {
			common.Print("initial value", snap)
		}

		return err
	})
}
