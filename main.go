package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

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

func (hw *httpWechat) Qr(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json; charset=UTF-8")
	type qr struct {
		Response
		QrURL string `json:"qrUrl"`
	}
	qrResp := new(qr)

	wx := wechat.NewWechat()
	qrurl, err := wx.GetQr()
	uuid := wx.GetUUID()
	hw.Lock()
	defer hw.Unlock()
	hw.wechat[uuid] = wx
	if err != nil {
		qrResp.Code = 1
		qrResp.Message = err.Error()
	}

	qrResp.Code = 0
	qrResp.Message = "获取成功"
	qrResp.UUID = uuid
	qrResp.QrURL = qrurl

	qrJSON, err := json.Marshal(qrResp)
	if err != nil {
		log.Print(err)
	}
	rw.Write(qrJSON)
}

func (hw *httpWechat) Login(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json; charset=UTF-8")
	type WebResp struct {
		Response
		LoginStatus string `json:"loginStatus"`
	}
	req.ParseForm()
	uuid := req.Form.Get("uuid")
	log.Printf("%+v", hw.wechat)
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
	rw.Write(qrJSON)
}

func (hw *httpWechat) GetContactList(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json; charset=UTF-8")
	type WebResp struct {
		Response
		*wechat.ContractResponse
	}
	req.ParseForm()
	uuid := req.Form.Get("uuid")
	webResp := new(WebResp)
	ww, ok := hw.wechat[uuid]
	log.Printf("uuid=%s %+v", uuid, ww)
	if !ok {
		webResp.Code = LoginFaildCode
		webResp.Message = "请登录"
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
	rw.Write(qrJSON)
}

func (hw *httpWechat) SendMessage(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json; charset=UTF-8")

	req.ParseForm()
	uuid := req.Form.Get("uuid")
	userName := req.Form.Get("userName")
	message := req.Form.Get("message")
	log.Printf("%+v", hw.wechat)
	webResp := new(Response)
	ww, ok := hw.wechat[uuid]
	if !ok {
		webResp.Code = LoginFaildCode
		webResp.Message = "请先登录"
	}
	err := ww.SendMsg(userName, message, false)

	qrJSON, err := json.Marshal(webResp)
	if err != nil {
		log.Print(err)
	}
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
	log.Printf("api: http://localhost:%d/qr %s", HTTPPort, "获取二维码")
	mux.HandleFunc("/checkLogin", hw.Login)
	log.Printf("api: http://localhost:%d/checkLogin %s", HTTPPort, "检查是否登录")
	mux.HandleFunc("/getContactList", hw.GetContactList)
	log.Printf("api: http://localhost:%d/getContactList %s", HTTPPort, "获取联系人")
	mux.HandleFunc("/sendMessage", hw.SendMessage)
	log.Printf("api: http://localhost:%d/sendMessage %s", HTTPPort, "发送消息")

	addr := fmt.Sprintf(":%d", HTTPPort)
	log.Printf("server is start at:  %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))

}

// 常驻线程 检查是否有人登录了
func (hw *httpWechat) initLogin() {
	for {
		if hw.wechat == nil {
			continue
		}
		for _, w := range hw.wechat {
			if !w.IsLogin() {
				w.Login()
			}
		}
	}
}
