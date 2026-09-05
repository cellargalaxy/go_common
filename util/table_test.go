package util

import (
	"testing"
)

// 综合场景：与被删除前的原单测保持一致的黄金渲染结果，
// 确保 AppendCell 的跨行跨列填充、AddCol/AddRow/RmRow 的整体行为未回退
func TestTableGoldenRender(t *testing.T) {
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

	want := `+----+----+----+----+----+
| X4 | X4 | 1C | X6 | X6 |
| X4 | X4 | 2C | X6 | X6 |
| =+ | =+ | 3C | X6 | X6 |
| 4A | 4B | 4C | 4D | 4D |
+----+----+----+----+----+`
	if got := table.Render(); got != want {
		t.Errorf("AppendCell 构表渲染结果不符\n实际:\n%s\n期望:\n%s", got, want)
	}

	table.AddCol(2, "+2")
	want = `+----+----+----+----+----+----+
| X4 | X4 | +2 | 1C | X6 | X6 |
| X4 | X4 | +2 | 2C | X6 | X6 |
| =+ | =+ | +2 | 3C | X6 | X6 |
| 4A | 4B | +2 | 4C | 4D | 4D |
+----+----+----+----+----+----+`
	if got := table.Render(); got != want {
		t.Errorf("AddCol 后渲染结果不符\n实际:\n%s\n期望:\n%s", got, want)
	}

	table.AddRow(3, 3, "+3")
	want = `+----+----+----+----+----+----+
| X4 | X4 | +2 | 1C | X6 | X6 |
| X4 | X4 | +2 | 2C | X6 | X6 |
| =+ | =+ | +2 | 3C | X6 | X6 |
| +3 | +3 | +3 |    |    |    |
| 4A | 4B | +2 | 4C | 4D | 4D |
+----+----+----+----+----+----+`
	if got := table.Render(); got != want {
		t.Errorf("AddRow 后渲染结果不符\n实际:\n%s\n期望:\n%s", got, want)
	}

	table.RmRow(3)
	want = `+----+----+----+----+----+----+
| X4 | X4 | +2 | 1C | X6 | X6 |
| X4 | X4 | +2 | 2C | X6 | X6 |
| =+ | =+ | +2 | 3C | X6 | X6 |
| 4A | 4B | +2 | 4C | 4D | 4D |
+----+----+----+----+----+----+`
	if got := table.Render(); got != want {
		t.Errorf("RmRow 后渲染结果不符\n实际:\n%s\n期望:\n%s", got, want)
	}
}

func TestNewTable(t *testing.T) {
	//空表
	empty := NewTable()
	if !empty.IsEmpty() {
		t.Errorf("NewTable() 应为空表")
	}
	if got := empty.ListLine(); len(got) != 0 {
		t.Errorf("NewTable().ListLine() = %v, 期望空", got)
	}
	if got := empty.Render(); got != "" {
		t.Errorf("空表 Render() = %q, 期望空串", got)
	}

	//带初始行，允许不等长（ragged）
	tb := NewTable([]string{"a", "b"}, []string{"c"})
	lines := tb.ListLine()
	if len(lines) != 2 || len(lines[0]) != 2 || len(lines[1]) != 1 {
		t.Fatalf("NewTable 初始行结构错误: %v", lines)
	}
	if lines[0][0] != "a" || lines[0][1] != "b" || lines[1][0] != "c" {
		t.Errorf("NewTable 内容错误: %v", lines)
	}

	//NewTable 必须复制入参，调用方之后修改原切片不能穿透
	src := []string{"orig"}
	cp := NewTable(src)
	src[0] = "mutated"
	if got := cp.GetRow(0); len(got) != 1 || got[0] != "orig" {
		t.Errorf("NewTable 未复制入参，原切片修改穿透: %v", got)
	}
}

