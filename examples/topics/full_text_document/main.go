// Example of the Amaquet full_text_document type, derived from its documented wire examples.
// Run: go run ./examples/topics/full_text_document
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

	key := "sdk:topic:full_text_document"
	handle := client.TypeFullTextDocument(key)
	_ = amaquet.Typed // Explicit nested value envelopes are needed for collection entries.
	common.Must(handle.Create(ctx, common.Args(`{"id": "doc-1", "metadata": {"source": "import"}}`), amaquet.WriteOptions{}))

	var result json.RawMessage
	common.Must(handle.AddText(ctx, common.Args(`{"name": "body", "value": "hello amaquet world", "weight": 2}`), &result))
	common.Print("full_text_document ADD_TEXT", result)

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("key snapshot", snapshot)
}
