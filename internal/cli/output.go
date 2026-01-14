package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

type output struct {
	Out  io.Writer
	JSON bool
}

func (o output) PrintJSON(v any) error {
	enc := json.NewEncoder(o.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (o output) PrintTable(headers []string, rows [][]string) error {
	w := tabwriter.NewWriter(o.Out, 0, 0, 2, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(w, joinRow(headers))
	}
	for _, row := range rows {
		fmt.Fprintln(w, joinRow(row))
	}
	return w.Flush()
}

func joinRow(cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	out := cols[0]
	for i := 1; i < len(cols); i++ {
		out += "\t" + cols[i]
	}
	return out
}
