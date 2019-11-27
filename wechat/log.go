package wechat

import (
	"log"
	"os"
)

func getLogger() *log.Logger {
	return log.New(os.Stdout, "wechat ", log.Ldate|log.Ltime)
}
