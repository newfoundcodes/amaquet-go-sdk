// persist_expiration: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/persist_expiration
package main

import (
	"context"
	"time"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("persist_expiration", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "utf8_string", "keep", amaquet.WriteOptions{TTL: time.Minute}); err != nil {
			return err
		}

		if err := c.Persist(ctx, key); err != nil {
			return err
		}

		ttl, err := c.TTL(ctx, key)
		if err == nil {
			common.Print("persistent TTL", ttl)
		}

		return err
	})
}
