// fifo_queue_push: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/fifo_queue_push
package main

import (
	"context"
	"encoding/json"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("fifo_queue_push", func(ctx context.Context, c *amaquet.Client, key string) error {
		h := c.TypeFIFOQueue(key)
		if err := h.Operation.Create(ctx, map[string]any{}, amaquet.WriteOptions{}); err != nil {
			return err
		}

		var result json.RawMessage
		if err := h.Enqueue(ctx, common.Args(`{"value":{"type":"utf8_string","value":"job"}}`), &result); err != nil {
			return err
		}

		err := h.Len(ctx, map[string]any{}, &result)
		if err == nil {
			common.Print("queued", result)
		}

		return err
	})
}