func TestTableIsEmpty(t *testing.T) {
	if !NewTable().IsEmpty() {
		t.Errorf("无行表应为空")
	}
	//全空串视为空
	if !NewTable([]string{"", ""}).IsEmpty() {
		t.Errorf("全空串表应为空")
	}
	//仅nil空洞加空串也视为空
	holes := NewTable()
	holes.SetCell(1, 1, "")
	if !holes.IsEmpty() {
		t.Errorf("仅nil空洞与空串的表应为空")
	}
	//任一非空单元格即非空
	if NewTable([]string{"", "x"}).IsEmpty() {
		t.Errorf("含非空单元格的表不应为空")
	}
	//空白字符不算空（不做TrimSpace）
	if NewTable([]string{" "}).IsEmpty() {
		t.Errorf("含空格的单元格按当前约定不算空")
	}
}

func TestTableString(t *testing.T) {
	tb := NewTable([]string{"a", "b"}, []string{"c"})
	if got := tb.String(); got != `[["a","b"],["c"]]` {
		t.Errorf("String() = %q", got)
	}
	//nil空洞序列化为null，用于暴露稀疏结构
	sparse := NewTable()
	sparse.SetCell(0, 2, "v")
	if got := sparse.String(); got != `[[null,null,"v"]]` {
		t.Errorf("稀疏表 String() = %q, 期望 [[null,null,\"v\"]]", got)
	}
	if got := NewTable().String(); got != `[]` {
		t.Errorf("空表 String() = %q", got)
	}
}

func TestTableGetRow(t *testing.T) {
	tb := NewTable([]string{"a", "b"}, []string{"c"})
	if got := tb.GetRow(0); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("GetRow(0) = %v", got)
	}
	if got := tb.GetRow(1); len(got) != 1 || got[0] != "c" {
		t.Errorf("GetRow(1) = %v", got)
	}
	//越界返回nil而非panic
	if got := tb.GetRow(2); got != nil {
		t.Errorf("GetRow(越界) = %v, 期望 nil", got)
	}
	if got := tb.GetRow(99); got != nil {
		t.Errorf("GetRow(99) = %v, 期望 nil", got)
	}
	//负下标曾panic(index out of range)，必须返回nil
	if got := tb.GetRow(-1); got != nil {
		t.Errorf("GetRow(-1) = %v, 期望 nil", got)
	}
	//返回值是副本，修改不能影响表格
	row := tb.GetRow(0)
	row[0] = "mutated"
	if again := tb.GetRow(0); again[0] != "a" {
		t.Errorf("GetRow 返回值非副本，修改穿透: %v", again)
	}
	//nil空洞转为空串
	sparse := NewTable()
	sparse.SetCell(0, 2, "v")
	if got := sparse.GetRow(0); len(got) != 3 || got[0] != "" || got[1] != "" || got[2] != "v" {
		t.Errorf("GetRow 对nil空洞 = %v, 期望 [\"\",\"\",\"v\"]", got)
	}
}

func TestTableSetCell(t *testing.T) {
	tb := NewTable()
	tb.SetCell(0, 0, "a")
	if got := tb.GetRow(0); len(got) != 1 || got[0] != "a" {
		t.Errorf("SetCell(0,0) = %v", got)
	}
	//覆盖同一位置
	tb.SetCell(0, 0, "b")
	if got := tb.GetRow(0); got[0] != "b" {
		t.Errorf("SetCell 覆盖失败 = %v", got)
	}
	//稀疏写入自动扩容，中间以nil空洞填充
	sparse := NewTable()
	sparse.SetCell(2, 3, "v")
	lines := sparse.ListLine()
	if len(lines) != 3 {
		t.Fatalf("SetCell 未扩容到3行: %v", lines)
	}
	if len(lines[0]) != 0 || len(lines[1]) != 0 {
		t.Errorf("SetCell 扩容出的前置行应为空行: %v", lines)
	}
	if len(lines[2]) != 4 || lines[2][3] != "v" || lines[2][0] != "" {
		t.Errorf("SetCell 目标行 = %v", lines[2])
	}

	//负下标曾panic，且不得改写其它单元格
	neg := NewTable([]string{"keep"})
	neg.SetCell(-1, 0, "bad")
	neg.SetCell(0, -1, "bad")
	neg.SetCell(-5, -5, "bad")
	if got := neg.ListLine(); len(got) != 1 || len(got[0]) != 1 || got[0][0] != "keep" {
		t.Errorf("SetCell 负下标破坏了表格内容: %v", got)
	}
}

