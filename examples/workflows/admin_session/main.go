// Standalone Amaquet SDK example. Run: go run ./examples/workflows/admin_session
package main

import (
	"fmt"
	"os"

	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

func main() {
	ctx, cancel := common.Context()
	defer cancel()

	a, e := common.OpenAdmin()
	common.Must(e)

	token := os.Getenv("AMAQUET_SESSION_TOKEN")
	if token == "" {
		fmt.Println("Set AMAQUET_SESSION_TOKEN for a cookie-based login")
		return
	}

	_, e = a.CreateSession(ctx, token, os.Getenv("AMAQUET_MFA_CODE"))
	common.Must(e)
	defer a.DeleteSession(ctx)

	v, e := a.CheckAuth(ctx)
	common.Must(e)
	common.Print("session", v)
}
