package upyun

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type UpyunBackup struct {
	Domain string
}

type FileStat struct {
	Type         string
	Size         int64
	LastModified int64
}

func (this *UpyunBackup) getFileStat(fRemotePath string, conf Conf) (fs FileStat, err error) {
	reqDate := UpyunTime(time.Now())
	reqSign := UpyunSign{
		Method:   "HEAD",
		Path:     fRemotePath,
		Password: conf.Password,
		Date:     reqDate,
	}
	reqToken := reqSign.Token()
	reqAuth := UpyunAuth{
		User:  conf.User,
		Token: reqToken,
	}

	if this.Domain == "" {
		this.Domain = DOMAIN_AUTO
	}

	reqUri := fmt.Sprintf("%s%s", this.Domain, fRemotePath)
	req := httplib.Head(reqUri)
	req.SetUserAgent("Go 1.1 package http")
	req.Header("Authorization", reqAuth.ToString())
	req.Header("Date", reqDate)
	resp, respErr := req.Response()
	if respErr != nil {
		err = errors.New(fmt.Sprintf("Get file `%s' stat response failed due to `%s'", fRemotePath, respErr.Error()))
		return
	}
	if resp.StatusCode != 200 {
		err = errors.New(fmt.Sprintf("Get file `%s' stat failed due to `%s'", fRemotePath, resp.Status))
		return
	}

	fType := resp.Header.Get("X-Upyun-File-Type")
	fSize, _ := strconv.ParseInt(resp.Header.Get("X-Upyun-File-Size"), 10, 64)
	fLastModified, _ := strconv.ParseInt(resp.Header.Get("X-Upyun-File-Date"), 10, 64)
	fs = FileStat{
		Type:         fType,
		Size:         fSize,
		LastModified: fLastModified,
	}
	return
}

func (this *UpyunBackup) SnapshotFiles(conf Conf, snapshotFile string) {
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

	reqDir := "/"
	reqPath := fmt.Sprintf("/%s%s", UrlEncode(conf.Bucket), reqDir)
	this.getPathList(reqPath, reqDir, conf, brWriter)
	L.Informational("Finish the snapshot of files")
}

func (this *UpyunBackup) getPathList(reqPath string, reqDir string, conf Conf, writer *bufio.Writer) {
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
		L.Error("Get path `%s' files response failed due to `%s'", reqPath, respErr.Error())
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
			//find more files under this dir
			reqDirX := fmt.Sprintf("%s%s/", reqDir, UrlEncode(fname))
			reqPathX := fmt.Sprintf("%s/%s/", reqPath, UrlEncode(fname))
			this.getPathList(reqPathX, reqDirX, conf, writer)
		case "N":
			items[0] = strings.Join([]string{reqDir, fname}, "")
			writer.WriteString(strings.Join(items, "\t"))
			writer.WriteString("\n")
		}
	}
}

func (this *UpyunBackup) BackupFiles(conf Conf, snapshotFile string) {
	spFile, readErr := os.Open(snapshotFile)
	if readErr != nil {
		L.Error("Read snapshot file error due to `%s'", readErr.Error())
		return
	}
	defer spFile.Close()

	var maxRoutines int32 = conf.Routine
	var currentRoutines int32 = 0
	bReader := bufio.NewScanner(spFile)
	bReader.Split(bufio.ScanLines)
	for bReader.Scan() {
		line := bReader.Text()
		items := strings.Split(line, "\t")
		itemCount := len(items)
		if itemCount != 4 && itemCount != 1 {
			L.Error("Error parsing file content `%s'", line)
			continue
		}
		fpath := items[0]
		var fsize int64 = -1
		var pErr error
		if itemCount == 4 {
			fsize, pErr = strconv.ParseInt(items[2], 10, 64)
			if pErr != nil {
				L.Error("Cannot parse file size for `%s'", fpath)
				continue
			}
		}
		if !strings.HasPrefix(fpath, "/") {
			L.Error("Must specify the path, which starts with a `/', for file `%s'", fpath)
			continue
		} else if strings.HasSuffix(fpath, "/") {
			L.Error("File path cannot ends with `/' for file `%s'", fpath)
			continue
		}
		for {
			curR := atomic.LoadInt32(&currentRoutines)
			if curR < maxRoutines {
				atomic.AddInt32(&currentRoutines, 1)
				go this.downloadFromAPI(fpath, fsize, conf, &currentRoutines)
				break
			} else {
				<-time.After(time.Microsecond * 1)
			}
		}
	}

	//check all download routine done
	for {
		curR := atomic.LoadInt32(&currentRoutines)
		L.Debug("Remained download routines: `%d'", curR)
		if curR == 0 {
			break
		} else {
			<-time.After(time.Second * 2)
		}
	}
}

