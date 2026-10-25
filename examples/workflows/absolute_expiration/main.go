// absolute_expiration: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/absolute_expiration
package main

import (
	"context"
	"time"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("absolute_expiration", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "utf8_string", "absolute expiry", amaquet.WriteOptions{}); err != nil {
			return err
		}

		if err := c.ExpireAt(ctx, key, time.Now().Add(2*time.Minute)); err != nil {
			return err
		}

		ttl, err := c.TTL(ctx, key)
		if err == nil {
			common.Print("remaining ttl", ttl)
		}

		return err
	})
}
