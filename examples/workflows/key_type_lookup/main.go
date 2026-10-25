// key_type_lookup: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/key_type_lookup
package main

import (
	"context"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("key_type_lookup", func(ctx context.Context, c *amaquet.Client, key string) error {
		if err := c.SetWithOptions(ctx, key, "integer", int64(12), amaquet.WriteOptions{}); err != nil {
			return err
		}

		typeName, err := c.TypeOf(ctx, key)
		if err == nil {
			common.Print("wire type", typeName)
		}

		return err
	})
}
