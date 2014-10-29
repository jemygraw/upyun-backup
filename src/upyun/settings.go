package upyun

import (
	"encoding/json"
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
	User     string `json:"user"`
	Password string `json:"password"`
	Bucket   string `json:"bucket"`
	LocalDir string `json:"localdir"`
	Domain   int32  `json:"domain"`
	Routine  int32  `json:"routine"`
	Debug    bool   `json:"debug"`
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
		err = openErr
		return
	}
	defer cfgH.Close()
	cfgData, readErr := ioutil.ReadAll(cfgH)
	if readErr != nil {
		err = readErr
		return
	}
	if unErr := json.Unmarshal(cfgData, &cfg); unErr != nil {
		err = unErr
		return
	}
	return
}
