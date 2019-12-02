package wechat

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// GetLogger get loger
func GetLogger() *log.Logger {
	goos := runtime.GOOS
	var logDir string
	switch goos {
	case "windows":
		logDir = os.TempDir()
	default:
		logDir = "/var/log/wxapi/"
	}
	err := os.Mkdir(logDir, 0777)
	isExit := os.IsExist(err)
	if !isExit && err != nil {
		log.Print(err)
	}
	time.Now()
	filePath := filepath.Join(logDir, time.Now().Format("wxapi-2006-01-02")+".log")

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if os.IsNotExist(err) {
		f, err = os.Create(filePath)
	}
	if !os.IsExist(err) && err != nil {
		log.Print(err)
	}
	return log.New(f, "wechat ", log.Ldate|log.Ltime)
}
