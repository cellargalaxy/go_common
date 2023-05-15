package util

import (
	"fmt"
	"testing"
)

func TestTable(t *testing.T) {
	table := NewTable()
	table.AppendCell(0, 2, 2, "X4")
	table.AppendCell(0, 1, 1, "1C")
	table.AppendCell(0, 3, 2, "X6")
	table.AppendCell(1, 1, 1, "2C")
	table.AppendCell(2, 1, 2, "=+")
	table.AppendCell(2, 1, 1, "3C")
	table.AppendCell(3, 1, 1, "4A")
	table.AppendCell(3, 1, 1, "4B")
	table.AppendCell(3, 1, 1, "4C")
	table.AppendCell(3, 1, 2, "4D")

	lines := table.ListLine()
	for i := range lines {
		fmt.Println(JsonStruct2String(lines[i]))
	}

	fmt.Println(table.Render())
	if table.Render() != `+----+----+----+----+----+
| X4 | X4 | 1C | X6 | X6 |
| X4 | X4 | 2C | X6 | X6 |
| =+ | =+ | 3C | X6 | X6 |
| 4A | 4B | 4C | 4D | 4D |
+----+----+----+----+----+` {
		t.Errorf(`lines.Render() !=`)
		return
	}

	table.AddCol(2, "+2")
	fmt.Println(table.Render())
	if table.Render() != `+----+----+----+----+----+----+
| X4 | X4 | +2 | 1C | X6 | X6 |
| X4 | X4 | +2 | 2C | X6 | X6 |
| =+ | =+ | +2 | 3C | X6 | X6 |
| 4A | 4B | +2 | 4C | 4D | 4D |
+----+----+----+----+----+----+` {
		t.Errorf(`lines.Render() !=`)
		return
	}

	table.AddRow(3, 3, "+3")
	fmt.Println(table.Render())
	if table.Render() != `+----+----+----+----+----+----+
| X4 | X4 | +2 | 1C | X6 | X6 |
| X4 | X4 | +2 | 2C | X6 | X6 |
| =+ | =+ | +2 | 3C | X6 | X6 |
| +3 | +3 | +3 |    |    |    |
| 4A | 4B | +2 | 4C | 4D | 4D |
+----+----+----+----+----+----+` {
		t.Errorf(`lines.Render() !=`)
		return
	}

	table.RmRow(3)
	fmt.Println(table.Render())
	if table.Render() != `+----+----+----+----+----+----+
| X4 | X4 | +2 | 1C | X6 | X6 |
| X4 | X4 | +2 | 2C | X6 | X6 |
| =+ | =+ | +2 | 3C | X6 | X6 |
| 4A | 4B | +2 | 4C | 4D | 4D |
+----+----+----+----+----+----+` {
		t.Errorf(`lines.Render() !=`)
		return
	}
}
