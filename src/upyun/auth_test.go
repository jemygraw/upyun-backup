package upyun

import (
	"fmt"
	"testing"
	"time"
)

func TestUpyunSign(t *testing.T) {
	sign := UpyunSign{
		Method:        "GET",
		Uri:           "/demobucket/",
		ContentLength: 0,
		Password:      "jemy12345",
		Date:          UpyunTime(time.Now()),
	}
	token := sign.Token()
	fmt.Println(token)

	auth := UpyunAuth{
		User:  "jemy",
		Token: token,
	}
	fmt.Println(auth.ToString())
}
