package wechat

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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

// Login login
func (w *Wechat) Login(ctx context.Context) (err error) {
	w.Log.Printf("%s Login start", w.GetUUID())
	err = w.login(ctx)
	if err != nil {
		w.Log.Printf("%s Login faild： error:%s", w.GetUUID(), err.Error())
		return
	}

	w.Log.Printf("%s webwxinit start", w.GetUUID())
	err = w.webwxinit()
	if err != nil {
		w.Log.Printf("%s webwxinit faild： error:%s", w.GetUUID(), err.Error())
		return
	}
	w.Log.Printf("%s Login success", w.GetUUID())
	return
}

// login fetch common params
func (w *Wechat) login(ctx context.Context) (err error) {
	err = w.waitForLogin(ctx)
	if err != nil {
		w.Log.Printf("%s login faild： error:%s", w.GetUUID(), err.Error())
		return
	}
	if w.redirectedURL == "" {
		return
	}
	response, err := w.Client.Get(w.redirectedURL + "&fun=new")
	if err != nil {
		w.Log.Printf("%s login faild： error:%s", w.GetUUID(), err.Error())
		return
	}
	defer response.Body.Close()
	reader := response.Body.(io.Reader)
	if err = xml.NewDecoder(reader).Decode(w.Request.BaseRequest); err != nil {
		w.Log.Printf("%s login faild： error:%s", w.GetUUID(), err.Error())
		return
	}
	w.Request.BaseRequest.DeviceID = w.deviceID
	w.Log.Printf("login success:%+v", w.Request.BaseRequest)
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
	w.Log.Printf("%s GetContactList start", w.GetUUID())
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
		w.Log.Printf("%s GetContactList faild: %s", w.GetUUID(), err.Error())
		return nil, err
	}

	defer response.Body.Close()
	reader := response.Body.(io.Reader)
	wxResponse := new(MemberResp)
	if err = json.NewDecoder(reader).Decode(&wxResponse); err != nil {
		w.Log.Printf("%s GetContactList parse json faild: %s", w.GetUUID(), err.Error())
		w.Log.Printf("webwxgetcontact: %+v", err)
		return nil, err
	}
	respjson, err := json.Marshal(wxResponse)
	w.Log.Printf("%s GetContactList resp: %s", w.GetUUID(), string(respjson))
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
	// 公众号先不返回了
	// contractResponse.PublicUserList = w.PublicUserList
	contractResponse.ContactList = w.ContactList
	w.Log.Printf("GetContactList response : %+v", string(jsonStr))
	return
}

// SendMsg send message
func (w *Wechat) SendMsg(toUserName, message string, isFile bool) (err error) {
	if !w.IsLogin() {
		return fmt.Errorf("请重新登录")
	}
	w.Log.Printf("%s sendMsg: toUserName:%s;message:%s", w.GetUUID(), toUserName, message)
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
		w.Log.Printf("%s get SendMsg faild:%s", w.GetUUID(), err.Error())
		return err
	}

	defer response.Body.Close()
	reader := response.Body.(io.Reader)
	wxResponse := new(MemberResp)
	if err = json.NewDecoder(reader).Decode(&wxResponse); err != nil {
		w.Log.Printf("%s json decode SendMsg: %+v", w.GetUUID(), err)
		return err
	}
	if w.Response.BaseResponse.Ret != StatusSuccess {
		return fmt.Errorf(w.Response.BaseResponse.ErrMsg)
	}
	w.Log.Printf("%s SendMsg success", w.GetUUID())
	return
}

