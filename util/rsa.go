package util

import (
	"context"
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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
