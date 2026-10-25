// Example of the Amaquet unsigned_integer type, derived from its documented wire examples.
// Run: go run ./examples/topics/unsigned_integer
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

	key := "sdk:topic:unsigned_integer"
	handle := client.TypeUnsignedInteger(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.

	var initial any
	initial = common.Args(`{"value": 42}`)["value"]
	common.Must(handle.Operation.Set(ctx, initial, amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Add(ctx, common.Args(`{"delta": 8}`), &result))
	common.Print("unsigned_integer ADD", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
