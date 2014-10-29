package upyun

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"time"
)

func Md5(from string) string {
	hash := md5.New()
	hash.Write([]byte(from))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func UrlEncode(from string) string {
	return url.QueryEscape(from)
}

func UpyunTime(t time.Time) string {
	return t.UTC().Format(time.RFC1123)
}
