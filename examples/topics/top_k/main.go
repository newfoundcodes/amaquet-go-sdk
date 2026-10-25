// Example of the Amaquet top_k type, derived from its documented wire examples.
// Run: go run ./examples/topics/top_k
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

	key := "sdk:topic:top_k"
	handle := client.TypeTopK(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{"k": 5}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Add(ctx, common.Args(`{"value": "alice", "count": 10}`), &result))
	common.Print("top_k ADD", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
