// bloom_filter_membership: independent Amaquet workflow demonstration.
// Run: go run ./examples/workflows/bloom_filter_membership
package main

import (
	"context"
	"encoding/json"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	common.RunExample("bloom_filter_membership", func(ctx context.Context, c *amaquet.Client, key string) error {
		h := c.TypeBloomFilter(key)
		if err := h.Operation.Create(ctx, common.Args(`{"expected":1000,"false_positive_rate":0.01}`), amaquet.WriteOptions{}); err != nil {
			return err
		}

		var result json.RawMessage
		if err := h.Add(ctx, common.Args(`{"value":"alice"}`), &result); err != nil {
			return err
		}

		err := h.Contains(ctx, common.Args(`{"value":"alice"}`), &result)
		if err == nil {
			common.Print("possibly contains", result)
		}

		return err
	})
}
