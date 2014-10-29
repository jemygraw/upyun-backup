package upyun

import (
	"bufio"
	"fmt"
	"github.com/astaxie/beego/httplib"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type UpyunBackup struct {
	Domain string
}

func (this *UpyunBackup) SnapshotFiles(bucketName string, conf Conf, snapshotFile string) {
	if _, err := os.Stat(snapshotFile); err != nil {
		L.Informational("No snapshot file `%s' found, will create one", snapshotFile)
	} else {
		rErr := os.Rename(snapshotFile, snapshotFile+".old")
		if rErr == nil {
			L.Informational("Rename the existing `%s` to `%s`", snapshotFile, snapshotFile+".old")
		} else {
			L.Error("Unable to rename cache file, plz manually delete `%s' and `%s.old'",
				snapshotFile, snapshotFile)
			return
		}
	}

	spFile, openErr := os.OpenFile(snapshotFile, os.O_CREATE|os.O_WRONLY, 0666)
	if openErr != nil {
		L.Error("Open snapshot file failed `%s'", openErr.Error())
		return
	}
	defer spFile.Close()
	L.Informational("Create a new snapshot file `%s'", snapshotFile)
	brWriter := bufio.NewWriter(spFile)
	defer brWriter.Flush()

	if this.Domain == "" {
		this.Domain = DOMAIN_AUTO
	}

	reqPath := fmt.Sprintf("/%s/", UrlEncode(bucketName))
	this.getPathList(reqPath, conf, brWriter)
	L.Informational("Finish the snapshot of files")
}

func (this *UpyunBackup) getPathList(reqPath string, conf Conf, writer *bufio.Writer) {
	reqDate := UpyunTime(time.Now())
	reqSign := UpyunSign{
		Method:   "GET",
		Path:     reqPath,
		Password: conf.Password,
		Date:     reqDate,
	}
	reqToken := reqSign.Token()
	reqAuth := UpyunAuth{
		User:  conf.User,
		Token: reqToken,
	}
	reqUri := fmt.Sprintf("%s%s", this.Domain, reqPath)
	req := httplib.Get(reqUri)
	req.SetUserAgent("Go 1.1 package http")
	req.Header("Authorization", reqAuth.ToString())
	req.Header("Date", reqDate)
	resp, respErr := req.Response()
	if respErr != nil {
		L.Error("Get path `%s' files response failed due to `%s'", reqPath, respErr)
		return
	}
	respData, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		L.Error("Get path `%s' files response data error", reqPath)
		return
	}
	bReader := bufio.NewScanner(strings.NewReader(string(respData)))
	bReader.Split(bufio.ScanLines)
	for bReader.Scan() {
		line := bReader.Text()
		items := strings.Split(line, "\t")
		if len(items) != 4 {
			L.Error("Invalid response format `%s'", line)
			continue
		}
		fname := items[0]
		ftype := items[1]
		switch ftype {
		case "F":
			//go and read
			reqPath = fmt.Sprintf("%s%s/", reqPath, UrlEncode(fname))
			this.getPathList(reqPath, conf, writer)
		case "N":
			items[0] = strings.Join([]string{reqPath, fname}, "")
			writer.WriteString(strings.Join(items, "\t"))
			writer.WriteString("\n")
		}
	}
}

func (this *UpyunBackup) BackupFiles(bucketName string, conf Conf, snapshotFile string) {

}
