package test

import (
	"context"
	"fmt"
	"github.com/cellargalaxy/go_common/util"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"testing"
	"time"
)

func TestInitLog(t *testing.T) {
	util.InitLog("")
	ctx := util.CreateLogCtx()
	logrus.WithContext(ctx).WithFields(logrus.Fields{"TestInitLog": "TestInitLog"}).Info("TestInitLog")
}

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

func TestTime2MsTs(t *testing.T) {
	ts := util.Time2MsTs(time.Now())
	t.Logf("ts: %+v\n", ts)
}

func TestMsTs2Time(t *testing.T) {
	date := util.MsTs2Time(1605091056123)
	t.Logf("ts: %+v\n", util.Time2MsTs(date))
}

func TestEnSHa256(t *testing.T) {
	ctx := context.Background()
	data, err := util.EnSha256(ctx, []byte("aa"))
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("data: %+v\n", data)
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
	token, err := util.GenJWT(ctx, "aa", &claims)
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("token: %+v\n", token)
}

func TestParseJWT(t *testing.T) {
	ctx := context.Background()
	var claims MyClaims
	token, err := util.ParseJWT(ctx, "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpYXQiOjE2Mjc3MjQ2MTIsImV4cCI6MTYyNzcyNDY0Mn0.FEu96aZAtw7LzqqOKlGvfYE4V133vNjSIC1SC92oPxE", "aa", &claims)
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
func TestGenGoLabel(t *testing.T) {
	ctx := context.Background()
	code, err := util.ReadFileOrCreateIfNotExist(ctx, "test.txt", "")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	fmt.Println(util.GenGoLabel(ctx, code, "gorm"))
}
func TestGenModel2Sql(t *testing.T) {
	ctx := context.Background()
	code, err := util.ReadFileOrCreateIfNotExist(ctx, "test.txt", "")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	fmt.Println(util.GenModel2Sql(ctx, code))
}

func TestWareDuration(t *testing.T) {
	rand.Seed(time.Now().Unix())
	ns := 2
	var object time.Duration
	object = util.WareDuration(time.Duration(ns))
	t.Logf("object: %d\n", object)
	object = util.WareDuration(time.Duration(ns))
	t.Logf("object: %d\n", object)
	object = util.WareDuration(time.Duration(ns))
	t.Logf("object: %d\n", object)
	object = util.WareDuration(time.Duration(ns))
	t.Logf("object: %d\n", object)
	object = util.WareDuration(time.Duration(ns))
	t.Logf("object: %d\n", object)
	object = util.WareDuration(time.Duration(ns))
	t.Logf("object: %d\n", object)
	object = util.WareDuration(time.Duration(ns))
	t.Logf("object: %d\n", object)
	object = util.WareDuration(time.Duration(ns))
	t.Logf("object: %d\n", object)
	object = util.WareDuration(time.Duration(ns))
	t.Logf("object: %d\n", object)
}

func TestParseCurl(t *testing.T) {
	ctx := context.Background()
	object, err := util.ParseCurl(ctx, ``)
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("object: %+v\n", util.ToJsonIndentString(object))
}
