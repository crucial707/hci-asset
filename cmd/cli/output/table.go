package output

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

// RenderTable prints a pretty table to stdout
func RenderTable(headers []string, rows [][]interface{}) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	headerRow := table.Row{}
	for _, h := range headers {
		headerRow = append(headerRow, h)
	}
	t.AppendHeader(headerRow)

	for _, row := range rows {
		t.AppendRow(table.Row(row))
	}

	t.Render()
}
