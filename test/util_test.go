package test

import (
	"context"
	"fmt"
	"github.com/cellargalaxy/go_common/model"
	"github.com/cellargalaxy/go_common/util"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"math/rand"
	"testing"
	"time"
)

const (
	jwtSecret = "jwtSecret"
)

func TestLogId(t *testing.T) {
	ctx := util.GenCtx()
	time.Sleep(3 * time.Second)
	logrus.WithContext(ctx).WithFields(logrus.Fields{"GetLogId": util.GetLogId(ctx)}).Info("TestLogId")
}

func TestReqId(t *testing.T) {
	ctx := util.GenCtx()
	time.Sleep(3 * time.Second)
	logrus.WithContext(ctx).WithFields(logrus.Fields{"GetReqId": util.GetReqId(ctx)}).Info("TestReqId")
	ctx = util.SetReqId(ctx)
	logrus.WithContext(ctx).WithFields(logrus.Fields{"GetReqId": util.GetReqId(ctx)}).Info("TestReqId")
}

func TestContainNum(t *testing.T) {
	object := util.ContainNum("0.39亿元（截止至：2020年12月31日）")
	t.Logf("object: %+v\n", object)
}

func TestFindNum(t *testing.T) {
	object := util.FindNum("0.39亿元（截止至：2020年12月31日）")
	t.Logf("object: %+v\n", object)
}

func TestParseStr2Time(t *testing.T) {
	ctx := context.Background()
	object, err := util.ParseStr2Time(ctx, util.DateLayout_2006_01_02, "2021-03-21", util.UTCLoc)
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
	data := util.EnSha256(ctx, []byte("aa"))
	t.Logf("data: %+v\n", data)
}

func TestGenDefaultJWT(t *testing.T) {
	ctx := util.GenCtx()
	time.Sleep(time.Second * 3)
	token, err := util.EnDefaultJwt(ctx, time.Minute, jwtSecret)
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	logrus.WithContext(ctx).Info(token)
	t.Logf("token: %+v\n", token)
}

func TestGenJWT(t *testing.T) {
	ctx := context.Background()
	var claims model.Claims
	claims.Ip = "Ip"
	claims.ServerName = "ServerName"
	claims.LogId = 123456789
	claims.ReqId = "ReqId"
	claims.ExpiresAt = time.Now().Unix() + 1000
	token, err := util.EnJwt(ctx, jwtSecret, &claims)
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("token: %+v\n", token)
}

func TestParseJWT(t *testing.T) {
	ctx := context.Background()
	var claims model.Claims
	token, err := util.DeJwt(ctx, "", jwtSecret, &claims)
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
	code, err := util.ReadFileWithString(ctx, "test.txt", "")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	fmt.Println(util.GenGoLabel(ctx, code, "gorm"))
}
func TestGenModel2Sql(t *testing.T) {
	ctx := context.Background()
	code, err := util.ReadFileWithString(ctx, "test.txt", "")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	fmt.Println(util.GenModel2Sql(ctx, code))
}

