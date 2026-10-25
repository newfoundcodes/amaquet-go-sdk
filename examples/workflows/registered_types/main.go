// registered_types: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/registered_types
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("registered_types", func(ctx context.Context, c *amaquet.Client, key string) error {
		remote, err := c.Types(ctx)

		if err == nil {
			common.Print("server types", remote)
			common.Print("SDK type count", len(amaquet.RegisteredTypes()))
		}

		return err
	})
}
