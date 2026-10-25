// Example of the Amaquet big_decimal type, derived from its documented wire examples.
// Run: go run ./examples/topics/big_decimal
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

	key := "sdk:topic:big_decimal"
	handle := client.TypeBigDecimal(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.

	var initial any
	initial = common.Args(`{"value": "12345678901234567890.000000000000000001"}`)["value"]
	common.Must(handle.Operation.Set(ctx, initial, amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Add(ctx, common.Args(`{"value": "0.3211"}`), &result))
	common.Print("big_decimal ADD", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
