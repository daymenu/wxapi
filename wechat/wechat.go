package wechat

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// NewWechat new wechat
func NewWechat(logger *log.Logger) *Wechat {
	return &Wechat{
		Client:   getHTTPClient(),
		deviceID: getDeviceID(),
		Request: &Request{
			BaseRequest: new(BaseRequest),
		},
		Response:  Response{},
		MemberMap: map[string]Member{},
		Log:       logger,
	}
}

//GetUUID  获取UUID
func (w *Wechat) GetUUID() string {
	return w.uuID
}

// GetQr 获取二维码路径
func (w *Wechat) GetQr() (path string, err error) {
	err = w.fetchuuID()
	if err != nil {
		w.Log.Printf("%s", err)
		return
	}
	err = w.fetchQr()
	if err != nil {
		w.Log.Printf("%s", err)
		return
	}
	return w.qrImagePath, nil
}

func (w *Wechat) set() {
	w.uuID = "oYLMjVpIwQ=="
	w.qrImagePath = "/home/hanjian/work/go/gopl/wechat/qrimages/oYLMjVpIwQ==.jpg"
	w.redirectedURL = "https://wx.qq.com/cgi-bin/mmwebwx-bin/webwxnewloginpage?ticket=AfW4DZldIe1qYGg-OZB-JG2t@qrticket_0&uuid=oYLMjVpIwQ==&lang=zh-CN&scan=1574557489"
	w.setBaseURL()
}

// Login login
func (w *Wechat) Login() (err error) {
	err = w.login()
	if err != nil {
		return
	}
	err = w.webwxinit()
	if err != nil {
		return
	}
	return
}

// login fetch common params
func (w *Wechat) login() (err error) {
	w.waitForLogin()
	if w.redirectedURL == "" {
		return
	}
	response, err := w.Client.Get(w.redirectedURL + "&fun=new")
	if err != nil {
		return
	}
	defer response.Body.Close()
	reader := response.Body.(io.Reader)
	if err = xml.NewDecoder(reader).Decode(w.Request.BaseRequest); err != nil {
		return
	}
	w.Request.BaseRequest.DeviceID = w.deviceID
	w.Log.Printf("login:%+v", w.Request.BaseRequest)
	return nil
}

//IsLogin  is login
func (w *Wechat) IsLogin() bool {
	if w.Request.BaseRequest.PassTicket == "" {
		return false
	}
	return true
}

// webwxinit
func (w *Wechat) webwxinit() (err error) {
	if w.Request.BaseRequest.PassTicket == "" {
		return
	}
	wxinitURL := fmt.Sprintf("%s?pass_ticket=%s", WebWxInitURL, w.Request.BaseRequest.PassTicket)
	data, err := json.Marshal(w.Request)
	response, err := w.Client.Post(wxinitURL, ContentTypeJSON, bytes.NewReader(data))
	if err != nil {
		w.Log.Printf("get webwxinit :%v", WebWxInitURL)
		return
	}

	defer response.Body.Close()
	reader := response.Body.(io.Reader)
	// wxResponse := new(Response)
	if err = json.NewDecoder(reader).Decode(&w.Response); err != nil {
		w.Log.Printf("webwxinit: %+v", err)
		return
	}
	for _, contact := range w.Response.ContactList {
		w.InitContactList = append(w.InitContactList, contact)
	}
	w.ChatSet = strings.Split(w.Response.ChatSet, ",")
	w.User = w.Response.User
	w.SyncKeyStr = ""
	for i, item := range w.Response.SyncKey.List {
		if i == 0 {
			w.SyncKeyStr = strconv.Itoa(item.Key) + "_" + strconv.Itoa(item.Val)
			continue
		}

		w.SyncKeyStr += "|" + strconv.Itoa(item.Key) + "_" + strconv.Itoa(item.Val)

	}
	cookies := response.Cookies()
	w.Log.Printf("cookie num: %+v", len(cookies))
	for cookie := range cookies {
		w.Log.Printf("cookie : %+v", cookie)
	}
	jsonStr, err := json.MarshalIndent(w.Response, "", "")
	w.Log.Printf("webwxinit response : %+v", string(jsonStr))
	if w.Response.BaseResponse.Ret != StatusSuccess {
		return
	}
	return nil
}

