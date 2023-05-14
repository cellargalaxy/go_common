package util

import (
	"github.com/jedib0t/go-pretty/v6/table"
)

func NewTable(lines ...[]string) *Table {
	table := &Table{}
	for i := range lines {
		line := make([]*string, 0, len(lines[i]))
		for j := range lines[i] {
			line = append(line, &lines[i][j])
		}
		table.table = append(table.table, line)
	}
	return table
}

type Table struct {
	table [][]*string
}

func (this *Table) Render() string {
	lines := this.GetTable()
	t := table.NewWriter()
	for i := range lines {
		cells := make([]interface{}, 0, len(lines[i]))
		for j := range lines[i] {
			cells = append(cells, lines[i][j])
		}
		t.AppendRow(cells)
	}
	return t.Render()
}
func (this *Table) GetTable() [][]string {
	table := make([][]string, 0, len(this.table))
	for i := range this.table {
		line := make([]string, 0, len(this.table[i]))
		for j := range this.table[i] {
			var cell string
			if this.table[i][j] != nil {
				cell = *this.table[i][j]
			}
			line = append(line, cell)
		}
		table = append(table, line)
	}
	return table
}
func (this *Table) SetCell(row, col int, value string) {
	for len(this.table) <= row {
		this.table = append(this.table, []*string{})
	}
	for len(this.table[row]) <= col {
		this.table[row] = append(this.table[row], nil)
	}
	this.table[row][col] = &value
}
func (this *Table) AppendCell(row, rowspan, colspan int, value string) {
	for len(this.table) <= row {
		this.table = append(this.table, []*string{})
	}
	col := len(this.table[row])
	for j := range this.table[row] {
		if this.table[row][j] == nil {
			col = j
			break
		}
	}
	for i := 0; i < rowspan; i++ {
		for j := 0; j < colspan; j++ {
			this.SetCell(row+i, col+j, value)
		}
	}
}
