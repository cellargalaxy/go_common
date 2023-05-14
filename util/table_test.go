package util

import (
	"fmt"
	"testing"
)

func TestTable(t *testing.T) {
	table := NewTable()
	table.AppendCell(0, 2, 2, "乘四")
	table.AppendCell(0, 1, 1, "1C")
	table.AppendCell(0, 3, 2, "乘六")
	table.AppendCell(1, 1, 1, "2C")
	table.AppendCell(2, 1, 2, "横向合并")
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
	if table.Render() != `+----------+----------+----+------+------+
| 乘四     | 乘四     | 1C | 乘六 | 乘六 |
| 乘四     | 乘四     | 2C | 乘六 | 乘六 |
| 横向合并 | 横向合并 | 3C | 乘六 | 乘六 |
| 4A       | 4B       | 4C | 4D   | 4D   |
+----------+----------+----+------+------+` {
		t.Errorf(`lines.Render() !=`)
		return
	}
}
