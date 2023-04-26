package util

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/cellargalaxy/go_common/model"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wumansgy/goEncrypt"
	"hash/crc32"
	"time"
)

func EnGzip(ctx context.Context, data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	defer CloseIo(ctx, writer)
	_, err := writer.Write(data)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("GZIP压缩，异常")
		return nil, errors.Errorf("GZIP压缩，异常: %+v", err)
	}
	err = writer.Close()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("GZIP压缩，异常")
		return nil, errors.Errorf("GZIP压缩，异常: %+v", err)
	}
	return buf.Bytes(), nil
}
func DeGzip(ctx context.Context, data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	defer CloseIo(ctx, reader)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("GZIP解压，异常")
		return nil, errors.Errorf("GZIP解压，异常: %+v", err)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("GZIP解压，异常")
		return nil, errors.Errorf("GZIP解压，异常: %+v", err)
	}
	return buf.Bytes(), nil
}

func EnBase64(ctx context.Context, data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
func DeBase64(ctx context.Context, text string) []byte {
	data, _ := deBase64(ctx, text)
	return data
}
func deBase64(ctx context.Context, text string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("Base64解码异常")
		return nil, errors.Errorf("Base64解码异常")
	}
	return data, nil
}

func GenAuthorizationHeader(ctx context.Context, token string) (string, string) {
	return AuthorizationKey, fmt.Sprintf("%s %s", BearerKey, token)
}
func EnAuthorizationJwt(ctx context.Context, secret string, expire time.Duration) (string, string) {
	token, _ := EnDefaultJwt(ctx, secret, expire)
	return GenAuthorizationHeader(ctx, token)
}
func EnDefaultJwt(ctx context.Context, secret string, expire time.Duration) (string, error) {
	now := time.Now()
	var claims model.Claims
	claims.IssuedAt = now.Add(-expire).Unix()
	claims.ExpiresAt = now.Add(expire).Unix()
	claims.Ip = GetIp()
	claims.ServerName = GetServerName()
	claims.LogId = GetLogId(ctx)
	claims.ReqId = GetOrGenReqIdString(ctx)
	return EnJwt(ctx, secret, claims)
}
func EnJwt(ctx context.Context, secret string, claims jwt.Claims) (string, error) {
	secretHash := EnSha256([]byte(secret))
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := jwtToken.SignedString(secretHash)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("JWT加密异常")
		return "", errors.Errorf("JWT加密异常: %+v", err)
	}
	return token, nil
}
func DeJwt(ctx context.Context, token, secret string, claims jwt.Claims) (*jwt.Token, error) {
	var err error
	secretHash := EnSha256([]byte(secret))

	var jwtToken *jwt.Token
	if claims != nil {
		jwtToken, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
			return secretHash, nil
		})
	} else {
		jwtToken, err = jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return secretHash, nil
		})
	}

	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("JWT解密异常")
		return nil, errors.Errorf("JWT解密异常: %+v", err)
	}
	return jwtToken, nil
}

func DeAesCbcString(ctx context.Context, text, secret string) (string, error) {
	data, err := deBase64(ctx, text)
	if err != nil {
		return "", err
	}
	de, err := DeAesCbc(ctx, data, []byte(secret))
	if err != nil {
		return "", err
	}
	return string(de), nil
}
func DeAesCbc(ctx context.Context, data, secret []byte) ([]byte, error) {
	secret = EnSha256(secret)
	ivAes := EnMd5(secret)
	de, err := goEncrypt.AesCbcDecrypt(data, secret, ivAes)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("AesCbc解密异常")
		return nil, errors.Errorf("AesCbc解密异常")
	}
	return de, nil
}
func EnAesCbcString(ctx context.Context, text, secret string) (string, error) {
	en, err := EnAesCbc(ctx, []byte(text), []byte(secret))
	if err != nil {
		return "", err
	}
	text = EnBase64(ctx, en)
	return text, nil
}
func EnAesCbc(ctx context.Context, data, secret []byte) ([]byte, error) {
	secret = EnSha256(secret)
	ivAes := EnMd5(secret)
	en, err := goEncrypt.AesCbcEncrypt(data, secret, ivAes)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("AesCbc加密异常")
		return nil, errors.Errorf("AesCbc加密异常")
	}
	return en, nil
}