func TestTableListLine(t *testing.T) {
	tb := NewTable([]string{"a", "b"}, []string{"c"})
	//返回值必须是副本，修改不能穿透
	got := tb.ListLine()
	got[0][0] = "mutated"
	if again := tb.ListLine(); again[0][0] != "a" {
		t.Errorf("ListLine 返回值非副本，修改穿透: %v", again)
	}
	if got := NewTable().ListLine(); len(got) != 0 {
		t.Errorf("空表 ListLine = %v", got)
	}
}

func TestTableAppendRow(t *testing.T) {
	tb := NewTable()
	tb.AppendRow("a", "b")
	tb.AppendRow("c")
	lines := tb.ListLine()
	if len(lines) != 2 || lines[0][0] != "a" || lines[0][1] != "b" || lines[1][0] != "c" {
		t.Fatalf("AppendRow = %v", lines)
	}
	//空入参追加一个空行
	tb.AppendRow()
	if lines = tb.ListLine(); len(lines) != 3 || len(lines[2]) != 0 {
		t.Errorf("AppendRow() 空入参 = %v", lines)
	}

	//关键回归：AppendRow(slice...) 曾直接持有调用方切片元素的地址，
	//调用方随后复用/修改该切片会穿透篡改已入表的数据。
	//tool/bank_statement.go 正是以 AppendRow(lines[i]...) 方式调用
	src := []string{"r0c0", "r0c1"}
	alias := NewTable()
	alias.AppendRow(src...)
	src[0] = "OVERWRITTEN"
	if got := alias.GetRow(0); got[0] != "r0c0" {
		t.Errorf("AppendRow 与入参切片共享底层数组，调用方修改穿透: %v", got)
	}

	//复用同一缓冲区连续追加多行，各行必须互相独立
	buf := []string{"v"}
	reuse := NewTable()
	reuse.AppendRow(buf...)
	buf[0] = "second"
	reuse.AppendRow(buf...)
	if r0, r1 := reuse.GetRow(0), reuse.GetRow(1); r0[0] != "v" || r1[0] != "second" {
		t.Errorf("复用缓冲区追加导致行数据串味: r0=%v r1=%v", r0, r1)
	}
}

func TestTableAppendRowTable(t *testing.T) {
	dst := NewTable([]string{"d0"})
	src1 := NewTable([]string{"s0"}, []string{"s1"})
	src2 := NewTable([]string{"s2"})
	dst.AppendRowTable(src1, src2)
	lines := dst.ListLine()
	if len(lines) != 4 {
		t.Fatalf("AppendRowTable 行数 = %d, 期望 4: %v", len(lines), lines)
	}
	if lines[0][0] != "d0" || lines[1][0] != "s0" || lines[2][0] != "s1" || lines[3][0] != "s2" {
		t.Errorf("AppendRowTable 内容/顺序错误: %v", lines)
	}

	//关键回归：曾直接append源表的行切片，两表共享同一行，改源表会篡改目标表
	a := NewTable([]string{"orig"})
	b := NewTable([]string{"orig"})
	a.AppendRowTable(b)
	b.SetCell(0, 0, "CHANGED")
	if got := a.GetRow(1); got[0] != "orig" {
		t.Errorf("AppendRowTable 与源表共享行，源表修改穿透: %v", got)
	}

	//nil表格曾panic(nil pointer dereference)，须跳过
	n := NewTable([]string{"keep"})
	n.AppendRowTable(nil)
	n.AppendRowTable(nil, NewTable([]string{"ok"}), nil)
	lines = n.ListLine()
	if len(lines) != 2 || lines[0][0] != "keep" || lines[1][0] != "ok" {
		t.Errorf("AppendRowTable 含nil时结果错误: %v", lines)
	}

	//空表与无入参
	z := NewTable([]string{"keep"})
	z.AppendRowTable()
	z.AppendRowTable(NewTable())
	if got := z.ListLine(); len(got) != 1 {
		t.Errorf("AppendRowTable 空入参改变了表格: %v", got)
	}
}

