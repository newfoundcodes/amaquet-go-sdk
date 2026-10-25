// Example of the Amaquet vector_set type, derived from its documented wire examples.
// Run: go run ./examples/topics/vector_set
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

	key := "sdk:topic:vector_set"
	handle := client.TypeVectorSet(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{"dim": 3, "metric": "cosine"}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Add(ctx, common.Args(`{"id": "v1", "vector": [1, 0, 0], "metadata": {"class": "a"}}`), &result))
	common.Print("vector_set ADD", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
