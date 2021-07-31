package test

import (
	"context"
	"github.com/cellargalaxy/go_common/util"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestContainNum(t *testing.T) {
	object := util.ContainNum("0.39亿元（截止至：2020年12月31日）")
	t.Logf("object: %+v\n", object)
}

func TestFindNum(t *testing.T) {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	object := util.FindNum("0.39亿元（截止至：2020年12月31日）")
	t.Logf("object: %+v\n", object)
}

func TestParseBeijingTime(t *testing.T) {
	object, err := util.Parse2BeijingTime(util.DateLayout_2006_01_02, "2021-03-21")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("object: %+v\n", object)
	t.Logf("object: %+v\n", object.Unix())
}

type MyClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func (this MyClaims) String() string {
	return util.ToJsonString(this)
}

//func (c MyClaims) Valid() error {
//	return nil
//}
func TestGenJWT(t *testing.T) {
	ctx := context.Background()
	var claims MyClaims
	claims.Username = "我是Username"
	claims.ExpiresAt = time.Now().Unix() + 100
	token, err := util.GenJWT(ctx, "Secret", &claims)
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("token: %+v\n", token)
}

func TestParseJWT(t *testing.T) {
	ctx := context.Background()
	var claims MyClaims
	token, err := util.ParseJWT(ctx, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6IuaIkeaYr1VzZXJuYW1lIiwiZXhwIjoxNjI3NzE1ODkyfQ.5S5EmU7eaGu1Se3omBqLyctUjn_nZQSC8LNch47HgqM", "Secret", &claims)
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("claims: %+v\n", claims)
	t.Logf("token.Valid: %+v\n", token.Valid)
	t.Logf("token.Claims: %+v\n", token.Claims)
}

func TestEnAesCbcString(t *testing.T) {
	ctx := context.Background()
	text, err := util.EnAesCbcString(ctx, "aaa", "bbb")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("text: %+v\n", text)
}
func TestDeAesCbcString(t *testing.T) {
	ctx := context.Background()
	text, err := util.DeAesCbcString(ctx, "SZw3gyyzBnvWHkayKDREaw==", "bbb")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("text: %+v\n", text)
}
