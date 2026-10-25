// Example of the Amaquet dense_vector type, derived from its documented wire examples.
// Run: go run ./examples/topics/dense_vector
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

	key := "sdk:topic:dense_vector"
	handle := client.TypeDenseVector(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.

	var initial any
	initial = common.Args(`{"value": [1.0, 0.0, 0.5]}`)["value"]
	common.Must(handle.Operation.Set(ctx, initial, amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Distance(ctx, common.Args(`{"vector": [1.0, 0.0, 0.5], "metric": "cosine"}`), &result))
	common.Print("dense_vector DISTANCE", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