// SendMedia 发送图片
func (w *Wechat) SendMedia(toUserName, mediaPath string) error {
	if !w.IsLogin() {
		return fmt.Errorf("请重新登录")
	}
	w.Log.Printf("%s SendMedia: toUserName:%s;mediaPath:%s", w.GetUUID(), toUserName, mediaPath)

	mediaID, err := w.UploadMedia(mediaPath)
	if err != nil {
		w.Log.Printf("%s UploadMedia faild: mediaPath=%s", w.GetUUID(), mediaPath)
		return err
	}
	wxurl := fmt.Sprintf("%s?pass_ticket=%s&fun=async&skey=%s&r=%d",
		WebSendMediaURL,
		w.Request.BaseRequest.PassTicket,
		w.Request.BaseRequest.Skey,
		time.Now().Unix(),
	)
	clientMsgID := fmt.Sprintf("%d0%s", time.Now().Unix(), strconv.Itoa(rand.Int())[3:6])
	params := make(map[string]interface{})
	params["BaseRequest"] = w.Request.BaseRequest
	msg := make(map[string]interface{})
	msg["Type"] = 1
	msg["Content"] = ""
	msg["MediaId"] = mediaID
	msg["FromUserName"] = w.User.UserName
	msg["LocalID"] = clientMsgID
	msg["ClientMsgId"] = clientMsgID
	msg["ToUserName"] = toUserName
	params["Msg"] = msg
	data, err := json.Marshal(params)
	response, err := w.Client.Post(wxurl, ContentTypeJSON, bytes.NewReader(data))
	if err != nil {
		w.Log.Printf("%s get SendMsg faild:%s", w.GetUUID(), err.Error())
		return err
	}

	defer response.Body.Close()
	reader := response.Body.(io.Reader)
	wxResponse := new(MemberResp)
	if err = json.NewDecoder(reader).Decode(&wxResponse); err != nil {
		w.Log.Printf("%s json decode SendMsg: %+v", w.GetUUID(), err)
		return err
	}
	if w.Response.BaseResponse.Ret != StatusSuccess {
		return fmt.Errorf(w.Response.BaseResponse.ErrMsg)
	}
	w.Log.Printf("%s SendMsg success", w.GetUUID())
	return nil
}

