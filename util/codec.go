package util

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto"
	"crypto/md5"
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
	_, err := writer.Write(data)
	if err != nil {
		writer.Close()
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
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("GZIP解压，异常")
		return nil, errors.Errorf("GZIP解压，异常: %+v", err)
	}
	defer reader.Close()

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
func DeBase64(ctx context.Context, text string) ([]byte, error) {
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
func EnAuthorizationJwt(ctx context.Context, expire time.Duration, secret string) (string, string) {
	token, _ := EnDefaultJwt(ctx, expire, secret)
	return GenAuthorizationHeader(ctx, token)
}
func EnDefaultJwt(ctx context.Context, expire time.Duration, secret string) (string, error) {
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
	secretByte := EnSha256(ctx, []byte(secret))
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := jwtToken.SignedString(secretByte)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "hex": fmt.Sprintf("%x", secretByte)}).Error("JWT加密异常")
		return "", errors.Errorf("JWT加密异常: %+v", err)
	}
	return token, nil
}
func DeJwt(ctx context.Context, token, secret string, claims jwt.Claims) (*jwt.Token, error) {
	var err error
	secretByte := EnSha256(ctx, []byte(secret))

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
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err, "hex": fmt.Sprintf("%x", secretByte)}).Error("JWT解密异常")
		return nil, errors.Errorf("JWT解密异常: %+v", err)
	}
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
func DeAesCbc(ctx context.Context, data, secret []byte) ([]byte, error) {
	secret = EnSha256(ctx, secret)
	ivAes := EnMd5(ctx, secret)
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
	secret = EnSha256(ctx, secret)
	ivAes := EnMd5(ctx, secret)
	en, err := goEncrypt.AesCbcEncrypt(data, secret, ivAes)
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("AesCbc加密异常")
		return nil, errors.Errorf("AesCbc加密异常")
	}
	return en, nil
}

// 256
func EnSha256(ctx context.Context, data []byte) []byte {
	hash := sha256.New()
	hash.Write(data)
	return hash.Sum(nil)
}
func EnSha256Hex(ctx context.Context, data string) string {
	return fmt.Sprintf("%x", EnSha256(ctx, []byte(data)))
}

// 128
func EnMd5(ctx context.Context, data []byte) []byte {
	hash := md5.New()
	hash.Write(data)
	return hash.Sum(nil)
}
func EnMd5Hex(ctx context.Context, data string) string {
	return fmt.Sprintf("%x", EnMd5(ctx, []byte(data)))
}

func EnCrc32(ctx context.Context, data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}
func EnCrc32Hex(ctx context.Context, data string) string {
	return fmt.Sprintf("%x", EnCrc32(ctx, []byte(data)))
}

func RsaVerify(ctx context.Context, data, sign, publicKey string) (bool, error) {
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil || block.Type != "PUBLIC KEY" {
		logrus.WithFields(logrus.Fields{}).Error("rsa校验，非法公钥")
		return false, errors.Errorf("rsa校验，非法公钥")
	}

	public, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		logrus.WithFields(logrus.Fields{}).Error("rsa校验，公钥解析异常")
		return false, errors.Errorf("rsa校验，公钥解析异常")
	}

	pubRsaKey, ok := public.(*rsa.PublicKey)
	if !ok {
		logrus.WithFields(logrus.Fields{}).Error("rsa校验，公钥转型失败")
		return false, errors.Errorf("rsa校验，公钥转型失败")
	}

	dataHash := md5.Sum([]byte(data))
	signData, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("rsa校验，签名解析异常")
		return false, errors.Errorf("rsa校验，签名解析异常: %+v", err)
	}

	err = rsa.VerifyPKCS1v15(pubRsaKey, crypto.MD5, dataHash[:], signData)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("rsa校验，签名校验异常")
		return false, errors.Errorf("rsa校验，签名校验异常: %+v", err)
	}

	return true, nil
}
