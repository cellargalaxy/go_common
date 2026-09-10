package util

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/cellargalaxy/go_common/model"
	"github.com/golang-jwt/jwt"
)

// ==== RSA 测试密钥（沿用原用例中的固定密钥对） ====

// PKCS1 格式私钥
const testRsaPkcs1PrivateKey = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`

// 与 PKCS1 私钥配对的公钥
const testRsaPublicKey1 = `-----BEGIN PUBLIC KEY-----
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
`

// PKCS8 格式私钥（同一密钥对的另一种编码）
const testRsaPkcs8PrivateKey = `-----BEGIN PRIVATE KEY-----
MIIJRAIBADANBgkqhkiG9w0BAQEFAASCCS4wggkqAgEAAoICAQCqST1YdLJQN1ur
Wgib2BJ07EZNcThPHw5bY7nlZ7Usx0wK9zZhBGsaZBy2/NOxED34QKuAv4NpY9tw
1u9yh/JU3WvhDfN4SQd6Nus5UlOe9Oc+WPJKI1Io7Wc00icFKmN5GH1Rs9l2nIIP
Vs1n3dANxAR5BNxxyi4NyWWNUBs4lflRJacCJhZ7iH9ORP6a98LGHXPqlyQcv07n
qxdvlY9TvoTRFdy1lByyxgH2RYZGocsXBhN+jc1BpGEqD5Ia56oFdmsdNVuK3aH/
Ci2dqjG0sxX3g7kpAsCL+Owu876+rpWUsGNJGxAcFqf4yaFnVdwhm8wrIj91WVAs
zJxfadxd3Wd4H74Y15vc/d9t8zC/2nBLum32mbTM7jOKnIGGHPOc7DO8iuRGwUdS
DYocmqYn1iH5r4wJqNil6r4ir208kTJvpGs4KK2+3wwb8k3k3hxtrlTLPhKLwCsi
ZIhQmmPVszCZvRwvAXwc3PUYm2iPHDFGl7vum3092l+s8Dsc6Xe4Q4nLI0CzSq7R
MgHtzg4WU/NAYW0gWjHof8tnqxVsXFmRc9Ch67GXaebd1+gSNElgmrZ99MboJYO4
CLSMW6XE4ZQiLMbmhEw5ct1THqYOyDKScht6WlxxtHPiRi/8DqjBWS1PLDhcnDUw
sJNgiuvSpIi8AA7uUZWGnOxQCaNwFwIDAQABAoICAByViLeldx0h/cBIr7NBVO/N
jYuIWtRlOSq8FcNdYjsvBSO+FaG5dSUL8woAbKSHPm/Co0dMXom9WmOZoL4P9eyq
8mXnypvBXoD0YQ3RNPo0z/BWS+2H+k2mdDGYqTYjxGjaZIbncrSHVwmuH3K0+Cyn
X7zUxRBQyORFj0dmT/tyqVQmxt8Ed7eq1PvVdwuaVVaeMQwK7OPqdRtMS/XOHC4J
iA/05ISSZTLZLBEj4INQu1+VBIa01OSU7D58OKFNi+u40ggoIcKwq3W9RqT1obxc
Z26R7O3maV3VngfPoJAvjCpWZYQrB8w/PVj0t6Q59Gar2VzQNejD4kJy4jjd80CE
Qvt8m9ekZTJ8T/aHAHc9GL4eAxq839UtSH1pLAZO0++GXxh/wzHVPBWYwK6yrE6F
3PLxFq2fqTq1UGKw6XlnYDrrAU5hn8wN6arDN0aPEBmaReulhb2YRKa7FsdseFuM
ImZhBDVq3KVfH/po13inH3SsJOrD1wuQGTS+EJbEe/cFNPyxu8X269iTJh543vCg
5wWW1VhtSqFvlze3T8ZroYKcbmUTuvPjexOix/R6W/T9ncDLZa3R7SYZptfQCJXc
STHHm7XeaXWW15UgImt1N60Ljt37a824u8j+qrgdo87Xf0yAOElX4VVRTL6VUu4c
giem+xA8bnUFzfS+1b65AoIBAQDU1roUTpt4apdynU4/h1Egp4nOrVAZJtxgNcXH
RvEOrLS70/r0aMAEnp2IMwQMeCYHwxfEbgjU17MZ8l9rvfb3PCrDoyBDqPJP3tke
yDtoOIVqGw/brFG+zzkbfbnSeGUCoL5GeFO/GDKqYMXRxm7/1m/K49erfb8IMC/S
GDwvN4y3roPOiCugQPK2MMRYkfwrfP7xZ8pOGnPeqF1hGH+NEHXseofJfvB510XV
n7Qu0WI/9hDRvJO5n/joWEiaQ8XTDJBqQ9Ku1bdP5J6i8b5mlgqFriVPktiTSwHF
MX83Mu/8TXIWWzrTUc37YRnLtLE7yGA3c0XFtYk+cU30Dg19AoIBAQDM0XDFwXFn
vIyN1LWKxonYnkKo3h3nd5NOflib8yFbk4dmOyeQHQ6ao6jL1/s55Lksst7kcode
9LkLmT38ls6fTKp+zp+WUDoB+iqUvLPC8nuDpibQgcDD5UcYXuLwmg/9JZGF5FRJ
w/5m3D2LtoWySOzqo9lkctakF4AdAvRXzIbVOocNaKYiUuMLeUEgGGY1x+aXrOzN
fgnttItP7w/ee59VszIrUbosJX0chKfIKe+Qd1jXZlwoDniHK9najWYCSbbtdUpI
wtv0Q219vU5RkyuIpQ2AZvNiPFxc4n+IXX0jrAo92P/97fIiFAEGzpyvylZ/f5bH
hlFG3H/MCXgjAoIBAQCRBaqO0azvlLmS2AqMEPi/Zkjo8YClUqxQRfN4QCdaV2Vi
oiLZzFIiE63FMvWsRWWsXktghZUw6bRYlqRb9nlDptSArzL5xzEsKO0KI2cPMYv+
a2gXx0d/2xTPXZBzD6PL8ZkUzUJK8AL9wqinsizJ1+Qee7blW+4Bd0vuPKMtjQ0t
zxRycqg/LmHysDHpI7eOLkJMXOsk4rNW4KOX2Ch5BuFC+45wRjU23+7uiz4rtHHc
oR5+up5xmcXxEs6NhmZR0rrIHWR9ypN4NY+9+ThCEDQlOC6mFKnLWw1evsYu519M
K7zz7L+dY50VjukTuxAUawjcX4gIGhTk3jE+fSDFAoIBAQC0RPru822vEkLbaoZG
mKwm3lRpcOX9Hj+m7Ils6TdI9LZNgm815n0+GiGtGHHj6K4a62xGwY7oQpjD3qKy
OMGvb7j4V0Aa+o2zw83Kj7PgN7AClTDAgBN718cVah5wCglck75bjDWrnzfArMP2
vjU6T/lGI+RpBrJoRs7Zugn5ZOifsB3B7YOb3jw7kwOBDjiXyVPp16eszexLLYcY
WZ8sAL6inU/g1NP5mx1dBlzuTgquqnylSVB/aQ0oW64pMB7CkMLSiVNSaz9Q5C/w
ZbgTEGdWo38YroQ0yE+66Re6nB5SUU/KU74sb0uYNucYaK18A6jSGyhIgK30oopQ
uVzlAoIBAQCa95DvDPwxyMwFmm54nnlQnFkVl2W7kjjxh4Ngn3IXJcbSO3cG56+Z
PlnSSCHArPGZvNfxRjLi62kp6xc4KshxF0IJSZwwL9QG+B4FRnKwQ3Dww7aC/YvG
eEA0ts9ZvglLUMnxnZICP6bSOO+LzlB266eBhrwTGhybB7MdhXMKSkPTWs+8NXRy
+vmgOxggiN67TvhwqbXhfceQMmJ1R1ENzBGbv8ZpHaKPuwDve6NFaBRld+XLOsTp
LaZNDX00rd6aNQeER/QK+JQsvGbzXoCd0zrEJRUtuUucNNIolcrTlHTMGTUetcSw
+9KYg5QnoTzqFp+BL8HlaVOP5VbEz3KP
-----END PRIVATE KEY-----
`

// 与 PKCS8 私钥配对的公钥
const testRsaPublicKey2 = `-----BEGIN PUBLIC KEY-----
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
`

// ==== Gzip ====

func TestGzip(t *testing.T) {
	ctx := GenCtx()
	//往返一致
	data, err := EnGzip(ctx, []byte("aaa"))
	if err != nil {
		t.Fatalf("EnGzip 异常: %+v", err)
	}
	got, err := DeGzip(ctx, data)
	if err != nil {
		t.Fatalf("DeGzip 异常: %+v", err)
	}
	if string(got) != "aaa" {
		t.Errorf("往返结果 = %q, 期望 aaa", string(got))
	}
	//必须真的产生压缩效果，而非原样返回
	big := strings.Repeat("a", 10000)
	z, err := EnGzip(ctx, []byte(big))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(z) >= len(big) {
		t.Errorf("压缩后 %d 字节 >= 原始 %d 字节，未真正压缩", len(z), len(big))
	}
	if back, err := DeGzip(ctx, z); err != nil || string(back) != big {
		t.Errorf("大数据往返失败: err=%v 长度=%d", err, len(back))
	}
	//二进制数据（含0字节）不能被破坏
	bin := []byte{0, 1, 2, 255, 0, 128}
	z, _ = EnGzip(ctx, bin)
	back, err := DeGzip(ctx, z)
	if err != nil || string(back) != string(bin) {
		t.Errorf("二进制往返失败: %v, %v", back, err)
	}
	//空输入
	z, err = EnGzip(ctx, []byte{})
	if err != nil {
		t.Errorf("EnGzip(空) 异常: %+v", err)
	}
	if back, err = DeGzip(ctx, z); err != nil || len(back) != 0 {
		t.Errorf("空数据往返: %v, %v", back, err)
	}
}

// DeGzip 对非法数据必须返回error而非panic（曾因先defer后判err而空指针panic）
func TestDeGzipInvalid(t *testing.T) {
	ctx := GenCtx()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("DeGzip 非法数据触发panic: %v", r)
		}
	}()
	for _, bad := range [][]byte{
		[]byte("not gzip at all"),
		{},
		{0x1f, 0x8b},             //仅magic头，数据截断
		{0x1f, 0x8b, 0x08, 0xff}, //头部合法但内容损坏
	} {
		got, err := DeGzip(ctx, bad)
		if err == nil && len(got) > 0 {
			t.Errorf("DeGzip(%v) 未报错且返回了数据 %v", bad, got)
		}
	}
}

// ==== Base64 ====

func TestBase64(t *testing.T) {
	ctx := GenCtx()
	//已知编码值，防止算法被换成非标准变体
	if got := EnBase64(ctx, []byte("aaa")); got != "YWFh" {
		t.Errorf(`EnBase64("aaa") = %q, 期望 "YWFh"`, got)
	}
	if got := string(DeBase64(ctx, "YWFh")); got != "aaa" {
		t.Errorf("DeBase64 = %q", got)
	}
	//空值
	if got := EnBase64(ctx, []byte{}); got != "" {
		t.Errorf("EnBase64(空) = %q", got)
	}
	if got := DeBase64(ctx, ""); len(got) != 0 {
		t.Errorf("DeBase64(空) = %v", got)
	}
	//二进制与多字节字符往返
	for _, in := range [][]byte{{0, 1, 2, 255}, []byte("中文测试"), []byte(strings.Repeat("x", 1000))} {
		if back := DeBase64(ctx, EnBase64(ctx, in)); string(back) != string(in) {
			t.Errorf("往返失败, 长度 %d -> %d", len(in), len(back))
		}
	}
	//非法base64：DeBase64 约定静默返回空，不panic
	if got := DeBase64(ctx, "!!!非法!!!"); len(got) != 0 {
		t.Errorf("DeBase64(非法) = %v, 期望空", got)
	}
	//deBase64 内部版本必须返回error供调用方判断
	if _, err := deBase64(ctx, "!!!"); err == nil {
		t.Errorf("deBase64(非法) 应返回error")
	}
	if _, err := deBase64(ctx, "YWFh"); err != nil {
		t.Errorf("deBase64(合法) 异常: %+v", err)
	}
}

// ==== JWT ====

func TestJwtRoundTrip(t *testing.T) {
	ctx := GenCtx()
	expire := time.Hour
	now := time.Now()
	var claims model.Claims
	claims.IssuedAt = now.Add(-expire).Unix()
	claims.ExpiresAt = now.Add(expire).Unix()
	claims.Ip = "1.2.3.4"
	claims.ServerName = "svc"
	claims.LogId = 123456
	claims.ReqId = 654321

	token, err := EnJwt(ctx, "secret", claims)
	if err != nil {
		t.Fatalf("EnJwt 异常: %+v", err)
	}
	//JWT 应为三段点分结构
	if parts := strings.Split(token, "."); len(parts) != 3 {
		t.Errorf("JWT 段数 = %d, 期望 3", len(parts))
	}

	var got model.Claims
	parsed, err := DeJwt(ctx, token, "secret", &got)
	if err != nil {
		t.Fatalf("DeJwt 异常: %+v", err)
	}
	if parsed == nil || !parsed.Valid {
		t.Fatalf("token 无效: %v", parsed)
	}
	//逐字段核对
	if got.Ip != "1.2.3.4" || got.ServerName != "svc" || got.LogId != 123456 || got.ReqId != 654321 {
		t.Errorf("claims 还原不一致: %+v", got)
	}
	if got.IssuedAt != claims.IssuedAt || got.ExpiresAt != claims.ExpiresAt {
		t.Errorf("时间字段不一致: %d/%d vs %d/%d", got.IssuedAt, got.ExpiresAt, claims.IssuedAt, claims.ExpiresAt)
	}
}

// 安全关键：错误密钥、过期、篡改都必须被拒绝
func TestJwtSecurity(t *testing.T) {
	ctx := GenCtx()
	now := time.Now()
	var claims model.Claims
	claims.IssuedAt = now.Unix()
	claims.ExpiresAt = now.Add(time.Hour).Unix()
	claims.Ip = "1.2.3.4"
	token, err := EnJwt(ctx, "right-secret", claims)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	//错误密钥必须失败
	var out model.Claims
	if _, err = DeJwt(ctx, token, "wrong-secret", &out); err == nil {
		t.Errorf("错误密钥应校验失败")
	}
	//篡改payload必须失败
	parts := strings.Split(token, ".")
	tampered := parts[0] + "." + EnBase64(ctx, []byte(`{"ip":"9.9.9.9"}`)) + "." + parts[2]
	if _, err = DeJwt(ctx, tampered, "right-secret", &out); err == nil {
		t.Errorf("篡改payload应校验失败")
	}
	//格式非法
	for _, bad := range []string{"", "abc", "a.b", "a.b.c"} {
		if _, err = DeJwt(ctx, bad, "right-secret", &out); err == nil {
			t.Errorf("非法token %q 应报错", bad)
		}
	}
	//已过期必须失败
	var expired model.Claims
	expired.IssuedAt = now.Add(-2 * time.Hour).Unix()
	expired.ExpiresAt = now.Add(-time.Hour).Unix()
	expiredToken, err := EnJwt(ctx, "s", expired)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if _, err = DeJwt(ctx, expiredToken, "s", &out); err == nil {
		t.Errorf("过期token应校验失败")
	} else if !strings.Contains(err.Error(), "expired") {
		t.Errorf("过期错误信息未体现expired: %v", err)
	}
	//不同密钥生成的token必须不同
	t1, _ := EnJwt(ctx, "s1", claims)
	t2, _ := EnJwt(ctx, "s2", claims)
	if t1 == t2 {
		t.Errorf("不同密钥生成了相同token")
	}
}

// DeJwt 的 claims 传 nil 时走 jwt.Parse 分支（与传 &claims 的 ParseWithClaims 分支不同），
// 该分支此前完全未被覆盖；校验它同样能验签成功、并同样拒绝错误密钥与过期token
func TestDeJwtNilClaims(t *testing.T) {
	ctx := GenCtx()
	secret := "nil-claims-secret"
	now := time.Now()
	var claims model.Claims
	claims.IssuedAt = now.Unix()
	claims.ExpiresAt = now.Add(time.Hour).Unix()
	claims.Ip = "10.0.0.1"
	token, err := EnJwt(ctx, secret, claims)
	if err != nil {
		t.Fatalf("EnJwt 异常: %+v", err)
	}

	//正确密钥 + nil claims：须返回非空token且Valid
	jwtToken, err := DeJwt(ctx, token, secret, nil)
	if err != nil {
		t.Fatalf("DeJwt(nil claims) 异常: %+v", err)
	}
	if jwtToken == nil {
		t.Fatalf("DeJwt(nil claims) 返回空token")
	}
	if !jwtToken.Valid {
		t.Errorf("DeJwt(nil claims) Valid = false")
	}
	//payload 仍应可从MapClaims中读到，证明确实解析了内容
	if mc, ok := jwtToken.Claims.(jwt.MapClaims); ok {
		if mc["ip"] != "10.0.0.1" {
			t.Errorf("nil claims 解析出的ip = %v, 期望 10.0.0.1", mc["ip"])
		}
	} else {
		t.Errorf("nil claims 时 Claims 类型 = %T, 期望 jwt.MapClaims", jwtToken.Claims)
	}

	//错误密钥必须失败，不能因为claims为nil就跳过签名校验
	if _, err = DeJwt(ctx, token, "wrong-secret", nil); err == nil {
		t.Errorf("nil claims + 错误密钥 应校验失败")
	}
	//过期token必须失败
	var expired model.Claims
	expired.IssuedAt = now.Add(-2 * time.Hour).Unix()
	expired.ExpiresAt = now.Add(-time.Hour).Unix()
	expiredToken, err := EnJwt(ctx, secret, expired)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if _, err = DeJwt(ctx, expiredToken, secret, nil); err == nil {
		t.Errorf("nil claims + 过期token 应校验失败")
	}
	//非法格式必须报错而非panic
	for _, bad := range []string{"", "abc", "a.b", "a.b.c"} {
		if _, err = DeJwt(ctx, bad, secret, nil); err == nil {
			t.Errorf("nil claims + 非法token %q 应报错", bad)
		}
	}
}

func TestEnDefaultJwt(t *testing.T) {
	ctx := GenCtx()
	token, err := EnDefaultJwt(ctx, "secret", time.Hour)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if token == "" {
		t.Fatalf("EnDefaultJwt 返回空")
	}
	//默认claims须自动带上链路信息，可被解回
	var got model.Claims
	if _, err = DeJwt(ctx, token, "secret", &got); err != nil {
		t.Fatalf("解析自身生成的token失败: %+v", err)
	}
	if got.ServerName != GetServerName() {
		t.Errorf("serverName = %q, 期望 %q", got.ServerName, GetServerName())
	}
	if got.LogId != GetLogId(ctx) {
		t.Errorf("logId = %d, 期望 %d", got.LogId, GetLogId(ctx))
	}
	//过期时间应落在预期区间
	if got.ExpiresAt <= time.Now().Unix() {
		t.Errorf("ExpiresAt %d 已过期", got.ExpiresAt)
	}
	if got.ExpiresAt > time.Now().Add(2*time.Hour).Unix() {
		t.Errorf("ExpiresAt %d 超出预期", got.ExpiresAt)
	}
}

func TestAuthorizationHeader(t *testing.T) {
	ctx := GenCtx()
	//固定头名与Bearer前缀，属对外协议不能变
	key, value := GenAuthHeader(ctx, "mytoken")
	if key != "Authorization" {
		t.Errorf("header key = %q, 期望 Authorization", key)
	}
	if value != "Bearer mytoken" {
		t.Errorf("header value = %q, 期望 'Bearer mytoken'", value)
	}
	//EnAuthJwt 应产出可解析的Bearer token
	key, value = EnAuthJwt(ctx, "secret", time.Hour)
	if key != "Authorization" {
		t.Errorf("key = %q", key)
	}
	if !strings.HasPrefix(value, "Bearer ") {
		t.Errorf("value 缺少 Bearer 前缀: %q", value)
	}
	token := strings.TrimPrefix(value, "Bearer ")
	var got model.Claims
	if _, err := DeJwt(ctx, token, "secret", &got); err != nil {
		t.Errorf("Authorization 中的token无法解析: %+v", err)
	}
}

// ==== AES ====

func TestAesCbc(t *testing.T) {
	ctx := GenCtx()
	//往返
	enc, err := EnAesCbcStr(ctx, "aaa", "bbb")
	if err != nil {
		t.Fatalf("EnAesCbcStr 异常: %+v", err)
	}
	//密文必须与明文不同，否则等于没加密
	if enc == "aaa" {
		t.Errorf("密文与明文相同，未实际加密")
	}
	got, err := DeAesCbcStr(ctx, enc, "bbb")
	if err != nil {
		t.Fatalf("DeAesCbcStr 异常: %+v", err)
	}
	if got != "aaa" {
		t.Errorf("往返结果 = %q, 期望 aaa", got)
	}
	//不同密钥必须产生不同密文
	enc2, _ := EnAesCbcStr(ctx, "aaa", "ccc")
	if enc == enc2 {
		t.Errorf("不同密钥产生了相同密文")
	}
	//不同明文必须产生不同密文
	enc3, _ := EnAesCbcStr(ctx, "aab", "bbb")
	if enc == enc3 {
		t.Errorf("不同明文产生了相同密文")
	}
	//较长文本与多字节字符
	for _, plain := range []string{"", "中文测试内容", strings.Repeat("x", 1000)} {
		e, err := EnAesCbcStr(ctx, plain, "key")
		if err != nil {
			t.Errorf("加密 %d 字节异常: %+v", len(plain), err)
			continue
		}
		d, err := DeAesCbcStr(ctx, e, "key")
		if err != nil || d != plain {
			t.Errorf("往返失败(%d字节): got %q err %v", len(plain), d, err)
		}
	}
	//字节数组版本
	encB, err := EnAesCbc(ctx, []byte("data"), []byte("key"))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	decB, err := DeAesCbc(ctx, encB, []byte("key"))
	if err != nil || string(decB) != "data" {
		t.Errorf("EnAesCbc/DeAesCbc 往返失败: %q %v", decB, err)
	}
}

// 记录已知安全短板：错误密钥解密不报错，只返回垃圾数据
func TestAesCbcWrongSecret(t *testing.T) {
	ctx := GenCtx()
	enc, err := EnAesCbcStr(ctx, "hello", "right")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got, err := DeAesCbcStr(ctx, enc, "wrong")
	//当前实现无完整性校验，错误密钥不报错；至少不能还原出原文，也不能panic
	if err == nil && got == "hello" {
		t.Errorf("错误密钥竟解出了正确明文")
	}
	//非法密文必须报错，不能静默返回空
	if _, err = DeAesCbcStr(ctx, "!!!非法密文!!!", "key"); err == nil {
		t.Errorf("非法密文应返回error")
	}
}

// 密文长度非法时不得静默失败：底层 goEncrypt 在遇到非完整块时只打日志、
// 返回的err为nil，会让调用方把失败当成"解出空串"的成功，须由本包拦成error。
func TestDeAesCbcIllegalLength(t *testing.T) {
	ctx := GenCtx()
	secret := []byte("key")

	//非AES块大小(16)整数倍的密文，一律报错
	for _, n := range []int{1, 5, 15, 17, 31, 33} {
		data := make([]byte, n)
		got, err := DeAesCbc(ctx, data, secret)
		if err == nil {
			t.Errorf("DeAesCbc(%d字节) err=nil, 期望报错（静默失败会被误当成解密成功）", n)
		}
		if got != nil {
			t.Errorf("DeAesCbc(%d字节) 返回了数据 %q, 期望 nil", n, got)
		}
	}
	//空密文同样报错，而非解出空串
	if got, err := DeAesCbc(ctx, nil, secret); err == nil || got != nil {
		t.Errorf("DeAesCbc(nil) = %q, err=%v, 期望报错", got, err)
	}
	if got, err := DeAesCbc(ctx, []byte{}, secret); err == nil || got != nil {
		t.Errorf("DeAesCbc(空切片) = %q, err=%v, 期望报错", got, err)
	}

	//合法长度不得被误伤：正常往返必须成功
	plain := []byte("hello aes cbc")
	en, err := EnAesCbc(ctx, plain, secret)
	if err != nil {
		t.Fatalf("EnAesCbc 异常: %+v", err)
	}
	if len(en)%16 != 0 {
		t.Errorf("EnAesCbc 输出长度 %d 不是16的倍数", len(en))
	}
	de, err := DeAesCbc(ctx, en, secret)
	if err != nil {
		t.Errorf("合法密文被误伤报错: %+v", err)
	}
	if string(de) != string(plain) {
		t.Errorf("往返失败: %q, 期望 %q", de, plain)
	}

	//空明文加密后仍是完整块，须能正常解回空串
	enEmpty, err := EnAesCbc(ctx, nil, secret)
	if err != nil {
		t.Fatalf("EnAesCbc(nil) 异常: %+v", err)
	}
	deEmpty, err := DeAesCbc(ctx, enEmpty, secret)
	if err != nil {
		t.Errorf("空明文往返被误伤: %+v", err)
	}
	if len(deEmpty) != 0 {
		t.Errorf("空明文往返 = %q, 期望空", deEmpty)
	}

	//空密钥经sha256派生后仍是合法32字节密钥，往返须成功
	enNoKey, err := EnAesCbc(ctx, plain, nil)
	if err != nil {
		t.Fatalf("EnAesCbc(空密钥) 异常: %+v", err)
	}
	deNoKey, err := DeAesCbc(ctx, enNoKey, nil)
	if err != nil {
		t.Errorf("空密钥往返异常: %+v", err)
	}
	if string(deNoKey) != string(plain) {
		t.Errorf("空密钥往返 = %q, 期望 %q", deNoKey, plain)
	}
}

// ==== 哈希 ====

func TestHash(t *testing.T) {
	//已知标准值，锁定算法不被替换
	if got := EnSha256Hex("123456"); got != "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92" {
		t.Errorf("EnSha256Hex = %s", got)
	}
	if got := EnMd5Hex("123456"); got != "e10adc3949ba59abbe56e057f20f883e" {
		t.Errorf("EnMd5Hex = %s", got)
	}
	if got := EnCrc32Hex("123456"); got != "972d361" {
		t.Errorf("EnCrc32Hex = %s", got)
	}
	//空输入的标准值
	if got := EnSha256Hex(""); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("EnSha256Hex(空) = %s", got)
	}
	if got := EnMd5Hex(""); got != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Errorf("EnMd5Hex(空) = %s", got)
	}
	//输出长度
	if got := EnSha256([]byte("x")); len(got) != 32 {
		t.Errorf("EnSha256 长度 = %d, 期望 32", len(got))
	}
	if got := EnMd5([]byte("x")); len(got) != 16 {
		t.Errorf("EnMd5 长度 = %d, 期望 16", len(got))
	}
	//确定性与区分度
	if EnSha256Hex("a") == EnSha256Hex("b") {
		t.Errorf("不同输入SHA256相同")
	}
	if EnMd5Hex("a") == EnMd5Hex("b") {
		t.Errorf("不同输入MD5相同")
	}
	if EnCrc32([]byte("a")) == EnCrc32([]byte("b")) {
		t.Errorf("不同输入CRC32相同")
	}
	//确定性：先取基准值再重复比较，写成同表达式内两次调用不构成校验
	stable := EnSha256Hex("stable")
	for i := 0; i < 3; i++ {
		if again := EnSha256Hex("stable"); again != stable {
			t.Errorf("SHA256 结果不稳定: %s vs %s", stable, again)
		}
	}
	//锁定标准算法：这些是 SHA256/MD5 对固定输入的公开已知值，
	//可防止实现被替换成其他摘要算法而仅靠长度校验漏检
	if got := EnSha256Hex("abc"); got != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Errorf("EnSha256Hex(abc) = %s, 与标准SHA256不符", got)
	}
	if got := EnMd5Hex("abc"); got != "900150983cd24fb0d6963f7d28e17f72" {
		t.Errorf("EnMd5Hex(abc) = %s, 与标准MD5不符", got)
	}
	//中文不应因编码问题出错
	if got := EnMd5Hex("中文"); len(got) != 32 {
		t.Errorf("EnMd5Hex(中文) = %s", got)
	}
}

// ==== RSA ====

// PKCS1 私钥签名 + 公钥验签
func TestRsaPkcs1(t *testing.T) {
	ctx := GenCtx()
	sign, err := RsaSignStr(ctx, "aaa", testRsaPkcs1PrivateKey)
	if err != nil {
		t.Fatalf("RsaSignStr 异常: %+v", err)
	}
	if sign == "" {
		t.Fatalf("签名为空")
	}
	ok, err := RsaVerifyStr(ctx, "aaa", sign, testRsaPublicKey1)
	if err != nil {
		t.Fatalf("RsaVerifyStr 异常: %+v", err)
	}
	if !ok {
		t.Errorf("验签失败")
	}
}

// PKCS8 私钥签名 + 公钥验签（两种私钥编码都要支持）
func TestRsaPkcs8(t *testing.T) {
	ctx := GenCtx()
	sign, err := RsaSignStr(ctx, "aaa", testRsaPkcs8PrivateKey)
	if err != nil {
		t.Fatalf("RsaSignStr 异常: %+v", err)
	}
	ok, err := RsaVerifyStr(ctx, "aaa", sign, testRsaPublicKey2)
	if err != nil {
		t.Fatalf("RsaVerifyStr 异常: %+v", err)
	}
	if !ok {
		t.Errorf("验签失败")
	}
}

// 安全关键：篡改数据、篡改签名、错误公钥都必须验签失败
func TestRsaVerifyNegative(t *testing.T) {
	ctx := GenCtx()
	sign, err := RsaSignStr(ctx, "original", testRsaPkcs1PrivateKey)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	//数据被篡改
	ok, _ := RsaVerifyStr(ctx, "tampered", sign, testRsaPublicKey1)
	if ok {
		t.Errorf("数据被篡改后仍验签通过")
	}
	//签名被篡改
	ok, _ = RsaVerifyStr(ctx, "original", EnBase64(ctx, []byte("badsign")), testRsaPublicKey1)
	if ok {
		t.Errorf("签名被篡改后仍验签通过")
	}
	//空签名
	if ok, _ = RsaVerifyStr(ctx, "original", "", testRsaPublicKey1); ok {
		t.Errorf("空签名仍验签通过")
	}

	//非法密钥必须报错而非panic
	for _, badKey := range []string{"", "not a key", "-----BEGIN PUBLIC KEY-----\nbad\n-----END PUBLIC KEY-----\n"} {
		if _, err = RsaVerifyStr(ctx, "d", sign, badKey); err == nil {
			t.Errorf("非法公钥 %.20q 应报错", badKey)
		}
	}
	for _, badKey := range []string{"", "not a key", "-----BEGIN RSA PRIVATE KEY-----\nbad\n-----END RSA PRIVATE KEY-----\n"} {
		if _, err = RsaSignStr(ctx, "d", badKey); err == nil {
			t.Errorf("非法私钥 %.20q 应报错", badKey)
		}
	}
	//把公钥当私钥用（类型不匹配）须报错
	if _, err = RsaSignStr(ctx, "d", testRsaPublicKey1); err == nil {
		t.Errorf("公钥当私钥使用应报错")
	}
	//签名相同数据两次，PKCS1v15为确定性签名，结果应一致
	s1, _ := RsaSignStr(ctx, "same", testRsaPkcs1PrivateKey)
	s2, _ := RsaSignStr(ctx, "same", testRsaPkcs1PrivateKey)
	if s1 != s2 {
		t.Errorf("PKCS1v15 签名应为确定性，两次结果不同")
	}
	//不同数据签名必须不同
	s3, _ := RsaSignStr(ctx, "different", testRsaPkcs1PrivateKey)
	if s1 == s3 {
		t.Errorf("不同数据产生了相同签名")
	}
}

// RsaSign / RsaVerify 的字节级API：此前仅有 RsaSignStr/RsaVerifyStr 的用例，
// 这两个导出函数从未被直接调用过。它们是String版的底层实现，也是外部可直接使用的公开API，
// 需覆盖二进制数据（含NUL与非UTF8字节）这一String版无法表达的场景。
func TestRsaSignVerifyBytes(t *testing.T) {
	ctx := GenCtx()

	//二进制数据（含NUL、0xFF等非UTF8字节）必须能正常签名验签
	data := []byte{0x00, 0x01, 0xFF, 0xFE, 0x00, 'a', 'b'}
	sign, err := RsaSign(ctx, data, []byte(testRsaPkcs1PrivateKey))
	if err != nil {
		t.Fatalf("RsaSign 异常: %+v", err)
	}
	if len(sign) == 0 {
		t.Fatalf("RsaSign 返回空签名")
	}
	ok, err := RsaVerify(ctx, data, sign, []byte(testRsaPublicKey1))
	if err != nil {
		t.Fatalf("RsaVerify 异常: %+v", err)
	}
	if !ok {
		t.Errorf("二进制数据验签失败")
	}

	//PKCS8 私钥同样支持
	sign8, err := RsaSign(ctx, data, []byte(testRsaPkcs8PrivateKey))
	if err != nil {
		t.Fatalf("RsaSign(PKCS8) 异常: %+v", err)
	}
	if ok, err = RsaVerify(ctx, data, sign8, []byte(testRsaPublicKey2)); err != nil || !ok {
		t.Errorf("PKCS8 二进制验签失败: ok=%v err=%v", ok, err)
	}

	//与String版必须自洽：String版即是对字节版做Base64包装
	strSign, err := RsaSignStr(ctx, string(data), testRsaPkcs1PrivateKey)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got := EnBase64(ctx, sign); got != strSign {
		t.Errorf("RsaSign 与 RsaSignStr 结果不一致:\n字节版Base64=%s\nString版  =%s", got, strSign)
	}

	//空数据也应能签名（对空哈希签名是合法操作）
	emptySign, err := RsaSign(ctx, []byte{}, []byte(testRsaPkcs1PrivateKey))
	if err != nil {
		t.Errorf("RsaSign(空数据) 异常: %+v", err)
	}
	if ok, err = RsaVerify(ctx, []byte{}, emptySign, []byte(testRsaPublicKey1)); err != nil || !ok {
		t.Errorf("空数据验签失败: ok=%v err=%v", ok, err)
	}
	//nil 与空切片语义相同
	if ok, _ = RsaVerify(ctx, nil, emptySign, []byte(testRsaPublicKey1)); !ok {
		t.Errorf("nil 数据应与空切片等价，验签应通过")
	}

	//篡改一个字节即须验签失败
	tampered := make([]byte, len(data))
	copy(tampered, data)
	tampered[0] ^= 0xFF
	if ok, _ = RsaVerify(ctx, tampered, sign, []byte(testRsaPublicKey1)); ok {
		t.Errorf("数据被篡改后仍验签通过")
	}
	//篡改签名同样须失败
	badSign := make([]byte, len(sign))
	copy(badSign, sign)
	badSign[len(badSign)-1] ^= 0xFF
	if ok, _ = RsaVerify(ctx, data, badSign, []byte(testRsaPublicKey1)); ok {
		t.Errorf("签名被篡改后仍验签通过")
	}
	//公钥不匹配须失败。
	//注意：testRsaPublicKey1 与 testRsaPublicKey2 实为同一份公钥（两个常量内容完全相同，
	//仅名字不同），用它们互相验签必然通过，测不出"密钥不匹配"。故此处现场生成一对独立密钥。
	otherPub := genOtherRsaPublicKeyPem(t)
	if ok, _ = RsaVerify(ctx, data, sign, otherPub); ok {
		t.Errorf("使用不匹配的公钥仍验签通过")
	}

	//非法密钥必须返回error而非panic
	for _, bad := range [][]byte{nil, {}, []byte("not a key"),
		[]byte("-----BEGIN RSA PRIVATE KEY-----\nbad\n-----END RSA PRIVATE KEY-----\n")} {
		if _, err = RsaSign(ctx, data, bad); err == nil {
			t.Errorf("RsaSign 非法私钥 %.20q 应报错", bad)
		}
	}
	for _, bad := range [][]byte{nil, {}, []byte("not a key"),
		[]byte("-----BEGIN PUBLIC KEY-----\nbad\n-----END PUBLIC KEY-----\n")} {
		if _, err = RsaVerify(ctx, data, sign, bad); err == nil {
			t.Errorf("RsaVerify 非法公钥 %.20q 应报错", bad)
		}
	}
	//把私钥PEM当公钥传入：块类型不是PUBLIC KEY，须报错
	if _, err = RsaVerify(ctx, data, sign, []byte(testRsaPkcs1PrivateKey)); err == nil {
		t.Errorf("私钥当公钥使用应报错")
	}
	//空签名须验签失败并报错
	if ok, err = RsaVerify(ctx, data, nil, []byte(testRsaPublicKey1)); ok || err == nil {
		t.Errorf("空签名应验签失败并报错: ok=%v err=%v", ok, err)
	}
}

// genOtherRsaPublicKeyPem 现场生成一对与测试固定密钥无关的RSA密钥，返回其公钥PEM。
// 用于验证"密钥不匹配时必须验签失败"——这一点无法用 testRsaPublicKey1/2 验证，
// 因为那两个常量的内容完全相同，实际上只是同一份公钥的两个别名。
func genOtherRsaPublicKeyPem(t *testing.T) []byte {
	t.Helper()
	//2048位足够且比4096快很多，本用例只关心"不是同一把钥匙"
	key, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		t.Fatalf("生成测试用RSA密钥异常: %+v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("序列化测试用公钥异常: %+v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
}

// 固定测试密钥的自检：testRsaPublicKey1 与 testRsaPublicKey2 目前内容完全相同。
// 明确记录这一事实，避免后续用例误以为它们是两把不同的钥匙而写出恒真断言。
func TestRsaTestKeysAreSamePair(t *testing.T) {
	ctx := GenCtx()
	if testRsaPublicKey1 != testRsaPublicKey2 {
		//若将来有人补齐成两对真实密钥，这里会提醒同步更新依赖该假设的用例
		t.Skip("testRsaPublicKey1/2 已不同，跨密钥用例可改用常量而非现场生成")
	}
	//两个私钥常量编码不同(PKCS1 / PKCS8)，但对应同一把私钥，
	//故两者的签名结果必须完全一致（PKCS1v15为确定性签名）
	s1, err := RsaSignStr(ctx, "probe", testRsaPkcs1PrivateKey)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	s8, err := RsaSignStr(ctx, "probe", testRsaPkcs8PrivateKey)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if s1 != s8 {
		t.Errorf("PKCS1与PKCS8私钥常量应为同一把私钥的两种编码，签名却不同")
	}
}
