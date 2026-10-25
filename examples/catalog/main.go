// The catalog runner previews documented operations without executing blocking
// or mutating operations by default. From the repository root:
// go run ./examples/sdk/catalog -type list
// go run ./examples/sdk/catalog -type list -create -op PUSH_BACK
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	amaquet "github.com/newfoundcodes/amaquet-go-sdk"
	"github.com/newfoundcodes/amaquet-go-sdk/examples/common"
)

type request struct {
	Command string         `json:"command"`
	Args    map[string]any `json:"args"`
}
type scenario struct {
	Type         string    `json:"type"`
	Construction request   `json:"construction"`
	Operations   []request `json:"operations"`
}

func main() {
	typeName := flag.String("type", "list", "Amaquet wire type")
	key := flag.String("key", "", "key override; defaults to sdk:catalog:<type>")
	create := flag.Bool("create", false, "apply the documented construction request")
	op := flag.String("op", "", "explicit operation to run; requires -create when using a new key")
	flag.Parse()

	valid := false
	for _, t := range amaquet.RegisteredTypes() {
		if t == *typeName {
			valid = true
			break
		}
	}

	if !valid {
		common.Must(fmt.Errorf("unknown type %q", *typeName))
	}

	path := filepath.Join("examples", "scenarios", *typeName+".json")
	b, err := os.ReadFile(path)
	common.Must(err)

	var s scenario
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.UseNumber()

	common.Must(d.Decode(&s))
	actualKey := *key

	if actualKey == "" {
		actualKey = "sdk:catalog:" + s.Type
	}

	s.Construction.Args["key"] = actualKey
	fmt.Printf("Type: %s\nConstruction: %s\n", s.Type, commonJSON(s.Construction))

	for _, cmd := range s.Operations {
		fmt.Printf("Documented OP: %v\n", cmd.Args["operation"])
	}

	if !*create && *op == "" {
		fmt.Println("Preview only; use -create or -op to execute.")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := common.Open(ctx)
	common.Must(err)
	defer client.Close()

	if *create {
		var result json.RawMessage

		common.Must(client.Command(ctx, s.Construction.Command, s.Construction.Args, &result))
		common.Print("constructed", result)
	}

	if *op != "" {
		matched := false
		for _, cmd := range s.Operations {
			if strings.EqualFold(fmt.Sprint(cmd.Args["operation"]), *op) {
				matched = true
				cmd.Args["key"] = actualKey

				var out json.RawMessage
				common.Must(client.Command(ctx, cmd.Command, cmd.Args, &out))
				common.Print("result", out)

				break
			}
		}

		if !matched {
			common.Must(fmt.Errorf("type %q does not document OP %q", s.Type, *op))
		}
	}
}

func commonJSON(x any) string {
	b, e := json.Marshal(x)
	common.Must(e)

	return string(b)
}
