// Standalone Amaquet SDK example. Run: go run ./examples/workflows/batch_cas
package main

import (
	"fmt"
	"time"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	c, e := common.Open(ctx)
	common.Must(e)
	defer c.Close()

	key := fmt.Sprintf("sdk:cas:%d", time.Now().UnixNano())
	common.Must(c.SetWithOptions(ctx, key, "integer", int64(10), amaquet.WriteOptions{}))

	s, e := c.GetSnapshot(ctx, key)
	common.Must(e)
	common.Must(c.CompareAndSet(ctx, key, s.Version, "integer", int64(12), amaquet.WriteOptions{}))

	results, e := c.Batch(ctx, []amaquet.BatchRequest{{Command: "GET", Args: map[string]any{"key": key}}, {Command: "PING"}}, false)
	common.Must(e)
	common.Print("sequential batch", results)
}
