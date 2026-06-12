package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/cobra"

	"getpaidhq/internal/cli/client"
	"getpaidhq/internal/cli/output"
)

// UsageError marks errors caused by bad invocation (exit code 2).
type UsageError struct{ msg string }

func (e *UsageError) Error() string { return e.msg }

func Usagef(format string, a ...any) error {
	return &UsageError{msg: fmt.Sprintf(format, a...)}
}

// FormatError renders any command error for stderr, unwrapping the API
// error envelope when present.
func FormatError(err error) string {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		s := fmt.Sprintf("error (%s): %s", apiErr.Code, apiErr.Message)
		if apiErr.Details != nil {
			if d, jerr := json.Marshal(apiErr.Details); jerr == nil {
				s += "\n  details: " + string(d)
			}
		}
		return s
	}
	return "error: " + err.Error()
}

// exactArgs is cobra.ExactArgs but yields a UsageError (exit code 2).
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return Usagef("%s expects %d argument(s), got %d", cmd.CommandPath(), n, len(args))
		}
		return nil
	}
}

// readData resolves a --data value: "@file" reads the file, "-" reads
// stdin, anything else is inline JSON.
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
		return nil, fmt.Errorf("reading --data: %w", err)
	}
	if !json.Valid(b) {
		return nil, Usagef("--data is not valid JSON")
	}
	return json.RawMessage(b), nil
}

// bodyOrData enforces the --data XOR typed-flags contract: when --data is
// set, no other local flag may be changed; otherwise build() constructs
// the typed request body.
func bodyOrData(cmd *cobra.Command, build func() (any, error)) (any, error) {
	dataVal, _ := cmd.Flags().GetString("data")
	if dataVal == "" {
		return build()
	}
	// Build the set of flag names local to this command (not inherited from
	// parent persistent flags), then find any locally-changed flag besides
	// --data.
	localNames := make(map[string]bool)
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) { localNames[f.Name] = true })
	conflict := ""
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if f.Name != "data" && localNames[f.Name] {
			conflict = f.Name
		}
	})
	if conflict != "" {
		return nil, Usagef("--data cannot be combined with --%s", conflict)
	}
	return readData(cmd.InOrStdin(), dataVal)
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

// annotate records which API operation a command covers; coverage_test.go
// diffs these annotations against openapi.json. path must match the spec
// path exactly (including the /api prefix and {param} names).
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

func addListFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.Int("page", 0, "page number (zero-indexed)")
	f.Int("limit", 10, "items per page")
	f.String("sort-by", "created_at", "sort field")
	f.String("sort-order", "desc", "asc or desc")
}

func listQuery(cmd *cobra.Command) url.Values {
	page, _ := cmd.Flags().GetInt("page")
	limit, _ := cmd.Flags().GetInt("limit")
	sortBy, _ := cmd.Flags().GetString("sort-by")
	order, _ := cmd.Flags().GetString("sort-order")
	return client.ListQuery(page, limit, sortBy, order)
}

// render prints raw JSON when -o json, otherwise delegates to the
// command's table renderer.
func render(app *App, raw []byte, table func(raw []byte) error) error {
	if app.Output == "json" {
		return output.JSON(app.Out, raw)
	}
	return table(raw)
}

// renderJSON is for auxiliary resources without a table shape: pretty
// JSON in both output modes.
func renderJSON(app *App, raw []byte) error {
	return output.JSON(app.Out, raw)
}

// listEnvelope decodes the {data,meta} list envelope around T.
type listEnvelope[T any] struct {
	Data []T `json:"data"`
	Meta struct {
		Total int `json:"total"`
		Page  int `json:"page"`
		Limit int `json:"limit"`
	} `json:"meta"`
}

// renderList renders a paginated list: table + meta footer, or raw JSON.
func renderList[T any](app *App, raw []byte, headers []string, row func(T) []string) error {
	if app.Output == "json" {
		return output.JSON(app.Out, raw)
	}
	var page listEnvelope[T]
	if err := json.Unmarshal(raw, &page); err != nil {
		return fmt.Errorf("decoding list response: %w", err)
	}
	rows := make([][]string, len(page.Data))
	for i, item := range page.Data {
		rows[i] = row(item)
	}
	if err := output.Table(app.Out, headers, rows); err != nil {
		return err
	}
	_, err := fmt.Fprintf(app.Out, "\ntotal %d · page %d · limit %d\n", page.Meta.Total, page.Meta.Page, page.Meta.Limit)
	return err
}

// renderDeleted handles 204 responses: confirmation in table mode,
// silence in json mode.
func renderDeleted(app *App, what string) error {
	if app.Output == "json" {
		return nil
	}
	_, err := fmt.Fprintf(app.Out, "%s deleted\n", what)
	return err
}