// GetContactList GetContactList
func (w *Wechat) GetContactList() (contractResponse *ContractResponse, err error) {
	if !w.IsLogin() {
		return nil, fmt.Errorf("请重新登录")
	}
	wxurl := fmt.Sprintf("%s?pass_ticket=%s&skey=%s&r=%d",
		WebWxContactListURL,
		w.Request.BaseRequest.PassTicket,
		w.Request.BaseRequest.Skey,
		time.Now().Unix(),
	)

	data, err := json.Marshal(w.Request)
	response, err := w.Client.Post(wxurl, ContentTypeJSON, bytes.NewReader(data))
	if err != nil {
		w.Log.Printf("get webwxgetcontact :%v", WebWxInitURL)
		return nil, err
	}

	defer response.Body.Close()
	reader := response.Body.(io.Reader)
	wxResponse := new(MemberResp)
	if err = json.NewDecoder(reader).Decode(&wxResponse); err != nil {
		w.Log.Printf("webwxgetcontact: %+v", err)
		return nil, err
	}
	if w.Response.BaseResponse.Ret != StatusSuccess {
		return nil, fmt.Errorf(w.Response.BaseResponse.ErrMsg)
	}
	w.MemberList = wxResponse.MemberList
	w.MemberCount = wxResponse.Count
	for _, member := range w.MemberList {
		w.MemberMap[member.UserName] = member
		if member.UserName[:2] == "@@" {
			w.GroupMemberList = append(w.GroupMemberList, member) //群聊

		} else if member.VerifyFlag&8 != 0 {
			w.PublicUserList = append(w.PublicUserList, member) //公众号
		} else if member.UserName[:1] == "@" {
			w.ContactList = append(w.ContactList, member)
		}
	}
	mb := Member{}
	mb.NickName = w.User.NickName
	mb.UserName = w.User.UserName
	w.MemberMap[w.User.UserName] = mb
	jsonStr, err := json.MarshalIndent(w.Response, "", "")
	for _, user := range w.ChatSet {
		exist := false
		for _, initUser := range w.InitContactList {
			if user == initUser.UserName {
				exist = true
				break
			}
		}
		if !exist {
			value, ok := w.MemberMap[user]
			if ok {
				contact := User{
					UserName:  value.UserName,
					NickName:  value.NickName,
					Signature: value.Signature,
				}

				w.InitContactList = append(w.InitContactList, contact)
			}
		}

	}
	contractResponse = &ContractResponse{}
	contractResponse.GroupMemberList = w.GroupMemberList
	contractResponse.PublicUserList = w.PublicUserList
	contractResponse.ContactList = w.ContactList
	w.Log.Printf("webwxinit response : %+v", string(jsonStr))
	return
}

// SendMsg send message
func (w *Wechat) SendMsg(toUserName, message string, isFile bool) (err error) {
	if !w.IsLogin() {
		return fmt.Errorf("请重新登录")
	}
	wxurl := fmt.Sprintf("%s?pass_ticket=%s&skey=%s&r=%d",
		WebWxSendMsg,
		w.Request.BaseRequest.PassTicket,
		w.Request.BaseRequest.Skey,
		time.Now().Unix(),
	)
	clientMsgID := fmt.Sprintf("%d0%s", time.Now().Unix(), strconv.Itoa(rand.Int())[3:6])

	params := make(map[string]interface{})
	params["BaseRequest"] = w.Request.BaseRequest
	msg := make(map[string]interface{})
	msg["Type"] = 1
	msg["Content"] = message
	msg["FromUserName"] = w.User.UserName
	msg["LocalID"] = clientMsgID
	msg["ClientMsgId"] = clientMsgID
	msg["ToUserName"] = toUserName
	params["Msg"] = msg
	data, err := json.Marshal(params)
	response, err := w.Client.Post(wxurl, ContentTypeJSON, bytes.NewReader(data))
	if err != nil {
		w.Log.Printf("get SendMsg :%v", WebWxInitURL)
		return err
	}

	defer response.Body.Close()
	reader := response.Body.(io.Reader)
	wxResponse := new(MemberResp)
	if err = json.NewDecoder(reader).Decode(&wxResponse); err != nil {
		w.Log.Printf("SendMsg: %+v", err)
		return err
	}
	if w.Response.BaseResponse.Ret != StatusSuccess {
		return fmt.Errorf(w.Response.BaseResponse.ErrMsg)
	}

	return
}

