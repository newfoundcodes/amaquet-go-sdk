// Standalone Amaquet SDK example. Run: go run ./examples/workflows/pubsub
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

	key := fmt.Sprintf("sdk:pubsub:%d", time.Now().UnixNano())
	common.Must(c.Create(ctx, key, "pubsub", nil))

	sub, e := c.Subscribe(ctx, key, "events", 32)
	common.Must(e)
	defer sub.Close()

	common.Must(c.Op(ctx, key, "PUBLISH", map[string]any{
		"channel": "events",
		"value":   amaquet.Typed("utf8_string", "ping"),
	}, nil))

	select {
	case msg := <-sub.C:
		common.Print("event", msg)

	case <-ctx.Done():
		common.Must(ctx.Err())
	}
}
