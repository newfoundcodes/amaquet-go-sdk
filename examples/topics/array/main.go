// Example of the Amaquet array type, derived from its documented wire examples.
// Run: go run ./examples/topics/array
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

	key := "sdk:topic:array"
	handle := client.TypeArray(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.

	common.Must(handle.Create(ctx, common.Args(`{"typed": true, "element_type": "utf8_string"}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Append(ctx, common.Args(`{"value": {"type": "utf8_string", "value": "alpha"}}`), &result))
	common.Print("array APPEND", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