func TestTableAppendCell(t *testing.T) {
	//单个单元格
	tb := NewTable()
	tb.AppendCell(0, 1, 1, "a")
	tb.AppendCell(0, 1, 1, "b")
	if got := tb.GetRow(0); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("AppendCell 顺序追加 = %v", got)
	}

	//跨行跨列填充：rowspan×colspan 个单元格都填同一值
	span := NewTable()
	span.AppendCell(0, 2, 3, "S")
	lines := span.ListLine()
	if len(lines) != 2 {
		t.Fatalf("AppendCell rowspan=2 行数 = %d", len(lines))
	}
	for i := 0; i < 2; i++ {
		if len(lines[i]) != 3 {
			t.Fatalf("AppendCell colspan=3 第%d行列数 = %d", i, len(lines[i]))
		}
		for j := 0; j < 3; j++ {
			if lines[i][j] != "S" {
				t.Errorf("AppendCell 跨区填充缺失 [%d][%d] = %q", i, j, lines[i][j])
			}
		}
	}

	//span为0时不写入任何单元格，但目标行仍被创建
	zero := NewTable()
	zero.AppendCell(0, 0, 0, "V")
	if got := zero.ListLine(); len(got) != 1 || len(got[0]) != 0 {
		t.Errorf("AppendCell span=0 = %v", got)
	}

	//优先填入首个nil空洞，而非直接追加到行尾
	hole := NewTable()
	hole.SetCell(0, 2, "at2")
	hole.AppendCell(0, 1, 1, "H")
	if got := hole.GetRow(0); len(got) != 3 || got[0] != "H" || got[2] != "at2" {
		t.Errorf("AppendCell 未填入首个nil空洞: %v", got)
	}

	//负行号曾panic(index out of range [-1])
	neg := NewTable([]string{"keep"})
	neg.AppendCell(-1, 1, 1, "bad")
	if got := neg.ListLine(); len(got) != 1 || got[0][0] != "keep" {
		t.Errorf("AppendCell 负行号破坏了表格: %v", got)
	}
}

