// connect_ping: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/connect_ping
package main

import (
	"context"
	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("connect_ping", func(ctx context.Context, c *amaquet.Client, key string) error {
		return c.Ping(ctx)
	})
}
