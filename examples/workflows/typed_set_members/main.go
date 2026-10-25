// typed_set_members: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/typed_set_members
package main

import (
	"context"
	"encoding/json"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("typed_set_members", func(ctx context.Context, c *amaquet.Client, key string) error {
		h := c.TypeSet(key)
		if err := h.Operation.Create(ctx, map[string]any{}, amaquet.WriteOptions{}); err != nil {
			return err
		}

		var result json.RawMessage
		if err := h.Add(ctx, common.Args(`{"value":{"type":"utf8_string","value":"member"}}`), &result); err != nil {
			return err
		}

		common.Print("added", result)
		return h.Members(ctx, map[string]any{}, &result)
	})
}
