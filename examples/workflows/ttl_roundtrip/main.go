// ttl_roundtrip: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/ttl_roundtrip
package main

import (
	"context"
	"time"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("ttl_roundtrip", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "utf8_string", "short lived", amaquet.WriteOptions{}); err != nil {
			return err
		}

		if err := c.Expire(ctx, key, 30*time.Second); err != nil {
			return err
		}

		ttl, err := c.TTL(ctx, key)
		if err == nil {
			common.Print("ttl ms", ttl)
		}

		return err
	})
}
