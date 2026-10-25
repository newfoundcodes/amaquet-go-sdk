// Example of the Amaquet fifo_queue type, derived from its documented wire examples.
// Run: go run ./examples/topics/fifo_queue
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

	key := "sdk:topic:fifo_queue"
	handle := client.TypeFIFOQueue(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Enqueue(ctx, common.Args(`{"value": {"type": "utf8_string", "value": "first"}}`), &result))
	common.Print("fifo_queue ENQUEUE", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
