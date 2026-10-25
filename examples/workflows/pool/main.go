// Standalone Amaquet SDK example. Run: go run ./examples/workflows/pool
package main

import (
	"fmt"
	"os"

	"github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	uri := os.Getenv("AMAQUET_URI")
	if uri == "" {
		uri = "amaquet://127.0.0.1:13378"
	}

	p, e := amaquet.DialPool(ctx, uri, 4, amaquet.DialOptions{APIKey: os.Getenv("AMAQUET_API_KEY")})
	common.Must(e)
	defer p.Close()

	var pong string
	common.Must(p.Command(ctx, "PING", nil, &pong))
	fmt.Println("pong:", pong)
}
