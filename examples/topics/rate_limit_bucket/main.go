// Example of the Amaquet rate_limit_bucket type, derived from its documented wire examples.
// Run: go run ./examples/topics/rate_limit_bucket
package main

import (
	"encoding/json"

	amaquet "github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	client, err := common.Open(ctx)
	common.Must(err)
	defer client.Close()

	key := "sdk:topic:rate_limit_bucket"
	handle := client.TypeRateLimitBucket(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{"capacity": 5, "refill_per_second": 2}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Allow(ctx, common.Args(`{"cost": 1}`), &result))
	common.Print("rate_limit_bucket ALLOW", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
