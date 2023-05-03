package util

import "testing"

func TestJson(t *testing.T) {
	type Json struct {
		Id   int    `csv:"id"`
		Name string `csv:"name"`
	}

	var list []Json
	var ll []Json

	list = append(list, Json{Id: 1, Name: "a"})
	list = append(list, Json{Id: 2, Name: "b"})

	text := JsonStruct2String(list)
	err := JsonString2Struct(text, &ll)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if len(ll) != 2 || ll[0].Id != 1 || ll[0].Name != "a" || ll[1].Id != 2 || ll[1].Name != "b" {
		t.Errorf(`len(ll) != 2`)
		return
	}
}
