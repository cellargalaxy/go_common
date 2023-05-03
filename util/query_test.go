package util

import "testing"

func TestQuery(t *testing.T) {
	ctx := GenCtx()

	type Query struct {
		Id   int    `query:"id"`
		Name string `query:"name"`
	}

	var list []Query
	var ll []Query

	list = append(list, Query{Id: 1, Name: "a"})
	list = append(list, Query{Id: 2, Name: "b"})

	text := QueryStruct2String(ctx, list)
	err := QueryString2Struct(ctx, text, &ll)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if len(ll) != 2 || ll[0].Id != 1 || ll[0].Name != "a" || ll[1].Id != 2 || ll[1].Name != "b" {
		t.Errorf(`len(ll) != 2`)
		return
	}
}
