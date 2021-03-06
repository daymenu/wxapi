package main

import (
	"context"
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
	SendMessage
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

var wechatChan = make(chan *wechat.Wechat, 500)

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

	// 放入待处理带缓存的channel
	wechatChan <- wx

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

// 【未扫码的话】 -> window.code=408;
// 【手机扫码但是未登录】 -> window.code = 201;
// 【手机取消登录】 -> window.code=400;
// 【手机授权登录】 -> window.code=200;
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

		logger.Printf("GetContactList:userId=%s response%s ip: %s", uuid, string(qrJSON), req.RemoteAddr)
		rw.Write(qrJSON)
		return
	}
	cr, err := ww.GetContactList()
	if err != nil {
		webResp.Code = LoginFaildCode
		webResp.Message = err.Error()
	}
	webResp.ContractResponse = cr
	qrJSON, err := json.Marshal(webResp)
	if err != nil {
		log.Print(err)
	}
	logger.Printf("GetContactList:userId=%s response%s ip: %s", uuid, string(qrJSON), req.RemoteAddr)
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

func (hw *httpWechat) SendImg(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json; charset=UTF-8")

	req.ParseForm()
	uuid := req.Form.Get("userId")
	userName := req.Form.Get("userName")
	path := req.Form.Get("path")

	logger.Printf("SendImg:userId=%s request:%s ip: %s", uuid, req.Form.Encode(), req.RemoteAddr)
	webResp := new(Response)
	ww, ok := hw.wechat[uuid]
	if !ok {
		webResp.Code = LoginFaildCode
		webResp.Message = "请先登录"
		qrJSON, err := json.Marshal(webResp)
		if err != nil {
			log.Print(err)
		}
		logger.Printf("SendImg:userId=%s response%s ip: %s", uuid, qrJSON, req.RemoteAddr)
		rw.Write(qrJSON)
		return
	}
	err := ww.SendMedia(userName, path)
	if err != nil {
		webResp.Message = err.Error()
		webResp.Code = SendMessage
	}
	qrJSON, err := json.Marshal(webResp)
	if err != nil {
		log.Print(err)
	}
	logger.Printf("SendImg:userId=%s response%s ip: %s", uuid, qrJSON, req.RemoteAddr)
	rw.Write(qrJSON)
}

func main() {
	hw := httpWechat{
		wechat: make(map[string]*wechat.Wechat),
	}

	// check login
	ctx := context.Background()
	go hw.initLogin(ctx)

	mux := http.NewServeMux()

	mux.HandleFunc("/qr", hw.Qr)
	mux.HandleFunc("/checkLogin", hw.Login)
	mux.HandleFunc("/getContactList", hw.GetContactList)
	mux.HandleFunc("/sendMessage", hw.SendMessage)
	mux.HandleFunc("/sendImg", hw.SendImg)

	addr := fmt.Sprintf(":%d", HTTPPort)

	fmt.Printf("version:0.04 wxapi start as %s\n", addr)
	logger.Printf("version:0.04 wxapi start as %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// 常驻线程 检查是否有人登录了/media/madison/工作/vue/vue-element-admin/src/assets/401_images/401.gif
func (hw *httpWechat) initLogin(ctx context.Context) {
	for wx := range wechatChan {
		if !wx.IsLogin() {
			loginCtx, cancel := context.WithDeadline(ctx, time.Now().Add(120*time.Second))
			fmt.Println(cancel)
			go wx.Login(loginCtx)
		}
	}
}

func (hw *httpWechat) syncCheck() {
	for key, wx := range hw.wechat {
		go func(key string, wx *wechat.Wechat) {
			syncResp, err := wx.SyncCheck()
			if err != nil {
				delete(hw.wechat, key)
			}
			fmt.Println(syncResp)
		}(key, wx)
	}
}
