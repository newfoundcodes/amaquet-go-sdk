// Example of the Amaquet radix_tree type, derived from its documented wire examples.
// Run: go run ./examples/topics/radix_tree
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

	key := "sdk:topic:radix_tree"
	handle := client.TypeRadixTree(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Insert(ctx, common.Args(`{"value": "amaquet", "id": "1"}`), &result))
	common.Print("radix_tree INSERT", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
