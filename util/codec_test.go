package util

import (
	"github.com/cellargalaxy/go_common/model"
	"testing"
	"time"
)

func TestGzip(t *testing.T) {
	ctx := GenCtx()

	data, err := EnGzip(ctx, []byte("aaa"))
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	data, err = DeGzip(ctx, data)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if string(data) != "aaa" {
		t.Errorf(`if string(data) != "aaa" {`)
		return
	}
}

func TestBase64(t *testing.T) {
	ctx := GenCtx()

	data := EnBase64(ctx, []byte("aaa"))
	value := DeBase64(ctx, data)
	if string(value) != "aaa" {
		t.Errorf(`if string(value) != "aaa" {`)
		return
	}
}

func TestJwt(t *testing.T) {
	ctx := GenCtx()

	expire := time.Hour
	now := time.Now()
	var claims model.Claims
	claims.IssuedAt = now.Add(-expire).Unix()
	claims.ExpiresAt = now.Add(expire).Unix()
	claims.Ip = "GetIp()"
	claims.ServerName = "GetServerName()"
	claims.LogId = 123456
	claims.ReqId = "GetOrGenReqIdString(ctx)"

	jwt, err := EnJwt(ctx, "secret", claims)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}

	var ccc model.Claims
	token, err := DeJwt(ctx, jwt, "secret", &ccc)
	t.Log(JsonStruct2String(ccc))
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if token == nil {
		t.Errorf(`if token == nil {`)
		return
	}
	if !token.Valid {
		t.Errorf(`if !token.Valid {`)
		return
	}
	if ccc.Ip != "GetIp()" {
		t.Errorf(`if ccc.Ip != "GetIp()" {`)
		return
	}
	if ccc.ServerName != "GetServerName()" {
		t.Errorf(`if ccc.ServerName != "GetServerName()" {`)
		return
	}
	if ccc.LogId != 123456 {
		t.Errorf(`if ccc.LogId != 123456 {`)
		return
	}
	if ccc.ReqId != "GetOrGenReqIdString(ctx)" {
		t.Errorf(`if ccc.ReqId != "GetOrGenReqIdString(ctx)" {`)
		return
	}
}

func TestAesCbc(t *testing.T) {
	ctx := GenCtx()

	value, err := EnAesCbcString(ctx, "aaa", "bbb")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	value, err = DeAesCbcString(ctx, value, "bbb")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if value != "aaa" {
		t.Errorf(`if value != "aaa" {`)
		return
	}
}

func TestHash(t *testing.T) {
	if EnSha256Hex("123456") != "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92" {
		t.Errorf(`EnSha256Hex("123456") %+v`, EnSha256Hex("123456"))
		return
	}
	if EnMd5Hex("123456") != "e10adc3949ba59abbe56e057f20f883e" {
		t.Errorf(`EnMd5Hex("123456") %+v`, EnMd5Hex("123456"))
		return
	}
	if EnCrc32Hex("123456") != "972d361" {
		t.Errorf(`EnCrc32Hex("123456") %+v`, EnCrc32Hex("123456"))
		return
	}
}

func TestRsa1(t *testing.T) {
	ctx := GenCtx()

	sign, err := RsaSignString(ctx, `aaa`, `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	ok, err := RsaVerifyString(ctx, `aaa`, sign, `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAqkk9WHSyUDdbq1oIm9gS
dOxGTXE4Tx8OW2O55We1LMdMCvc2YQRrGmQctvzTsRA9+ECrgL+DaWPbcNbvcofy
VN1r4Q3zeEkHejbrOVJTnvTnPljySiNSKO1nNNInBSpjeRh9UbPZdpyCD1bNZ93Q
DcQEeQTcccouDclljVAbOJX5USWnAiYWe4h/TkT+mvfCxh1z6pckHL9O56sXb5WP
U76E0RXctZQcssYB9kWGRqHLFwYTfo3NQaRhKg+SGueqBXZrHTVbit2h/wotnaox
tLMV94O5KQLAi/jsLvO+vq6VlLBjSRsQHBan+MmhZ1XcIZvMKyI/dVlQLMycX2nc
Xd1neB++GNeb3P3fbfMwv9pwS7pt9pm0zO4zipyBhhzznOwzvIrkRsFHUg2KHJqm
J9Yh+a+MCajYpeq+Iq9tPJEyb6RrOCitvt8MG/JN5N4cba5Uyz4Si8ArImSIUJpj
1bMwmb0cLwF8HNz1GJtojxwxRpe77pt9PdpfrPA7HOl3uEOJyyNAs0qu0TIB7c4O
FlPzQGFtIFox6H/LZ6sVbFxZkXPQoeuxl2nm3dfoEjRJYJq2ffTG6CWDuAi0jFul
xOGUIizG5oRMOXLdUx6mDsgyknIbelpccbRz4kYv/A6owVktTyw4XJw1MLCTYIrr
0qSIvAAO7lGVhpzsUAmjcBcCAwEAAQ==
-----END PUBLIC KEY-----
`)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if !ok {
		t.Errorf(`if !ok {`)
		return
	}
}

func TestRsa2(t *testing.T) {
	ctx := GenCtx()

	sign, err := RsaSignString(ctx, `aaa`, `-----BEGIN PRIVATE KEY-----
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
+9KYg5QnoTzqFp+BL8HlaVOP5VbEz3KP
-----END PRIVATE KEY-----
`)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	ok, err := RsaVerifyString(ctx, `aaa`, sign, `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAqkk9WHSyUDdbq1oIm9gS
dOxGTXE4Tx8OW2O55We1LMdMCvc2YQRrGmQctvzTsRA9+ECrgL+DaWPbcNbvcofy
VN1r4Q3zeEkHejbrOVJTnvTnPljySiNSKO1nNNInBSpjeRh9UbPZdpyCD1bNZ93Q
DcQEeQTcccouDclljVAbOJX5USWnAiYWe4h/TkT+mvfCxh1z6pckHL9O56sXb5WP
U76E0RXctZQcssYB9kWGRqHLFwYTfo3NQaRhKg+SGueqBXZrHTVbit2h/wotnaox
tLMV94O5KQLAi/jsLvO+vq6VlLBjSRsQHBan+MmhZ1XcIZvMKyI/dVlQLMycX2nc
Xd1neB++GNeb3P3fbfMwv9pwS7pt9pm0zO4zipyBhhzznOwzvIrkRsFHUg2KHJqm
J9Yh+a+MCajYpeq+Iq9tPJEyb6RrOCitvt8MG/JN5N4cba5Uyz4Si8ArImSIUJpj
1bMwmb0cLwF8HNz1GJtojxwxRpe77pt9PdpfrPA7HOl3uEOJyyNAs0qu0TIB7c4O
FlPzQGFtIFox6H/LZ6sVbFxZkXPQoeuxl2nm3dfoEjRJYJq2ffTG6CWDuAi0jFul
xOGUIizG5oRMOXLdUx6mDsgyknIbelpccbRz4kYv/A6owVktTyw4XJw1MLCTYIrr
0qSIvAAO7lGVhpzsUAmjcBcCAwEAAQ==
-----END PUBLIC KEY-----
`)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if !ok {
		t.Errorf(`if !ok {`)
		return
	}
}
