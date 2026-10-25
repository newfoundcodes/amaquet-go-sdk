// Example of the Amaquet quantized_vector type, derived from its documented wire examples.
// Run: go run ./examples/topics/quantized_vector
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

	key := "sdk:topic:quantized_vector"
	handle := client.TypeQuantizedVector(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.

	var initial any
	initial = common.Args(`{"value": {"Data": [1, 2, -3], "Scale": 0.1, "ZeroPoint": 0}}`)["value"]
	common.Must(handle.Operation.Set(ctx, initial, amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Dequantize(ctx, common.Args(`{}`), &result))
	common.Print("quantized_vector DEQUANTIZE", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
