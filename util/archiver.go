package util

import (
	"bytes"
	"compress/gzip"
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
