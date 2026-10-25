// Example of the Amaquet record type, derived from its documented wire examples.
// Run: go run ./examples/topics/record
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

	key := "sdk:topic:record"
	handle := client.TypeRecord(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{"schema": [{"name": "name", "type": "utf8_string", "nullable": false}, {"name": "age", "type": "integer", "nullable": true}]}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Set(ctx, common.Args(`{"field": "name", "value": {"type": "utf8_string", "value": "alice"}}`), &result))
	common.Print("record SET", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
