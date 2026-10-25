// stream_append: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/stream_append
package main

import (
	"context"
	"encoding/json"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("stream_append", func(ctx context.Context, c *amaquet.Client, key string) error {
		h := c.TypeStream(key)
		if err := h.Operation.Create(ctx, map[string]any{}, amaquet.WriteOptions{}); err != nil {
			return err
		}

		var result json.RawMessage
		if err := h.Add(ctx, common.Args(`{"fields":{"message":{"type":"utf8_string","value":"hello"}}}`), &result); err != nil {
			return err
		}

		err := h.Len(ctx, map[string]any{}, &result)
		if err == nil {
			common.Print("entries", result)
		}

		return err
	})
}
