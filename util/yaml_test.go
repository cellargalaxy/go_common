package util

import "testing"

func TestYaml(t *testing.T) {
	ctx := GenCtx()

	type Yaml struct {
		Id   int    `yaml:"id"`
		Name string `yaml:"name"`
	}

	var list []Yaml
	var ll []Yaml

	list = append(list, Yaml{Id: 1, Name: "a"})
	list = append(list, Yaml{Id: 2, Name: "b"})

	text := YamlStruct2String(ctx, list)
	err := YamlString2Struct(ctx, text, &ll)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if len(ll) != 2 || ll[0].Id != 1 || ll[0].Name != "a" || ll[1].Id != 2 || ll[1].Name != "b" {
		t.Errorf(`len(ll) != 2`)
		return
	}
}
