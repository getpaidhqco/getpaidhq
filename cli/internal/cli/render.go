package cli

import (
	"encoding/json"
	"fmt"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
	"github.com/getpaidhqco/getpaidhq/cli/internal/cli/output"
)

// renderOne renders a single resource: pretty JSON when -o json, otherwise a
// one-row table built by row.
func renderOne[T any](app *App, v T, headers []string, row func(T) []string) error {
	if app.Output == "json" {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return output.JSON(app.Out, b)
	}
	return output.Table(app.Out, headers, [][]string{row(v)})
}

// renderList renders a paginated ListResponse: a table plus a meta footer, or
// raw JSON. The server's list envelope carries items as raw JSON in Data, so
// they are decoded into []T for the table.
func renderList[T any](app *App, lr *apigen.ListResponse, headers []string, row func(T) []string) error {
	if app.Output == "json" {
		b, err := json.Marshal(lr)
		if err != nil {
			return err
		}
		return output.JSON(app.Out, b)
	}
	var items []T
	if len(lr.Data) > 0 {
		if err := json.Unmarshal(lr.Data, &items); err != nil {
			return fmt.Errorf("decoding list data: %w", err)
		}
	}
	rows := make([][]string, len(items))
	for i, it := range items {
		rows[i] = row(it)
	}
	if err := output.Table(app.Out, headers, rows); err != nil {
		return err
	}
	m := lr.Meta.Or(apigen.ListResponseMeta{})
	_, err := fmt.Fprintf(app.Out, "\ntotal %d · page %d · limit %d\n", m.Total.Or(0), m.Page.Or(0), m.Limit.Or(0))
	return err
}

// renderValue prints a resource as pretty JSON in both output modes — for
// auxiliary resources that have no useful table shape.
func renderValue[T any](app *App, v T) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return output.JSON(app.Out, b)
}

// renderDeleted confirms a deletion in table mode and stays silent in json mode.
func renderDeleted(app *App, what string) error {
	if app.Output == "json" {
		return nil
	}
	_, err := fmt.Fprintf(app.Out, "%s deleted\n", what)
	return err
}
