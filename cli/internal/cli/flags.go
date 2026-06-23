package cli

import (
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// exactArgs is cobra.ExactArgs but yields a UsageError (exit code 2).
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return Usagef("%s expects %d argument(s), got %d", cmd.CommandPath(), n, len(args))
		}
		return nil
	}
}

// parseKV turns repeated key=value flag values into a map.
func parseKV(pairs []string, flag string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, Usagef("--%s expects key=value, got %q", flag, p)
		}
		m[k] = v
	}
	return m, nil
}

// readData resolves a --data value: "@file" reads the file, "-" reads stdin,
// anything else is treated as inline JSON.
func readData(stdin io.Reader, val string) (json.RawMessage, error) {
	var b []byte
	var err error
	switch {
	case val == "-":
		b, err = io.ReadAll(stdin)
	case strings.HasPrefix(val, "@"):
		b, err = os.ReadFile(strings.TrimPrefix(val, "@"))
	default:
		b = []byte(val)
	}
	if err != nil {
		return nil, Usagef("reading --data: %v", err)
	}
	if !json.Valid(b) {
		return nil, Usagef("--data is not valid JSON")
	}
	return json.RawMessage(b), nil
}

// bindBody implements the --data XOR typed-flags contract and returns the
// request body as the generated input type T. With --data set, the raw JSON is
// decoded into T (using the generated UnmarshalJSON) and no other local flag
// may be changed; otherwise build populates T from typed flags.
func bindBody[T any](cmd *cobra.Command, build func(*T) error) (*T, error) {
	var body T
	dataVal, _ := cmd.Flags().GetString("data")
	if dataVal == "" {
		if err := build(&body); err != nil {
			return nil, err
		}
		return &body, nil
	}
	if conflict := dataConflict(cmd); conflict != "" {
		return nil, Usagef("--data cannot be combined with --%s", conflict)
	}
	raw, err := readData(cmd.InOrStdin(), dataVal)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, Usagef("decoding --data: %v", err)
	}
	return &body, nil
}

// rawBodyJSON builds a raw JSON request body for endpoints whose generated
// input type is free-form (jx.Raw). With --data set the raw bytes pass through
// (and no other local flag may change); otherwise build returns a value that is
// marshaled to JSON.
func rawBodyJSON(cmd *cobra.Command, build func() (any, error)) ([]byte, error) {
	dataVal, _ := cmd.Flags().GetString("data")
	if dataVal != "" {
		if conflict := dataConflict(cmd); conflict != "" {
			return nil, Usagef("--data cannot be combined with --%s", conflict)
		}
		raw, err := readData(cmd.InOrStdin(), dataVal)
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	v, err := build()
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// dataConflict reports the name of any locally-changed flag besides --data,
// enforcing that --data is used alone.
func dataConflict(cmd *cobra.Command) string {
	localNames := make(map[string]bool)
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) { localNames[f.Name] = true })
	conflict := ""
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if f.Name != "data" && localNames[f.Name] {
			conflict = f.Name
		}
	})
	return conflict
}

// addListFlags registers the standard pagination flags on a list command.
func addListFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.Int("page", 0, "page number (zero-indexed)")
	f.Int("limit", 10, "items per page")
	f.String("sort-by", "created_at", "sort field")
	f.String("sort-order", "desc", "asc or desc")
}

// listArgs reads the standard pagination flags. Each list command builds its
// own generated *Params from these (the generated param types are distinct per
// operation but share these four fields).
func listArgs(cmd *cobra.Command) (page, limit int, sortBy, sortOrder string) {
	page, _ = cmd.Flags().GetInt("page")
	limit, _ = cmd.Flags().GetInt("limit")
	sortBy, _ = cmd.Flags().GetString("sort-by")
	sortOrder, _ = cmd.Flags().GetString("sort-order")
	return
}

// addDataFlag registers the raw-JSON-body escape hatch on a mutating command.
func addDataFlag(cmd *cobra.Command) {
	cmd.Flags().String("data", "", "raw JSON body (@file, -, or inline)")
}

// annotate records which API operation a command covers; coverage_test.go
// diffs these annotations against openapi.json. path must match the spec path
// exactly (including the /api prefix and {param} names).
func annotate(cmd *cobra.Command, method, path string) *cobra.Command {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	op := method + " " + path
	if cur := cmd.Annotations["api.ops"]; cur != "" {
		op = cur + "\n" + op
	}
	cmd.Annotations["api.ops"] = op
	return cmd
}