func TestWareDuration(t *testing.T) {
	rand.Seed(time.Now().Unix())
	ns := time.Second
	var object time.Duration
	for i := 0; i < 100; i++ {
		object = util.WareDuration(time.Duration(ns))
		t.Logf("object: %d\n", object)
	}
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

type CsvStruct struct {
	Int     int       `json:"int" csv:"client_age"`
	Float64 float64   `json:"float_64" csv:"float_64"`
	String  string    `json:"string" csv:"string"`
	Bool    bool      `json:"bool" csv:"bool"`
	Time    time.Time `json:"time" csv:"time"`
}

func TestWriteCsvWithFile(t *testing.T) {
	ctx := context.Background()
	var list []CsvStruct
	list = append(list, CsvStruct{1, 1.1, "a", true, time.Now()})
	list = append(list, CsvStruct{2, 2.2, "b", false, time.Now()})
	err := util.WriteCsv2FileByStruct(ctx, list, "tmp/test.csv")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
}

func TestReadCsvWithFile(t *testing.T) {
	ctx := context.Background()
	var list []CsvStruct
	err := util.ReadCsvWithFile2Struct(ctx, "tmp/test.csv", &list)
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("object\n")
	for i := range list {
		t.Logf("object: %+v\n", util.ToJsonString(list[i]))
	}
}

func TestGenId(t *testing.T) {
	id := util.GenId()
	fmt.Println(fmt.Sprint(id))
}

func TestParseId(t *testing.T) {
	ctx := util.GenCtx()
	id := util.GenId()
	fmt.Println(util.ParseId(ctx, id))
}

func TestGetReadFile(t *testing.T) {
	ctx := context.Background()
	file, err := util.GetReadFile(ctx, "test.go")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("object: %+v\n", file)
}

func TestGetWriteFile(t *testing.T) {
	ctx := context.Background()
	file, err := util.GetWriteFile(ctx, "test.go")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("object: %+v\n", file)
}

func TestCreateFolderPath(t *testing.T) {
	ctx := util.GenCtx()
	err := util.CreateFolderPath(ctx, "./tmp")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
}

func TestGetIp(t *testing.T) {
	ctx := util.GenCtx()
	for i := 0; i < 1000; i++ {
		logrus.WithContext(ctx).WithFields(logrus.Fields{}).Info(util.GetIp())
		time.Sleep(time.Second)
	}
}

func claims(ctx *gin.Context) {
	util.ClaimsHttp(ctx, jwtSecret)
}
func TestHttpClaims(t *testing.T) {
	fmt.Println("http://127.0.0.1:8888/ping")
	engine := gin.Default()
	engine.Use(util.GinLog)
	engine.GET("/ping", claims, util.Ping)
	err := engine.Run(":8888")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
}

func validate(ctx *gin.Context) {
	util.ValidateHttp(ctx, jwtSecret)
}
func TestHttpValidate(t *testing.T) {
	fmt.Println("http://127.0.0.1:8888/ping")
	engine := gin.Default()
	engine.Use(util.GinLog)
	engine.GET("/ping", validate, util.Ping)
	err := engine.Run(":8888")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
}

func TestSleep(t *testing.T) {
	ctx := util.GenCtx()
	ctx, _ = context.WithTimeout(ctx, time.Second*1)
	start := time.Now()
	util.Sleep(ctx, time.Second*2)
	fmt.Println(time.Now().Sub(start))
}

func TestOnceSingleGoPool(t *testing.T) {
	ctx := util.GenCtx()
	//ctx, _ = context.WithTimeout(ctx, time.Second*7)
	pool, err := util.NewOnceSingleGoPool(ctx, "test", func(ctx context.Context, pool *util.SingleGoPool) {
		defer util.Defer(func(err interface{}, stack string) {
			if err != nil {
				fmt.Println("err", err)
			}
		})
		//fmt.Println("Sleep")
		//util.Sleep(ctx, time.Minute*500)
		//if util.CtxDone(ctx) {
		//	fmt.Println("CtxDone")
		//	return
		//}
		//util.Sleep(ctx, time.Millisecond*500)
		for {
			now := time.Now()
			fmt.Println("Now1", now)
			//fmt.Println("CtxDone", util.CtxDone(ctx))
			//fmt.Println(ctx.Deadline())
			//fmt.Println()
			if now.Unix()%15 == 0 {
				fmt.Println("cancel 1")
				//pool.Cancel(ctx)
				return
			}
			//if util.CtxDone(ctx) {
			//	fmt.Println("util.CtxDone(ctx), return 1")
			//	return
			//}
			time.Sleep(time.Millisecond * 500)
		}
		//fmt.Println("/", 1/(now.Unix()%2))
		//var object []string
		//fmt.Println("object[0]", object[0])
		//var object map[string]string
		//object[""] = ""
	})
	//fmt.Println("doing", pool.Doing(ctx))
	time.Sleep(time.Second * 3)
	//fmt.Println("doing", pool.Doing(ctx))
	//time.Sleep(time.Second * 60)
	pool.AddOnceTask(ctx, "???", func(ctx context.Context, pool *util.SingleGoPool) {
		defer util.Defer(func(err interface{}, stack string) {
			if err != nil {
				fmt.Println("err", err)
			}
		})
		//time.Sleep(time.Minute * 500)
		//util.Sleep(ctx, time.Millisecond*500)
		for {
			now := time.Now()
			fmt.Println("Now2", now)
			//fmt.Println("CtxDone", util.CtxDone(ctx))
			//fmt.Println(ctx.Deadline())
			//fmt.Println()
			if now.Unix()%15 == 0 {
				fmt.Println("cancel 2")
				//pool.Cancel(ctx)
				return
			}
			//if util.CtxDone(ctx) {
			//	fmt.Println("util.CtxDone(ctx), return 2")
			//	return
			//}
			time.Sleep(time.Millisecond * 500)
		}
		//fmt.Println("/", 1/(now.Unix()%2))
		//var object []string
		//fmt.Println("object[0]", object[0])
		//var object map[string]string
		//object[""] = ""
	})
	time.Sleep(time.Second * 3)
	pool.AddOnceTask(ctx, "???", func(ctx context.Context, pool *util.SingleGoPool) {
		defer util.Defer(func(err interface{}, stack string) {
			if err != nil {
				fmt.Println("err", err)
			}
		})
		for {
			now := time.Now()
			fmt.Println("Now3", now)
			time.Sleep(time.Millisecond * 500)
		}
	})

	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	time.Sleep(time.Second * 60)
}

func TestDaemonSingleGoPool(t *testing.T) {
	ctx := util.GenCtx()
	//ctx, _ = context.WithTimeout(ctx, time.Second*7)
	var err error
	pool, err := util.NewDaemonSingleGoPool(ctx, "test", time.Millisecond*500, func(ctx context.Context, pool *util.SingleGoPool) {
		defer util.Defer(func(err interface{}, stack string) {
			if err != nil {
				fmt.Println("err", err)
			}
		})
		//time.Sleep(time.Minute * 500)
		//util.Sleep(ctx, time.Millisecond*500)
		now := time.Now()
		fmt.Println("Now", now)
		if now.Unix()%15 == 0 {
			//fmt.Println("cancel")
			//pool.Cancel(ctx)
			return
		}
		//fmt.Println("/", 1/(now.Unix()%2))
		//var object []string
		//fmt.Println("object[0]", object[0])
		//var object map[string]string
		//object[""] = ""
	})
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	fmt.Println("doing", pool.Doing(ctx))
	time.Sleep(time.Second * 20)
	fmt.Println("doing", pool.Doing(ctx))
	time.Sleep(time.Second * 60)
}

func TestHttpClientSpider(t *testing.T) {
	ctx := util.GenCtx()
	ctx = util.SetCtxValue(ctx, util.LogIdKey, 1)
	//ctx, _ = context.WithTimeout(ctx, time.Second*7)
	response, err := util.GetHttpSpiderRequest(ctx).Get("https://wstbd.dynv6.net/server_center/static/html/qq.html?ccc=ddd#eee")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	object, err := util.DealHttpResponse(ctx, "TestHttpClientSpider", response, err)
	time.Sleep(time.Second * 1)

	response, err = util.GetHttpSpiderRequest(ctx).Get("https://wstbd.dynv6.net/server_center/static/html/zz.html?ccc=ddd#eee")
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	object, err = util.DealHttpResponse(ctx, "TestHttpClientSpider", response, err)
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("object: %+v\n", object)
}

func TestHttpApiRetry(t *testing.T) {
	ctx := util.GenCtx()
	type Data struct {
		Ts int64  `json:"ts"`
		Sn string `json:"sn"`
	}
	type Response struct {
		model.HttpResponse
		Data Data `json:"data"`
	}
	var object Response
	err := util.HttpApiWithTry(ctx, "TestHttpApiRetry", 0, util.SpiderSleepsDefault, &object, func() (*resty.Response, error) {
		return util.GetHttpSpiderRequest(ctx).Post("https://wstbd.dynv6.net/server_center/ping")
	})
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}
	t.Logf("object: %+v\n", object)
	t.Logf("object: %+v\n", util.ToJsonString(object))
}
