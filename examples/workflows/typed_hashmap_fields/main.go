// typed_hashmap_fields: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/typed_hashmap_fields
package main

import (
	"context"
	"encoding/json"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("typed_hashmap_fields", func(ctx context.Context, c *amaquet.Client, key string) error {
		h := c.TypeHashmap(key)
		if err := h.Operation.Create(ctx, map[string]any{}, amaquet.WriteOptions{}); err != nil {
			return err
		}

		var result json.RawMessage
		if err := h.Set(ctx, common.Args(`{"field":"name","value":{"type":"utf8_string","value":"Amaquet"}}`), &result); err != nil {
			return err
		}

		return h.Get(ctx, common.Args(`{"field":"name"}`), &result)
	})
}
