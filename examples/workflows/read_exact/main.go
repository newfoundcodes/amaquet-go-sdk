// Standalone Amaquet SDK example. Run: go run ./examples/workflows/read_exact
package main

import (
	"fmt"
	"time"

	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	c, e := common.Open(ctx)
	common.Must(e)
	defer c.Close()

	key := fmt.Sprintf("sdk:exact:%d", time.Now().UnixNano())
	common.Must(c.Set(ctx, key, "unsigned_integer", uint64(18446744073709551615), 0))

	s, e := c.GetSnapshot(ctx, key)
	common.Must(e)

	fmt.Println("exact JSON numeric token:", string(s.Value))
}
