package util

import (
	pretty_table "github.com/jedib0t/go-pretty/v6/table"
)

func NewTable(lines ...[]string) *Table {
	tab := &Table{}
	for i := range lines {
		line := make([]*string, 0, len(lines[i]))
		for j := range lines[i] {
			line = append(line, &lines[i][j])
		}
		tab.lines = append(tab.lines, line)
	}
	return tab
}

type Table struct {
	lines [][]*string
}

func (this Table) String() string {
	return JsonStruct2String(this.lines)
}

func (this *Table) IsEmpty() bool {
	for i := range this.lines {
		for j := range this.lines[i] {
			if this.lines[i][j] != nil {
				return false
			}
		}
	}
	return true
}
func (this *Table) Render() string {
	lines := this.ListLine()
	tab := pretty_table.NewWriter()
	for i := range lines {
		line := make([]interface{}, 0, len(lines[i]))
		for j := range lines[i] {
			line = append(line, lines[i][j])
		}
		tab.AppendRow(line)
	}
	return tab.Render()
}
func (this *Table) ListLine() [][]string {
	lines := make([][]string, 0, len(this.lines))
	for i := range this.lines {
		line := make([]string, 0, len(this.lines[i]))
		for j := range this.lines[i] {
			var cell string
			if this.lines[i][j] != nil {
				cell = *this.lines[i][j]
			}
			line = append(line, cell)
		}
		lines = append(lines, line)
	}
	return lines
}
func (this *Table) listLine() [][]*string {
	lines := make([][]*string, 0, len(this.lines))
	for i := range this.lines {
		line := make([]*string, 0, len(this.lines[i]))
		for j := range this.lines[i] {
			var cell *string
			if this.lines[i][j] != nil {
				value := *this.lines[i][j]
				cell = &value
			}
			line = append(line, cell)
		}
		lines = append(lines, line)
	}
	return lines
}
func (this *Table) GetRow(row int) []string {
	for len(this.lines) <= row {
		return nil
	}
	line := make([]string, 0, len(this.lines[row]))
	for j := range this.lines[row] {
		var cell string
		if this.lines[row][j] != nil {
			cell = *this.lines[row][j]
		}
		line = append(line, cell)
	}
	return line
}
func (this *Table) SetCell(row, col int, value string) {
	this.setCell(row, col, &value)
}
func (this *Table) setCell(row, col int, value *string) {
	for len(this.lines) <= row {
		this.lines = append(this.lines, []*string{})
	}
	for len(this.lines[row]) <= col {
		this.lines[row] = append(this.lines[row], nil)
	}
	this.lines[row][col] = value
}
func (this *Table) AppendCell(row, rowspan, colspan int, value string) {
	for len(this.lines) <= row {
		this.lines = append(this.lines, []*string{})
	}
	col := len(this.lines[row])
	for j := range this.lines[row] {
		if this.lines[row][j] == nil {
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
func (this *Table) AddCol(col int, value string) {
	lines := this.listLine()
	for i := range lines {
		for j := range lines[i] {
			if j < col {
				this.setCell(i, j, lines[i][j])
			}
			if j == col {
				this.setCell(i, j, &value)
				this.setCell(i, j+1, lines[i][j])
			}
			if col < j {
				this.setCell(i, j+1, lines[i][j])
			}
		}
	}
}
func (this *Table) AddRow(row, rowspan int, value string) {
	rows := make([]*string, rowspan)
	for i := range rows {
		rows[i] = &value
	}
	lines := make([][]*string, 0)
	for i := range this.lines {
		if i == row {
			lines = append(lines, rows)
			lines = append(lines, this.lines[i])
			continue
		}
		lines = append(lines, this.lines[i])
	}
	this.lines = lines
}
func (this *Table) RmRow(row int) {
	lines := make([][]*string, 0)
	for i := range this.lines {
		if i == row {
			continue
		}
		lines = append(lines, this.lines[i])
	}
	this.lines = lines
}
func (this *Table) AppendRow(values ...string) {
	rows := make([]*string, len(values))
	for i := range rows {
		rows[i] = &values[i]
	}
	this.lines = append(this.lines, rows)
}
func (this *Table) AppendRowTable(table *Table) {
	if table == nil {
		return
	}
	this.lines = append(this.lines, table.lines...)
}
