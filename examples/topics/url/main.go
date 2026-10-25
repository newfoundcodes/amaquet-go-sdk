// Example of the Amaquet url type, derived from its documented wire examples.
// Run: go run ./examples/topics/url
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

	key := "sdk:topic:url"
	handle := client.TypeURL(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.

	var initial any
	initial = common.Args(`{"value": "https://example.com:8443/path?q=amaquet"}`)["value"]
	common.Must(handle.Operation.Set(ctx, initial, amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.Host(ctx, common.Args(`{}`), &result))
	common.Print("url HOST", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
