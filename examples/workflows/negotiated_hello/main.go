// negotiated_hello: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/negotiated_hello
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("negotiated_hello", func(ctx context.Context, c *amaquet.Client, key string) error {
		info, err := c.Hello(ctx, "sdk-example-hello")
		if err == nil {
			common.Print("negotiated server", info)
		}

		return err
	})
}
