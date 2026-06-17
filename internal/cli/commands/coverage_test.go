package commands_test

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"getpaidhq/internal/cli"
)

// Operations the CLI deliberately does not cover.
var coverageSkip = map[string]bool{
	"POST /api/notify": true, // inbound PSP webhook receiver, not a client operation
}

var specMethods = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}

func TestEveryAPIOperationHasACommand(t *testing.T) {
	raw, err := os.ReadFile("../../../openapi.json")
	if err != nil {
		t.Fatalf("openapi.json: %v (regenerate by booting the server)", err)
	}
	var spec struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}

	covered := map[string]bool{}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if ops := c.Annotations["api.ops"]; ops != "" {
			for _, op := range strings.Split(ops, "\n") {
				covered[op] = true
			}
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(cli.NewRootCmd(strings.NewReader(""), io.Discard, io.Discard))

	specOps := map[string]bool{}
	for path, methods := range spec.Paths {
		for m := range methods {
			if !specMethods[m] {
				continue
			}
			specOps[strings.ToUpper(m)+" "+path] = true
		}
	}

	for op := range specOps {
		if !covered[op] && !coverageSkip[op] {
			t.Errorf("no CLI command covers %s", op)
		}
	}
	for op := range covered {
		if !specOps[op] {
			t.Errorf("command annotation %q matches no operation in openapi.json (typo in annotate()?)", op)
		}
	}
}
