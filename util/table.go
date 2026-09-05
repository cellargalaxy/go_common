package util

import (
	"github.com/jedib0t/go-pretty/v6/table"
)

func NewTable(lines ...[]string) *Table {
	table := &Table{}
	table.lines = make([][]*string, len(lines))
	for i := range lines {
		table.lines[i] = S2Ps(lines[i]...)
	}
	return table
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
			if this.lines[i][j] != nil && *this.lines[i][j] != "" {
				return false
			}
		}
	}
	return true
}
func (this *Table) Render() string {
	lines := this.ListLine()
	table := table.NewWriter()
	for i := range lines {
		line := make([]interface{}, 0, len(lines[i]))
		for j := range lines[i] {
			line = append(line, lines[i][j])
		}
		table.AppendRow(line)
	}
	return table.Render()
}
func (this *Table) ListLine() [][]string {
	lines := make([][]string, len(this.lines))
	for i := range this.lines {
		lines[i] = P2Ss(this.lines[i]...)
	}
	return lines
}
func (this *Table) listLine() [][]*string {
	lines := make([][]*string, len(this.lines))
	for i := range this.lines {
		lines[i] = CopyArray(this.lines[i]...)
	}
	return lines
}
func (this *Table) GetRow(row int) []string {
	//负下标不能落到 this.lines[row]，否则直接panic；越界与负值统一返回nil
	if row < 0 || len(this.lines) <= row {
		return nil
	}
	return P2Ss(this.lines[row]...)
}
func (this *Table) SetCell(row, col int, value string) {
	this.setCell(row, col, &value)
}
func (this *Table) setCell(row, col int, value *string) {
	//负下标的扩容循环条件恒为false，会直接走到 this.lines[row][col] 而panic。
	//写入类操作遇到非法下标不能改写其它位置的单元格（那是静默数据损坏），故直接忽略
	if row < 0 || col < 0 {
		return
	}
	for len(this.lines) <= row {
		this.lines = append(this.lines, []*string{})
	}
	for len(this.lines[row]) <= col {
		this.lines[row] = append(this.lines[row], nil)
	}
	this.lines[row][col] = value
}
func (this *Table) AppendCell(row, rowspan, colspan int, value string) {
	//负行号在下面的 this.lines[row] 取值处会panic
	if row < 0 {
		return
	}
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
	//负col时下面三个分支中 col<j 恒成立，会把每个单元格右移却不写入新值，
	//结果是把首个单元格复制一份（[a b] -> [a a b]），属静默数据损坏，故直接忽略
	if col < 0 {
		return
	}
	lines := this.listLine()
	for i := range lines {
		for j := range lines[i] {
			if j < col {
				this.setCell(i, j, lines[i][j])
			}
			if j == col {
				//每行插入的新格必须各自持有独立指针，
				//否则所有行的该列共享同一个 &value，改一行会串改所有行
				cell := value
				this.setCell(i, j, &cell)
				this.setCell(i, j+1, lines[i][j])
			}
			if col < j {
				this.setCell(i, j+1, lines[i][j])
			}
		}
	}
}
func (this *Table) AddRow(row, rowspan int, value string) {
	//负rowspan会让 make([]*string, rowspan) 直接panic(makeslice: len out of range)。
	//本类型其余写入方法(SetCell/AppendCell/AddCol)对非法下标一律忽略，此处保持一致：
	//夹紧为0，等价于插入一个空行
	if rowspan < 0 {
		rowspan = 0
	}
	rows := make([]*string, rowspan)
	for i := range rows {
		//不能让整行共享同一个 &value：与 AppendRow 修复前同源的指针别名问题，
		//后续任一格通过指针被改写会连带改掉同行其余格子。逐格复制一份再取地址
		cell := value
		rows[i] = &cell
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
	//不能取 &values[i]：values 与调用方传入的切片共享底层数组
	//（AppendRow(lines[i]...) 是本库既有调用方式），调用方之后修改该切片会穿透进表格。
	//这里按值复制一份再取地址
	rows := make([]*string, len(values))
	for i := range rows {
		value := values[i]
		rows[i] = &value
	}
	this.lines = append(this.lines, rows)
}
func (this *Table) AppendRowTable(tables ...*Table) {
	for i := range tables {
		//nil表格会在取 tables[i].lines 时panic
		if tables[i] == nil {
			continue
		}
		//直接append源表的行切片会让两张表共享同一行，改一张会影响另一张，故逐行复制
		for j := range tables[i].lines {
			this.lines = append(this.lines, CopyArray(tables[i].lines[j]...))
		}
	}
}