func (this *UpyunBackup) downloadFromAPI(fpath string, fsize int64, conf Conf, currentRoutines *int32) {
	defer func() {
		atomic.AddInt32(currentRoutines, -1)
		runtime.Gosched()
	}()
	fLocalPath := filepath.Join(conf.LocalDir, ".", fpath)
	fRemotePath := fmt.Sprintf("/%s%s", conf.Bucket, fpath)
	//escape the remote path
	fRemotePathParts := strings.Split(fRemotePath, "/")
	for i := 0; i < len(fRemotePathParts); i++ {
		fRemotePathParts[i] = UrlEncode(fRemotePathParts[i])
	}
	fRemotePath = strings.Join(fRemotePathParts, "/")

	//check whether the local file exists and not changed, otherwise go ahead to download
	fLocalStat, fLocalStatErr := os.Stat(fLocalPath)
	if fLocalStatErr == nil {
		// file exists
		if fsize == -1 {
			L.Debug("Check remote file stat for `%s'", fRemotePath)
			//get file size from upyun server
			fRemoteStat, fRemoteStatErr := this.getFileStat(fRemotePath, conf)
			if fRemoteStatErr != nil {
				L.Error("Remote stat `%s' error due to `%s'", fRemotePath, fRemoteStatErr.Error())
				return
			}
			fsize = fRemoteStat.Size
		}
		if fLocalStat.Size() == fsize {
			L.Debug("Local file `%s' exists and no changes, pass...", fLocalPath)
			return
		}
	}

	//mkdir if necessary
	lastSlashIndex := strings.LastIndex(fLocalPath, string(filepath.Separator))
	if lastSlashIndex == -1 {
		L.Error("Get local path failed for file `%s'", fLocalPath)
		return
	}
	locaPath := fLocalPath[:lastSlashIndex]
	if mkdErr := os.MkdirAll(locaPath, 0775); mkdErr != nil {
		L.Error("Failed to create local dir `%s' due to `%s'", locaPath, mkdErr.Error())
		return
	}

	//create auth
	reqDate := UpyunTime(time.Now())
	reqSign := UpyunSign{
		Method:   "GET",
		Path:     fRemotePath,
		Password: conf.Password,
		Date:     reqDate,
	}
	reqToken := reqSign.Token()
	reqAuth := UpyunAuth{
		User:  conf.User,
		Token: reqToken,
	}

	if this.Domain == "" {
		this.Domain = DOMAIN_AUTO
	}

	reqUri := fmt.Sprintf("%s%s", this.Domain, fRemotePath)
	req := httplib.Get(reqUri)
	req.SetUserAgent("Go 1.1 package http")
	req.Header("Authorization", reqAuth.ToString())
	req.Header("Date", reqDate)
	resp, respErr := req.Response()
	if respErr != nil {
		L.Error("Get path `%s' files response failed due to `%s'", fRemotePath, respErr.Error())
		return
	}
	//check status code
	if resp.StatusCode != 200 {
		respData, _ := ioutil.ReadAll(resp.Body)
		L.Error("Response for `%s' not ok `%s'", fRemotePath, string(respData))
		return
	}
	L.Debug("Downloading `%s' -> `%s'", fRemotePath, fLocalPath)
	//write data to local file
	localFileH, openErr := os.OpenFile(fLocalPath, os.O_CREATE|os.O_WRONLY, 0666)
	if openErr != nil {
		L.Error("Open local file `%s' failed due to `%s'", fLocalPath, openErr.Error())
		return
	}
	defer localFileH.Close()

	_, wErr := io.Copy(localFileH, resp.Body)
	if wErr != nil {
		L.Error("Write local file `%s' failed due to `%s'", wErr.Error())
	}
}