func TestTableAddCol(t *testing.T) {
	//在中间插入一列，原有列右移
	tb := NewTable([]string{"a", "b", "c"})
	tb.AddCol(1, "N")
	if got := tb.GetRow(0); len(got) != 4 || got[0] != "a" || got[1] != "N" || got[2] != "b" || got[3] != "c" {
		t.Errorf("AddCol(1) = %v, 期望 [a N b c]", got)
	}
	//在首列插入
	head := NewTable([]string{"a", "b"})
	head.AddCol(0, "N")
	if got := head.GetRow(0); len(got) != 3 || got[0] != "N" || got[1] != "a" || got[2] != "b" {
		t.Errorf("AddCol(0) = %v, 期望 [N a b]", got)
	}
	//每一行都要插入
	multi := NewTable([]string{"a", "b"}, []string{"c", "d"})
	multi.AddCol(1, "N")
	if r0, r1 := multi.GetRow(0), multi.GetRow(1); r0[1] != "N" || r1[1] != "N" {
		t.Errorf("AddCol 未作用于所有行: %v / %v", r0, r1)
	}
	//不等长行：各行按自身长度处理
	ragged := NewTable([]string{"a", "b"}, []string{"c", "d", "e"})
	ragged.AddCol(1, "N")
	if got := ragged.GetRow(0); len(got) != 3 || got[0] != "a" || got[1] != "N" || got[2] != "b" {
		t.Errorf("AddCol ragged 短行 = %v", got)
	}
	if got := ragged.GetRow(1); len(got) != 4 || got[0] != "c" || got[1] != "N" || got[2] != "d" || got[3] != "e" {
		t.Errorf("AddCol ragged 长行 = %v", got)
	}

	//col超出行长度时不改变表格（既有行为，无处可插）
	beyond := NewTable([]string{"a", "b"})
	beyond.AddCol(5, "N")
	if got := beyond.GetRow(0); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("AddCol 越界应不改变表格, got %v", got)
	}
	//col恰等于行长度同样不插入（既有行为，追加请用SetCell/AppendCell）
	edge := NewTable([]string{"a", "b"})
	edge.AddCol(2, "N")
	if got := edge.GetRow(0); len(got) != 2 {
		t.Errorf("AddCol(col==len) 结果 = %v", got)
	}

	//负col曾把首个单元格复制一份（[a b] -> [a a b]）造成静默数据损坏
	neg := NewTable([]string{"a", "b"})
	neg.AddCol(-1, "N")
	if got := neg.GetRow(0); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("AddCol 负col损坏数据: %v, 期望保持 [a b]", got)
	}

	//空表不受影响
	empty := NewTable()
	empty.AddCol(0, "N")
	if got := empty.ListLine(); len(got) != 0 {
		t.Errorf("空表 AddCol = %v", got)
	}
}

func TestTableAddRow(t *testing.T) {
	//在指定位置前插入一行，rowspan决定该行的列数
	tb := NewTable([]string{"r0"}, []string{"r1"})
	tb.AddRow(1, 2, "N")
	lines := tb.ListLine()
	if len(lines) != 3 {
		t.Fatalf("AddRow 行数 = %d, 期望 3: %v", len(lines), lines)
	}
	if lines[0][0] != "r0" || lines[2][0] != "r1" {
		t.Errorf("AddRow 原有行顺序被破坏: %v", lines)
	}
	if len(lines[1]) != 2 || lines[1][0] != "N" || lines[1][1] != "N" {
		t.Errorf("AddRow 新行 = %v, 期望 [N N]", lines[1])
	}
	//插入到首行
	head := NewTable([]string{"r0"})
	head.AddRow(0, 1, "N")
	if got := head.ListLine(); len(got) != 2 || got[0][0] != "N" || got[1][0] != "r0" {
		t.Errorf("AddRow(0) = %v", got)
	}
	//rowspan=0 插入空行
	zero := NewTable([]string{"r0"}, []string{"r1"})
	zero.AddRow(1, 0, "N")
	if got := zero.ListLine(); len(got) != 3 || len(got[1]) != 0 {
		t.Errorf("AddRow rowspan=0 = %v", got)
	}
	//row越界时不插入（既有行为：追加请用AppendRow）
	beyond := NewTable([]string{"r0"}, []string{"r1"})
	beyond.AddRow(2, 1, "N")
	if got := beyond.ListLine(); len(got) != 2 {
		t.Errorf("AddRow(row==len) 结果 = %v", got)
	}
	beyond.AddRow(99, 1, "N")
	if got := beyond.ListLine(); len(got) != 2 {
		t.Errorf("AddRow(99) 结果 = %v", got)
	}
	//负row不插入且不破坏数据
	neg := NewTable([]string{"r0"})
	neg.AddRow(-1, 1, "N")
	if got := neg.ListLine(); len(got) != 1 || got[0][0] != "r0" {
		t.Errorf("AddRow 负row = %v", got)
	}
	//负rowspan曾panic(makeslice: len out of range)：
	//本类型其余写入方法对非法下标一律忽略，此处须夹紧为0即插入空行，不得崩溃
	negSpan := NewTable([]string{"r0"}, []string{"r1"})
	negSpan.AddRow(1, -3, "N")
	got := negSpan.ListLine()
	if len(got) != 3 {
		t.Fatalf("AddRow 负rowspan 行数 = %d, 期望 3: %v", len(got), got)
	}
	if len(got[1]) != 0 {
		t.Errorf("AddRow 负rowspan 应插入空行, got %v", got[1])
	}
	//原有行内容不得被破坏
	if got[0][0] != "r0" || got[2][0] != "r1" {
		t.Errorf("AddRow 负rowspan 破坏了原有行: %v", got)
	}
	//与 rowspan=0 的结果必须一致（负值语义上等价于0）
	zeroSpan := NewTable([]string{"r0"}, []string{"r1"})
	zeroSpan.AddRow(1, 0, "N")
	if zeroSpan.String() != negSpan.String() {
		t.Errorf("AddRow 负rowspan(%s) 与 rowspan=0(%s) 结果不一致", negSpan.String(), zeroSpan.String())
	}
}

