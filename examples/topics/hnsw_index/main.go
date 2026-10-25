// Example of the Amaquet hnsw_index type, derived from its documented wire examples.
// Run: go run ./examples/topics/hnsw_index
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

	key := "sdk:topic:hnsw_index"
	handle := client.TypeHNSWIndex(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{"dim": 3, "metric": "cosine", "m": 8, "ef_search": 32}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Add(ctx, common.Args(`{"id": "v1", "vector": [1, 0, 0], "metadata": {}}`), &result))
	common.Print("hnsw_index ADD", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
