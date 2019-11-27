package wechat

import (
	"bytes"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetRootPath root path
func GetRootPath() string {
	root, err := os.Getwd()
	if err != nil {
		return ""
	}
	return root
}

// JsResult wechat result
type JsResult map[string]string

// Get get JsResult value
func (wr *JsResult) Get(key string) (value string) {
	return (*wr)[key]
}

// ParseJsResult parse wechat result
func ParseJsResult(result []byte) *JsResult {
	wechatResult := JsResult{}
	lines := bytes.Split(result, []byte(";"))
	for _, line := range lines {
		lineStr := string(bytes.Trim(line, "\n\t\r"))
		i := strings.Index(lineStr, "=")
		if i == -1 {
			continue
		}
		key := strings.Trim(lineStr[:i], " ")
		value := strings.Trim(strings.Trim(lineStr[i+1:], " "), `"`)
		wechatResult[key] = value
	}
	return &wechatResult
}

func getDeviceID() string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	deviceID := rnd.Int63n(1000000000000000)
	return "e" + strconv.FormatInt(deviceID, 10)
}
