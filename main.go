package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/daymenu/wxapi/wechat"
)

// HTTPPort HTTPPort
const (
	HTTPPort = 9190

	ResponseSuccess = 0
	LoginFaildCode  = 108 + iota
	FetchFaildCode
)

type httpWechat struct {
	wechat map[string]*wechat.Wechat
	sync.RWMutex
}

// Response Response
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	UUID    string `json:"uuid"`
}

var logger = wechat.GetLogger()

func (hw *httpWechat) Qr(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json; charset=UTF-8")
	type qr struct {
		Response
		QrURL string `json:"qrUrl"`
	}
	qrResp := new(qr)
	req.ParseForm()
	uuid := req.Form.Get("userId")

	logger.Printf("qr:userId=%s request:%s  ip: %s", uuid, req.Form.Encode(), req.RemoteAddr)
	wx := wechat.NewWechat(logger)

	qrurl, err := wx.GetQr()
	hw.Lock()
	defer hw.Unlock()
	hw.wechat[uuid] = wx
	if err != nil {
		qrResp.Code = 1
		qrResp.Message = err.Error()
	}

	qrResp.Code = 0
	qrResp.Message = "获取成功"
	qrResp.UUID = wx.GetUUID()
	qrResp.QrURL = qrurl

	qrJSON, err := json.Marshal(qrResp)
	if err != nil {
		log.Print(err)
	}
	logger.Printf("qr:userId=%s response%s ip: %s", uuid, qrJSON, req.RemoteAddr)
	rw.Write(qrJSON)
}

func (hw *httpWechat) Login(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json; charset=UTF-8")
	type WebResp struct {
		Response
		LoginStatus string `json:"loginStatus"`
	}
	req.ParseForm()
	uuid := req.Form.Get("userId")
	logger.Printf("login :userId=%s request:%s  ip: %s", uuid, req.Form.Encode(), req.RemoteAddr)
	webResp := new(WebResp)
	wechat, ok := hw.wechat[uuid]
	if ok && wechat.IsLogin() {
		webResp.LoginStatus = "0"
	} else {
		webResp.LoginStatus = "1"
	}
	qrJSON, err := json.Marshal(webResp)
	if err != nil {
		log.Print(err)
	}
	logger.Printf("login:userId=%s response%s ip: %s", uuid, qrJSON, req.RemoteAddr)
	rw.Write(qrJSON)
}

func (hw *httpWechat) GetContactList(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json; charset=UTF-8")
	type WebResp struct {
		Response
		*wechat.ContractResponse
	}
	req.ParseForm()
	uuid := req.Form.Get("userId")
	logger.Printf("GetContactList:userId=%s request:%s ip: %s", uuid, req.Form.Encode(), req.RemoteAddr)
	webResp := new(WebResp)
	ww, ok := hw.wechat[uuid]
	log.Printf("uuid=%s %+v", uuid, ww)
	if !ok {
		webResp.Code = LoginFaildCode
		webResp.Message = "请登录"
		qrJSON, err := json.Marshal(webResp)
		if err != nil {
			log.Print(err)
		}

		logger.Printf("GetContactList:userId=%s response%s ip: %s", uuid, qrJSON, req.RemoteAddr)
		rw.Write(qrJSON)
		return
	}
	cr, err := ww.GetContactList()
	if err != nil {
		webResp.Code = FetchFaildCode
		webResp.Message = "请求失败，请重试"
	}
	webResp.ContractResponse = cr
	qrJSON, err := json.Marshal(webResp)
	if err != nil {
		log.Print(err)
	}
	logger.Printf("GetContactList:userId=%s response%s ip: %s", uuid, qrJSON, req.RemoteAddr)
	rw.Write(qrJSON)
}

func (hw *httpWechat) SendMessage(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json; charset=UTF-8")

	req.ParseForm()
	uuid := req.Form.Get("userId")
	userName := req.Form.Get("userName")
	message := req.Form.Get("message")

	logger.Printf("SendMessage:userId=%s request:%s ip: %s", uuid, req.Form.Encode(), req.RemoteAddr)
	webResp := new(Response)
	ww, ok := hw.wechat[uuid]
	if !ok {
		webResp.Code = LoginFaildCode
		webResp.Message = "请先登录"
		qrJSON, err := json.Marshal(webResp)
		if err != nil {
			log.Print(err)
		}
		logger.Printf("GetContactList:userId=%s response%s ip: %s", uuid, qrJSON, req.RemoteAddr)
		rw.Write(qrJSON)
		return
	}
	err := ww.SendMsg(userName, message, false)

	qrJSON, err := json.Marshal(webResp)
	if err != nil {
		log.Print(err)
	}
	logger.Printf("SendMessage:userId=%s response%s ip: %s", uuid, qrJSON, req.RemoteAddr)
	rw.Write(qrJSON)
}

func main() {
	hw := httpWechat{
		wechat: make(map[string]*wechat.Wechat),
	}

	// check login
	go hw.initLogin()

	mux := http.NewServeMux()

	mux.HandleFunc("/qr", hw.Qr)
	mux.HandleFunc("/checkLogin", hw.Login)
	mux.HandleFunc("/getContactList", hw.GetContactList)
	mux.HandleFunc("/sendMessage", hw.SendMessage)

	addr := fmt.Sprintf(":%d", HTTPPort)

	fmt.Printf("version:0.02 wxapi start as %s\n", addr)
	logger.Printf("version:0.02 wxapi start as %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// 常驻线程 检查是否有人登录了
func (hw *httpWechat) initLogin() {
	for {
		if hw.wechat == nil {
			continue
		}
		hw.RLock()
		fmt.Println(hw.wechat)
		time.Sleep(1 * time.Second)
		for userID, w := range hw.wechat {
			logger.Printf("check %s login ....", userID)
			if !w.IsLogin() {
				logger.Printf("userId : %s no login", userID)
				go w.Login()
			}
		}
		hw.RUnlock()
	}
}
