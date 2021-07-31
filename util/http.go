package util

import (
	"github.com/cellargalaxy/go_common/consd"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const TokenKey = "Authorization"
const ClaimsKey = "claims"

func CreateErrResponse(message string) map[string]interface{} {
	return gin.H{"code": consd.HttpFailCode, "msg": message, "data": nil}
}

func CreateResponse(data interface{}, err error) map[string]interface{} {
	if err == nil {
		return gin.H{"code": consd.HttpSuccessCode, "msg": nil, "data": data}
	} else {
		return gin.H{"code": consd.HttpFailCode, "msg": err.Error(), "data": data}
	}
}

func Ping(context *gin.Context) {
	context.JSON(http.StatusOK, gin.H{"code": consd.HttpSuccessCode, "msg": "pong", "data": map[string]interface{}{"timestamp": time.Now().Unix()}})
}

//token检查
func Validate(c *gin.Context, validateHandler func(c *gin.Context) (string, jwt.Claims, error)) {
	token := c.Request.Header.Get(TokenKey)
	logrus.WithContext(c).WithFields(logrus.Fields{"token": token}).Info("解析token")
	tokens := strings.SplitN(token, " ", 2)
	if len(tokens) != 2 || tokens[0] != "Bearer" {
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("Authorization非法"))
		return
	}
	secret, claims, err := validateHandler(c)
	if err != nil {
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse(err.Error()))
		return
	}
	jwtToken, err := ParseJWT(c, tokens[1], secret, claims)
	if err != nil {
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse(err.Error()))
		return
	}
	if jwtToken == nil {
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("JWT token为空"))
		return
	}
	if !jwtToken.Valid {
		c.Abort()
		c.JSON(http.StatusOK, CreateErrResponse("JWT token非法"))
		return
	}
	c.Set(ClaimsKey, jwtToken.Claims)
}

func StaticCache(c *gin.Context) {
	if strings.HasPrefix(c.Request.RequestURI, "/static") {
		c.Header("Cache-Control", "max-age=86400")
	}
}
