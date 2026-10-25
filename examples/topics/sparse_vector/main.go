// Example of the Amaquet sparse_vector type, derived from its documented wire examples.
// Run: go run ./examples/topics/sparse_vector
package main

import (
	"fmt"

	amaquet "github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	client, err := common.Open(ctx)
	common.Must(err)
	defer client.Close()

	key := "sdk:topic:sparse_vector"
	handle := client.TypeSparseVector(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.

	var initial any
	initial = common.Args(`{"value": {"dim": 8, "values": {"1": 0.5, "5": 1.25}}}`)["value"]

	common.Must(handle.Operation.Set(ctx, initial, amaquet.WriteOptions{}))
	fmt.Println("Created key:", key)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