// 256
func EnSha256(data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}
func EnSha256Hex(data string) string {
	return fmt.Sprintf("%x", EnSha256([]byte(data)))
}

// 128
func EnMd5(data []byte) []byte {
	hash := md5.New()
	hash.Write(data)
	return hash.Sum(nil)
}
func EnMd5Hex(data string) string {
	return fmt.Sprintf("%x", EnMd5([]byte(data)))
}

func EnCrc32(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}
func EnCrc32Hex(data string) string {
	return fmt.Sprintf("%x", EnCrc32([]byte(data)))
}

func RsaSignString(ctx context.Context, data, privateKey string) (string, error) {
	sign, err := RsaSign(ctx, []byte(data), []byte(privateKey))
	if err != nil {
		return "", err
	}
	return EnBase64(ctx, sign), nil
}
func RsaSign(ctx context.Context, data, privateKey []byte) ([]byte, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		logrus.WithFields(logrus.Fields{}).Error("RSA签名，非法私钥")
		return nil, errors.Errorf("RSA签名，非法私钥")
	}

	var private *rsa.PrivateKey
	var err error
	switch block.Type {
	case "RSA PRIVATE KEY":
		private, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			logrus.WithFields(logrus.Fields{"err": err}).Error("RSA签名，私钥解析异常")
			return nil, errors.Errorf("RSA签名，私钥解析异常")
		}
	case "PRIVATE KEY":
		pri, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			logrus.WithFields(logrus.Fields{"err": err}).Error("RSA签名，私钥解析异常")
			return nil, errors.Errorf("RSA签名，私钥解析异常")
		}
		var ok bool
		private, ok = pri.(*rsa.PrivateKey)
		if !ok {
			logrus.WithFields(logrus.Fields{}).Error("RSA签名，私钥转型失败")
			return nil, errors.Errorf("RSA签名，私钥转型失败")
		}
	default:
		logrus.WithFields(logrus.Fields{}).Error("RSA签名，非法私钥")
		return nil, errors.Errorf("RSA签名，非法私钥")
	}

	dataHash := EnSha256(data)
	sign, err := rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, dataHash)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("RSA签名，签名生成异常")
		return nil, errors.Errorf("RSA签名，签名生成异常: %+v", err)
	}

	return sign, nil
}
func RsaVerifyString(ctx context.Context, data, sign, publicKey string) (bool, error) {
	signData, err := deBase64(ctx, sign)
	if err != nil {
		return false, err
	}
	return RsaVerify(ctx, []byte(data), signData, []byte(publicKey))
}
func RsaVerify(ctx context.Context, data, sign, publicKey []byte) (bool, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil || block.Type != "PUBLIC KEY" {
		logrus.WithFields(logrus.Fields{}).Error("RSA校验，非法公钥")
		return false, errors.Errorf("RSA校验，非法公钥")
	}

	public, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("RSA校验，公钥解析异常")
		return false, errors.Errorf("RSA校验，公钥解析异常")
	}

	pubRsaKey, ok := public.(*rsa.PublicKey)
	if !ok {
		logrus.WithFields(logrus.Fields{}).Error("RSA校验，公钥转型失败")
		return false, errors.Errorf("RSA校验，公钥转型失败")
	}

	dataHash := EnSha256(data)
	err = rsa.VerifyPKCS1v15(pubRsaKey, crypto.SHA256, dataHash, sign)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("RSA校验，签名校验异常")
		return false, errors.Errorf("RSA校验，签名校验异常: %+v", err)
	}

	return true, nil
}
