package upyun

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/astaxie/beego/logs"
	"io/ioutil"
	"os"
)

const (
	DOMAIN_AUTO     = "http://v0.api.upyun.com"
	DOMAIN_DIANXIN  = "http://v1.api.upyun.com"
	DOMAIN_LIANTONG = "http://v2.api.upyun.com"
	DOMAIN_YIDONG   = "http://v3.api.upyun.com"
)

var L *logs.BeeLogger

type Conf struct {
	User     string `json:"user"`     // operator name
	Password string `json:"password"` // operator password
	Bucket   string `json:"bucket"`   // bucket name
	LocalDir string `json:"localdir"` // local directory, must exists
	Domain   int32  `json:"domain"`   // upyun domain
	Routine  int32  `json:"routine"`  // download routine count
	Debug    bool   `json:"debug"`    // debug mode
}

func InitLogs(jsonConfig string, debugMode bool) {
	L = logs.NewLogger(1)
	L.SetLevel(logs.LevelInformational)
	L.SetLogger("file", jsonConfig)
	if debugMode {
		L.SetLevel(logs.LevelDebug)
		L.SetLogger("console", jsonConfig)
	}
}

func LoadConfig(cfgFile string) (cfg Conf, err error) {
	cfgH, openErr := os.Open(cfgFile)
	if openErr != nil {
		err = errors.New(fmt.Sprintf("Open config file error: `%s'", openErr.Error()))
		return
	}
	defer cfgH.Close()
	cfgData, readErr := ioutil.ReadAll(cfgH)
	if readErr != nil {
		err = errors.New(fmt.Sprintf("Read config file error: `%s'", readErr.Error()))
		return
	}
	if unErr := json.Unmarshal(cfgData, &cfg); unErr != nil {
		err = errors.New(fmt.Sprintf("Parse config file error: `%s'", unErr.Error()))
		return
	}
	return
}
