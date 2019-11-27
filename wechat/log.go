package wechat

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

func getLogger() *log.Logger {
	logDir := "/var/log/wxapi/"
	err := os.Mkdir(logDir, 0777)
	isExit := os.IsExist(err)
	if !isExit && err != nil {
		log.Print(err)
	}
	time.Now()
	filePath := filepath.Join(logDir, time.Now().Format("2006-01-02 15:04:05")+".log")
	f, err := os.Create(filePath)
	if !os.IsExist(err) && err != nil {
		log.Print(err)
	}
	return log.New(f, "wechat ", log.Ldate|log.Ltime)
}
