package util

import (
	"context"
	"github.com/cellargalaxy/go_common/consd"
	"github.com/cellargalaxy/go_common/model"
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

func ParseCurl(ctx context.Context, curl string) (*model.HttpRequestParam, error) {
	var param model.HttpRequestParam
	param.Header = make(map[string]string)
	lines := strings.Split(curl, "\n")
	for i := range lines {
		line := lines[i]
		line = strings.Trim(line, " ")
		line = strings.Trim(line, "\t")
		if strings.HasSuffix(line, "\\") {
			line = line[:len(line)-1]
		}
		if strings.HasPrefix(line, "curl") {
			line = line[4:]
			line = strings.Trim(line, " ")
			line = strings.Trim(line, "\t")
			if strings.HasPrefix(line, "'") || strings.HasPrefix(line, "\"") {
				line = line[1:]
			}
			if strings.HasSuffix(line, "'") || strings.HasSuffix(line, "\"") {
				line = line[:len(line)-1]
			}
			param.Url = line
			continue
		}
		if strings.HasPrefix(line, "-H") {
			line = line[2:]
			line = strings.Trim(line, " ")
			line = strings.Trim(line, "\t")
			if strings.HasPrefix(line, "'") || strings.HasPrefix(line, "\"") {
				line = line[1:]
			}
			if strings.HasSuffix(line, "'") || strings.HasSuffix(line, "\"") {
				line = line[:len(line)-1]
			}
			ss := strings.Split(line, ":")
			if len(ss) < 2 {
				continue
			}
			key := ss[0]
			key = strings.Trim(key, " ")
			key = strings.Trim(key, "\t")
			value := ss[1]
			value = strings.Trim(value, " ")
			value = strings.Trim(value, "\t")
			param.Header[key] = value
			continue
		}
		if strings.HasPrefix(line, "--data-raw") {
			line = line[10:]
			line = strings.Trim(line, " ")
			line = strings.Trim(line, "\t")
			if strings.HasPrefix(line, "'") || strings.HasPrefix(line, "\"") {
				line = line[1:]
			}
			if strings.HasSuffix(line, "'") || strings.HasSuffix(line, "\"") {
				line = line[:len(line)-1]
			}
			param.Body = line
			continue
		}
	}
	return &param, nil
}
