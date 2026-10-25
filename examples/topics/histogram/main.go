// Example of the Amaquet histogram type, derived from its documented wire examples.
// Run: go run ./examples/topics/histogram
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

	key := "sdk:topic:histogram"
	handle := client.TypeHistogram(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{"bounds": [1, 5, 10, 50, 100]}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Observe(ctx, common.Args(`{"value": 3}`), &result))
	common.Print("histogram OBSERVE", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
