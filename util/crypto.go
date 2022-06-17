package util

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/cellargalaxy/go_common/model"
	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
	"github.com/wumansgy/goEncrypt"
	"time"
)

func GenDefaultJWT(ctx context.Context, expire time.Duration, defaultServerName, secret string) (string, error) {
	now := time.Now()
	var claims model.Claims
	claims.IssuedAt = now.Unix()
	claims.ExpiresAt = now.Unix() + int64(expire.Seconds())
	claims.RequestId = fmt.Sprint(GenId())
	claims.Caller = GetServerName(defaultServerName)
	return GenJWT(ctx, secret, claims)
}
func GenJWT(ctx context.Context, secret string, claims jwt.Claims) (string, error) {
	secretByte, err := EnSha256(ctx, []byte(secret))
	if err != nil {
		return "", err
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := jwtToken.SignedString(secretByte)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("JWT加密异常")
		return "", fmt.Errorf("JWT加密异常: %+v", err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"token": token}).Info("JWT加密")
	return token, nil
}
func ParseJWT(ctx context.Context, token, secret string, claims jwt.Claims) (*jwt.Token, error) {
	secretByte, err := EnSha256(ctx, []byte(secret))
	if err != nil {
		return nil, err
	}

	var jwtToken *jwt.Token
	if claims != nil {
		jwtToken, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
			return secretByte, nil
		})
	} else {
		jwtToken, err = jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return secretByte, nil
		})
	}

	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("JWT解密异常")
		return nil, fmt.Errorf("JWT解密异常: %+v", err)
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{"claims": ToJsonString(claims)}).Info("JWT解密")
	return jwtToken, nil
}

func DeAesCbcString(ctx context.Context, text, secret string) (string, error) {
	data, err := DeBase64(ctx, text)
	if err != nil {
		return "", err
	}
	de, err := DeAesCbc(ctx, data, []byte(secret))
	if err != nil {
		return "", err
	}
	return string(de), nil
}
func EnAesCbcString(ctx context.Context, text, secret string) (string, error) {
	en, err := EnAesCbc(ctx, []byte(text), []byte(secret))
	if err != nil {
		return "", err
	}
	text, err = EnBase64(ctx, en)
	if err != nil {
		return "", err
	}
	return text, nil
}

func DeAesCbc(ctx context.Context, data, secret []byte) ([]byte, error) {
	secret, err := EnSha256(ctx, secret)
	if err != nil {
		return nil, err
	}
	ivAes, err := EnMd5(ctx, secret)
	if err != nil {
		return nil, err
	}
	de, err := goEncrypt.AesCbcDecrypt(data, secret, ivAes)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("AesCbc解密异常")
		return nil, fmt.Errorf("AesCbc解密异常")
	}
	return de, nil
}
func EnAesCbc(ctx context.Context, data, secret []byte) ([]byte, error) {
	secret, err := EnSha256(ctx, secret)
	if err != nil {
		return nil, err
	}
	ivAes, err := EnMd5(ctx, secret)
	if err != nil {
		return nil, err
	}
	en, err := goEncrypt.AesCbcEncrypt(data, secret, ivAes)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("AesCbc加密异常")
		return nil, fmt.Errorf("AesCbc加密异常")
	}
	return en, nil
}

func EnBase64(ctx context.Context, data []byte) (string, error) {
	return base64.StdEncoding.EncodeToString(data), nil
}
func DeBase64(ctx context.Context, text string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("Base64解码异常")
		return nil, fmt.Errorf("Base64解码异常")
	}
	return data, nil
}

//256
func EnSha256(ctx context.Context, data []byte) ([]byte, error) {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil), nil
}

//128
func EnMd5(ctx context.Context, data []byte) ([]byte, error) {
	hash := md5.New()
	hash.Write(data)
	return hash.Sum(nil), nil
}
