// trie: documented construction and named operations, direct Go SDK usage.
// Run: go run ./examples/topics/trie
// Optional: go run ./examples/topics/trie -op <DOCUMENTED_OPERATION>
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	amaquet "github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	operation := flag.String("op", "", "documented type operation; runs only when requested")
	flag.Parse()

	ctx, cancel := common.Context()
	defer cancel()

	client, err := common.Open(ctx)
	common.Must(err)
	defer client.Close()

	key := fmt.Sprintf("sdk:topic:trie:%d", time.Now().UnixNano())
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 3*time.Second)
		defer done()

		_, _ = client.Delete(cleanupCtx, key)
	}()

	handle := client.TypeTrie(key)
	common.Must(handle.Operation.Create(ctx, common.Args(`{}`), amaquet.WriteOptions{}))

	fmt.Printf("Created %s as trie\n", key)
	if *operation != "" {
		switch *operation {
		case "INSERT":
			var result json.RawMessage
			common.Must(handle.Insert(ctx, common.Args(`{"value":"amaquet","id":"1"}`), &result))
			common.Print("INSERT", result)

		case "PREFIX":
			var result json.RawMessage
			common.Must(handle.Prefix(ctx, common.Args(`{"prefix":"ama","limit":10}`), &result))
			common.Print("PREFIX", result)

		case "EXACT":
			var result json.RawMessage
			common.Must(handle.Exact(ctx, common.Args(`{"value":"amaquet"}`), &result))
			common.Print("EXACT", result)

		default:
			common.Must(fmt.Errorf("unknown or undocumented operation: %q", *operation))
		}
	}

	snapshot, err := handle.KeySnapshot(ctx)
	common.Must(err)
	common.Print("snapshot", snapshot)
}