// UploadMedia 上传图片
func (w *Wechat) UploadMedia(mediaPath string) (mediaID string, err error) {

	if !w.IsLogin() {
		err = fmt.Errorf("请重新登录")
		return
	}
	w.Log.Printf("%s UploadMedia: mediaPath:%s", w.GetUUID(), mediaPath)
	f, err := os.Open(mediaPath)
	if err != nil {
		return
	}
	fStat, err := f.Stat()
	if err != nil {
		return
	}
	bodyBuf := new(bytes.Buffer)
	bodyWriter := multipart.NewWriter(bodyBuf)
	_, filename := filepath.Split(mediaPath)
	fw, err := bodyWriter.CreateFormFile("filename", filename)
	if _, err = io.Copy(fw, f); err != nil {
		return
	}
	if err != nil {
		w.Log.Printf("UploadMedia: %s bodyWriter.CreateFormFile faild %+v", w.GetUUID(), err)
		return
	}
	fInfo := strings.Split(filename, ".")
	if len(fInfo) != 2 {
		err = fmt.Errorf("文件没有后缀")
	}
	ext := fInfo[1]
	fw, _ = bodyWriter.CreateFormField("id")
	fw.Write([]byte("WU_FILE_0"))
	fw, _ = bodyWriter.CreateFormField("name")
	fw.Write([]byte(filename))
	fw, _ = bodyWriter.CreateFormField("type")
	if ext == "gif" {
		fw.Write([]byte("image/gif"))
	} else {
		fw.Write([]byte("image/jpeg"))
	}
	fw, _ = bodyWriter.CreateFormField("lastModifieDate")
	fw.Write([]byte("Mon Feb 13 2017 17:27:23 GMT+8000(CST)"))
	fw, _ = bodyWriter.CreateFormField("size")

	fw.Write([]byte(strconv.FormatInt(fStat.Size(), 10)))
	fw, _ = bodyWriter.CreateFormField("mediatype")
	if ext == "gif" {
		fw.Write([]byte("doc"))
	} else {
		fw.Write([]byte("pic"))
	}
	if err != nil {
		w.Log.Printf("UploadMedia: %s open faild %+v", w.GetUUID(), err)
		return
	}

	wxurl := fmt.Sprintf("%s?pass_ticket=%s&fun=async&skey=%s&r=%d",
		WebSendMediaURL,
		w.Request.BaseRequest.PassTicket,
		w.Request.BaseRequest.Skey,
		time.Now().Unix(),
	)
	uploadResp := Request{
		BaseRequest:   w.Request.BaseRequest,
		TotalLen:      fStat.Size(),
		StartPos:      0,
		DataLen:       fStat.Size(),
		ClientMediaID: mediaID,
	}
	jur, err := json.Marshal(uploadResp)
	fw, _ = bodyWriter.CreateFormField("uploadmediarequest")
	fw.Write(jur)
	fw, _ = bodyWriter.CreateFormField("webwx_data_ticket")
	req, err := http.NewRequest(http.MethodPost, wxurl, bodyBuf)
	req.Header.Add("Content-Type", bodyWriter.FormDataContentType())
	req.Header.Add("User-Agent", UserAgent)

	if err != nil {
		return
	}
	fmt.Println(wxurl)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	mediaResp := new(MediaResponse)
	if err = json.NewDecoder(resp.Body).Decode(mediaResp); err != nil {
		return
	}

	return mediaResp.MediaID, nil
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
func (w *Wechat) waitForLogin(ctx context.Context) error {
	w.Log.Printf("%s waitForLogin start", w.GetUUID())

	if w.uuID == "" {
		return nil
	}
	params := url.Values{}
	params.Add("uuid", w.uuID)
	params.Add("tip", "1")
	params.Add("_", strconv.FormatInt(time.Now().Unix(), 10))
	params.Encode()
	loginURL := FetchLoginURL + "?" + params.Encode()

	loginTicker := time.NewTicker(5 * time.Second)
	fmt.Println(loginURL, loginTicker)
	for {
		select {
		case <-loginTicker.C:
			go func(ctx context.Context) {
				fmt.Println(time.Now())
				redirectedURL, err := w.fetchForLogin(loginURL)
				if err != nil {
					return
				}
				w.redirectedURL = redirectedURL
			}(ctx)
			if w.redirectedURL != "" {
				return nil
			}
		case <-ctx.Done():
			w.Log.Printf("%s waitForLogin faild: 登录超时", w.GetUUID())
			return fmt.Errorf("登录超时")
		}
	}
}

func (w *Wechat) fetchForLogin(url string) (redirectedURL string, er error) {
	w.Log.Printf("%s fetchForLogin start", w.GetUUID())
	response, err := w.Client.Get(url)
	if err != nil {
		w.Log.Printf("%s fetchForLogin faild: %s", w.GetUUID(), err.Error())
		return
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		w.Log.Printf("%s fetchForLogin faild: %s", w.GetUUID(), err.Error())
		return
	}
	w.Log.Println(string(data))
	result := ParseJsResult(data)
	windowCode := result.Get("window.code")
	if windowCode == WxResultSuccessCode {
		redirectedURL = result.Get("window.redirect_uri")
	}
	w.Log.Printf("%s fetchForLogin success", w.GetUUID())
	return
}

// SyncCheck sync check
func (w *Wechat) SyncCheck() (syncResp *SyncCheckResp, err error) {
	w.Log.Printf("SyncCheck: %s start", w.GetUUID())
	params := url.Values{}
	curTime := strconv.FormatInt(time.Now().Unix(), 10)
	params.Set("r", curTime)
	params.Set("sid", w.Request.BaseRequest.Wxsid)
	params.Set("uin", strconv.FormatInt(int64(w.Request.BaseRequest.Wxuin), 10))
	params.Set("sky", w.Request.BaseRequest.Skey)
	params.Set("deviceid", w.SyncKeyStr)
	params.Set("_", curTime)
	checkURL, err := url.Parse(WebSyncCheckURL)
	if err != nil {
		w.Log.Printf("SyncCheck: %s faild: %+v", w.GetUUID(), err)
		return
	}
	checkURL.RawQuery = params.Encode()
	w.Log.Printf(checkURL.String())
	resp, err := w.Client.Get(checkURL.String())
	if err != nil {
		w.Log.Printf("SyncCheck: %s get faild: %+v", w.GetUUID(), err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.Log.Printf("SyncCheck: %s read body faild: %+v", w.GetUUID(), err)
		return
	}
	bodyStr := string(body)
	w.Log.Println(bodyStr)

	syncResp = &SyncCheckResp{}
	regexCompile := regexp.MustCompile(`window.synccheck={retcode:"(\d+)",selector:"(\d+)"}`)
	pmSub := regexCompile.FindStringSubmatch(bodyStr)
	w.Log.Printf("the data:%+v", pmSub)
	if len(pmSub) != 0 {
		syncResp.RetCode, err = strconv.Atoi(pmSub[1])
		syncResp.Selector, err = strconv.Atoi(pmSub[2])
		w.Log.Printf("sync resp: %+v", resp)
	} else {
		err = fmt.Errorf("regex error in window.redirect_uri")
		return
	}
	return
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