func TestTableRmRow(t *testing.T) {
	tb := NewTable([]string{"r0"}, []string{"r1"}, []string{"r2"})
	tb.RmRow(1)
	lines := tb.ListLine()
	if len(lines) != 2 || lines[0][0] != "r0" || lines[1][0] != "r2" {
		t.Errorf("RmRow(1) = %v, 期望 [[r0] [r2]]", lines)
	}
	//删首行
	head := NewTable([]string{"r0"}, []string{"r1"})
	head.RmRow(0)
	if got := head.ListLine(); len(got) != 1 || got[0][0] != "r1" {
		t.Errorf("RmRow(0) = %v", got)
	}
	//删末行
	tail := NewTable([]string{"r0"}, []string{"r1"})
	tail.RmRow(1)
	if got := tail.ListLine(); len(got) != 1 || got[0][0] != "r0" {
		t.Errorf("RmRow(末行) = %v", got)
	}
	//越界与负值均为无操作
	oob := NewTable([]string{"r0"}, []string{"r1"})
	oob.RmRow(9)
	oob.RmRow(-1)
	if got := oob.ListLine(); len(got) != 2 {
		t.Errorf("RmRow 越界/负值应无操作, got %v", got)
	}
	//删空后为空表
	one := NewTable([]string{"only"})
	one.RmRow(0)
	if !one.IsEmpty() || len(one.ListLine()) != 0 {
		t.Errorf("删除唯一行后应为空表: %v", one.ListLine())
	}
	//空表删除不panic
	NewTable().RmRow(0)
}

func TestTableRender(t *testing.T) {
	//空表渲染为空串
	if got := NewTable().Render(); got != "" {
		t.Errorf("空表 Render = %q", got)
	}
	//单行
	want := "+---+---+\n| a | b |\n+---+---+"
	if got := NewTable([]string{"a", "b"}).Render(); got != want {
		t.Errorf("Render 单行 = %q, 期望 %q", got, want)
	}
	//不等长行：缺失处补空白，列数按最长行对齐
	want = "+---+---+\n| a | b |\n| c |   |\n+---+---+"
	if got := NewTable([]string{"a", "b"}, []string{"c"}).Render(); got != want {
		t.Errorf("Render ragged = %q, 期望 %q", got, want)
	}
	//nil空洞渲染为空白而非"null"
	sparse := NewTable()
	sparse.SetCell(0, 2, "x")
	got := sparse.Render()
	want = "+--+--+---+\n|  |  | x |\n+--+--+---+"
	if got != want {
		t.Errorf("Render 稀疏表 = %q, 期望 %q", got, want)
	}
	//Render 是只读操作，不能改动表格自身
	tb := NewTable([]string{"a", "b"}, []string{"c"})
	before := tb.String()
	tb.Render()
	if after := tb.String(); before != after {
		t.Errorf("Render 修改了表格状态: %s -> %s", before, after)
	}
}