// fetchuuID get uuid
func (w *Wechat) fetchuuID() (err error) {
	uuIDStr := "window.QRLogin.uuid"
	CodeStr := "window.QRLogin.code"
	params := url.Values{}
	params.Set("appid", AppID)
	params.Set("fun", "new")
	params.Set("lang", Lang)
	params.Set("_", strconv.FormatInt(time.Now().Unix(), 10))
	response, err := w.Client.Post(LoginURL, "", strings.NewReader(params.Encode()))

	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	wr := ParseJsResult(data)
	code := wr.Get(CodeStr)
	if code != "200" {
		return fmt.Errorf("GetuuID: not found uuid")
	}
	w.uuID = wr.Get(uuIDStr)
	w.Log.Printf("uuID:%s", w.uuID)
	return nil
}

// FetchQr  fetch login qrcode
func (w *Wechat) fetchQr() error {
	if w.uuID == "" {
		return fmt.Errorf("FetchQr: not found uuID")
	}
	w.qrImagePath = QrURL + w.uuID
	// params := url.Values{}
	// params.Set("t", "webwx")
	// params.Set("_", strconv.FormatInt(time.Now().Unix(), 10))
	// request, err := http.NewRequest(http.MethodPost, QrURL+w.uuID, strings.NewReader(params.Encode()))
	// request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// request.Header.Set("Cache-Control", "no-cache")

	// response, err := w.Client.Do(request)
	// if err != nil {
	// 	return fmt.Errorf("FetchQr: not found qrcode")
	// }
	// defer response.Body.Close()
	// data, err := ioutil.ReadAll(response.Body)
	// if err != nil {
	// 	return err
	// }
	// err = os.Mkdir(qrImagePath, 0777)
	// isExit := os.IsExist(err)
	// if !isExit && err != nil {
	// 	return fmt.Errorf("make dir faild, path:%s", qrImagePath)
	// }

	// filePath := filepath.Join(qrImagePath, w.uuID+".jpg")
	// f, err := os.Create(filePath)
	// if !os.IsExist(err) && err != nil {
	// 	return fmt.Errorf("make file faild, path:%s , %v", filePath, err)
	// }
	// n, err := f.Write(data)
	// if err != nil || n == 0 {
	// 	return fmt.Errorf("write qrcode faild")
	// }
	// w.qrImagePath = filePath

	// w.Log.Printf("qrImagePath:%s", w.qrImagePath)
	return nil
}

func (w *Wechat) setBaseURL() {
	url, err := url.Parse(w.redirectedURL)
	if err != nil {
		w.Log.Print(err)
	}
	w.baseURL = fmt.Sprintf("%s://%s%s", url.Scheme, url.Hostname(), url.Path)
}

// waitForLogin fetch login status
func (w *Wechat) waitForLogin() error {
	if w.uuID == "" {
		return nil
	}
	params := url.Values{}
	params.Add("uuid", w.uuID)
	params.Add("tip", "1")
	params.Add("_", strconv.FormatInt(time.Now().Unix(), 10))
	params.Encode()
	loginURL := FetchLoginURL + "?" + params.Encode()

	code := make(chan string)
	redirectedURL := make(chan string, 1)

	go func() { code <- "start" }()
	for {
		select {
		case success := <-code:
			if success != WxResultSuccessCode {
				time.Sleep(10 * time.Second)
				go w.fetchForLogin(loginURL, code, redirectedURL)
			} else {
				w.redirectedURL = <-redirectedURL
				w.Log.Printf("redirectedURL: %s", w.redirectedURL)
				return nil
			}
		case <-time.After(LoginTimeout * time.Second):
			w.Log.Print("Login is timeout")
			return fmt.Errorf("Login is timeout")
		}
	}
}

func (w *Wechat) fetchForLogin(url string, code, redirectedURL chan<- string) {
	response, err := w.Client.Get(url)
	if err != nil {
		code <- "faild"
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		code <- "faild"
	}
	result := ParseJsResult(data)
	windowCode := result.Get("window.code")
	if windowCode == WxResultSuccessCode {
		redirectedURL <- result.Get("window.redirect_uri")
	}
	code <- result.Get("window.code")
}

// getHTTPClient http client
func getHTTPClient() *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	transport := http.DefaultTransport.(*http.Transport)
	transport.ResponseHeaderTimeout = 1. * time.Minute
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	return &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   1 * time.Minute,
	}
}
