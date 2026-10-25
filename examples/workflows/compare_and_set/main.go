// compare_and_set: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/compare_and_set
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("compare_and_set", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "integer", int64(10), amaquet.WriteOptions{}); err != nil {
			return err
		}

		snap, err := c.GetSnapshot(ctx, key)
		if err != nil {
			return err
		}

		if err := c.CompareAndSet(ctx, key, snap.Version, "integer", int64(11), amaquet.WriteOptions{}); err != nil {
			return err
		}

		snap, err = c.GetSnapshot(ctx, key)
		if err == nil {
			common.Print("updated", snap)
		}

		return err
	})
}
