// Standalone Amaquet SDK example. Run: go run ./examples/workflows/identity
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

	pin := os.Getenv("AMAQUET_IDENTITY_KEY")
	c, e := amaquet.DialWithOptions(ctx, uri, amaquet.DialOptions{
		APIKey:              os.Getenv("AMAQUET_API_KEY"),
		RequireIdentity:     pin != "",
		ExpectedIdentityKey: pin,
	})

	common.Must(e)
	defer c.Close()

	info := c.ServerInfo()
	fmt.Println(
		"server:", info.Name,
		"protocol:", info.SelectedProtocol,
		"identity present:", info.IdentityPublicKey != "",
	)
}
