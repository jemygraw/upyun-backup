package upyun

import (
	"fmt"
)

// Upyun Signature
type UpyunSign struct {
	Method        string // http method
	Path          string // request path
	ContentLength int64  // content length under post
	Password      string // operator password
	Date          string // request date in format RFC1123
}

func (this *UpyunSign) ToString() string {
	if this.Method == "GET" || this.Method == "HEAD" {
		this.ContentLength = 0
	}
	buf := fmt.Sprintf("%s&%s&%s&%d&%s",
		this.Method, this.Path, this.Date,
		this.ContentLength, Md5(this.Password))
	return buf
}

func (this *UpyunSign) Token() string {
	return Md5(this.ToString())
}

// Upyun Authentication
type UpyunAuth struct {
	User  string // operator name
	Token string // signed token
}

func (this *UpyunAuth) ToString() string {
	return fmt.Sprintf("UpYun %s:%s", this.User, this.Token)
}
