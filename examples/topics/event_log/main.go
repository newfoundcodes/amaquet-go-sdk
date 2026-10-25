// Example of the Amaquet event_log type, derived from its documented wire examples.
// Run: go run ./examples/topics/event_log
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

	key := "sdk:topic:event_log"
	handle := client.TypeEventLog(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Append(ctx, common.Args(`{"event_type": "created", "aggregate_id": "agg1", "payload": {"type": "utf8_string", "value": "born"}}`), &result))
	common.Print("event_log APPEND", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
