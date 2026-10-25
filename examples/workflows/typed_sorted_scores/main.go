// typed_sorted_scores: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/typed_sorted_scores
package main

import (
	"context"
	"encoding/json"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("typed_sorted_scores", func(ctx context.Context, c *amaquet.Client, key string) error {
		h := c.TypeSortedSet(key)
		if err := h.Operation.Create(ctx, map[string]any{}, amaquet.WriteOptions{}); err != nil {
			return err
		}

		var result json.RawMessage
		if err := h.Add(ctx, common.Args(`{"member":{"type":"utf8_string","value":"alpha"},"score":10.5}`), &result); err != nil {
			return err
		}

		err := h.Score(ctx, common.Args(`{"member":{"type":"utf8_string","value":"alpha"}}`), &result)
		if err == nil {
			common.Print("score", result)
		}

		return err
	})
}
