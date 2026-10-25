// series_sample: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/series_sample
package main

import (
	"context"
	"encoding/json"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("series_sample", func(ctx context.Context, c *amaquet.Client, key string) error {
		h := c.TypeTimeseries(key)
		if err := h.Operation.Create(ctx, map[string]any{}, amaquet.WriteOptions{}); err != nil {
			return err
		}

		var result json.RawMessage
		if err := h.Add(ctx, common.Args(`{"timestamp":1730000000000,"value":10}`), &result); err != nil {
			return err
		}

		err := h.Last(ctx, map[string]any{}, &result)
		if err == nil {
			common.Print("latest", result)
		}

		return err
	})
}
