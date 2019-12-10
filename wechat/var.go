package wechat

import (
	"encoding/xml"
	"log"
	"net/http"
)

// const code
const (
	StatusSuccess       = 0
	WxResultSuccessCode = "200"
	LoginTimeout        = 50
)

// brower
const (
	UserAgent       = "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:57.0) Gecko/20100101 Firefox/57.0"
	Lang            = "zh-CN"
	ContentTypeJSON = "application/json; charset=UTF-8"
)

// path
var (
	qrImagePath = GetRootPath() + "/qrimages"
)

// hosts
const (
	SyncHostQQ                = "qq.com"
	SyncHostQQWXWebpush       = "webpush.wx.qq.com"
	SyncHostQQWXWebpush2      = "webpush2.wx.qq.com"
	SyncHostQQWX2             = "wx2.qq.com"
	SyncHostQQWX8             = "wx8.qq.com"
	SyncHostQQWX8Webpush      = "webpush.wx8.qq.com"
	SyncHostQQWebpush         = "webpush.wx2.qq.com"
	SyncHostQQWeiXinWebpush   = "webpush.weixin.qq.com"
	SyncHostWechat            = "wechat.com"
	SyncHostWechatWebpush     = "webpush.wechat.com"
	SyncHostWechatWebpush1    = "webpush1.wechat.com"
	SyncHostWechatWebpush2    = "webpush2.wechat.com"
	SyncHostWechatWebWebpush  = "webpush.web.wechat.com"
	SyncHostWechatWeb2        = "web2.wechat.com"
	SyncHostWechatWeb2Webpush = "webpush.web2.wechat.com"
)

// url
const (
	LoginURL = "https://login.weixin.qq.com/jslogin"
	QrURL    = "https://login.weixin.qq.com/qrcode/"

	FetchLoginURL       = "https://login.weixin.qq.com/cgi-bin/mmwebwx-bin/login"
	WxBaseURL           = "https://wx.qq.com/cgi-bin/mmwebwx-bin"
	WebWxInitURL        = WxBaseURL + "/webwxinit"
	WebWxContactListURL = WxBaseURL + "/webwxgetcontact"
	WebWxSendMsg        = WxBaseURL + "/webwxsendmsg"
	WebSyncCheckURL     = WxBaseURL + "/synccheck"
	WebUploadMediaURL   = WxBaseURL + "/webwxuploadmedia"
	WebSendMediaURL     = WxBaseURL + "/webwxsendmsgimg"
	TuringURL           = "http://www.tuling123.com/openapi/api"
)

// appID
const (
	// appID
	AppID        = "wx782c26e4c19acffb"
	AppKEY       = "391ad66ebad2477b908dce8e79f101e7"
	TUringUserID = "abc123"
	// DeviceID
	deviceID = "e123456789002237"
)

// Wechat struct
type Wechat struct {
	User            User
	Users           []string
	Debug           bool
	deviceID        string
	uuID            string
	qrImagePath     string
	baseURL         string
	redirectedURL   string
	Client          *http.Client
	Request         *Request
	Response        Response
	MemberList      []Member
	ChatSet         []string
	SyncKeyStr      string
	ContactList     []Member
	InitContactList []User //谈话的人
	MemberMap       map[string]Member
	GroupMemberList []Member //群友
	PublicUserList  []Member //公众号
	SpecialUserList []Member //特殊账号
	GroupList       []string
	MemberCount     int
	Log             *log.Logger
}

// BaseRequest login xml response
type BaseRequest struct {
	XMLName     xml.Name `xml:"error" json:"-"`
	Ret         int      `xml:"ret" json:"-"`
	Message     string   `xml:"message" json:"-"`
	Skey        string   `xml:"skey" json:"Skey"`
	Wxsid       string   `xml:"wxsid" json:"Sid"`
	Wxuin       int64    `xml:"wxuin" json:"Uin"`
	PassTicket  string   `xml:"pass_ticket" json:"-"`
	DeviceID    string   `xml:"-" json:"DeviceID"`
	IsGrayScale int      `xml:"isgrayscale" json:"-"`
}

// Request Request
type Request struct {
	BaseRequest   *BaseRequest
	MemberCount   int    `json:",omitempty"`
	MemberList    []User `json:",omitempty"`
	Topic         string `json:",omitempty"`
	ChatRoomName  string `json:",omitempty"`
	DelMemberList string `json:",omitempty"`
	AddMemberList string `json:",omitempty"`
	ClientMediaID string `json:",omitempty"`
	TotalLen      int64  `json:",omitempty"`
	StartPos      int    `json:",omitempty"`
	DataLen       int64  `json:",omitempty"`
	MediaType     int    `json:",omitempty"`
}

// User user struct
type User struct {
	UserName          string `json:"UserName"`
	Uin               int64  `json:"Uin"`
	NickName          string `json:"NickName"`
	HeadImgURL        string `json:"HeadImgUrl" xml:""`
	RemarkName        string `json:"RemarkName" xml:""`
	PYInitial         string `json:"PYInitial" xml:""`
	PYQuanPin         string `json:"PYQuanPin" xml:""`
	RemarkPYInitial   string `json:"RemarkPYInitial" xml:""`
	RemarkPYQuanPin   string `json:"RemarkPYQuanPin" xml:""`
	HideInputBarFlag  int    `json:"HideInputBarFlag" xml:""`
	StarFriend        int    `json:"StarFriend" xml:""`
	Sex               int    `json:"Sex" xml:""`
	Signature         string `json:"Signature" xml:""`
	AppAccountFlag    int    `json:"AppAccountFlag" xml:""`
	VerifyFlag        int    `json:"VerifyFlag" xml:""`
	ContactFlag       int    `json:"ContactFlag" xml:""`
	WebWxPluginSwitch int    `json:"WebWxPluginSwitch" xml:""`
	HeadImgFlag       int    `json:"HeadImgFlag" xml:""`
	SnsFlag           int    `json:"SnsFlag" xml:""`
}

// SyncKey SyncKey
type SyncKey struct {
	Count int      `json:"Count"`
	List  []KeyVal `json:"List"`
}

// KeyVal KeyVal
type KeyVal struct {
	Key int `json:"Key"`
	Val int `json:"Val"`
}

// BaseResponse BaseResponse
type BaseResponse struct {
	Ret    int
	ErrMsg string
}

// InnerResponse InnerResponse
type InnerResponse struct {
	BaseResponse *BaseResponse `json:"BaseResponse"`
}

// Response Response
type Response struct {
	InnerResponse
	User                User    `json:"User"`
	Count               int     `json:"Count"`
	ContactList         []User  `json:"ContactList"`
	SyncKey             SyncKey `json:"SyncKey"`
	ChatSet             string  `json:"ChatSet"`
	SKey                string  `json:"SKey"`
	ClientVersion       int     `json:"ClientVersion"`
	SystemTime          int     `json:"SystemTime"`
	GrayScale           int     `json:"GrayScale"`
	InviteStartCount    int     `json:"InviteStartCount"`
	MPSubscribeMsgCount int     `json:"MPSubscribeMsgCount"`
	//MPSubscribeMsgList  string  `json:"MPSubscribeMsgList"`
	ClickReportInterval int `json:"ClickReportInterval"`
}

// Member member
type Member struct {
	Uin              int64 `json:"-"`
	UserName         string
	NickName         string
	HeadImgURL       string
	ContactFlag      int    `json:"-"`
	MemberCount      int    `json:"-"`
	MemberList       []User `json:"-"`
	RemarkName       string `json:"-"`
	HideInputBarFlag int    `json:"-"`
	Sex              int    `json:"-"`
	Signature        string `json:"-"`
	VerifyFlag       int    `json:"-"`
	OwnerUin         int    `json:"-"`
	PYInitial        string `json:"-"`
	PYQuanPin        string `json:"-"`
	RemarkPYInitial  string `json:"-"`
	RemarkPYQuanPin  string `json:"-"`
	StarFriend       int    `json:"-"`
	AppAccountFlag   int    `json:"-"`
	Statues          int    `json:"-"`
	AttrStatus       int    `json:"-"`
	Province         string `json:"-"`
	City             string `json:"-"`
	Alias            string `json:"-"`
	SnsFlag          int    `json:"-"`
	UniFriend        int    `json:"-"`
	DisplayName      string `json:"-"`
	ChatRoomID       int    `json:"-"`
	KeyWord          string `json:"-"`
	EncryChatRoomID  string `json:"-"`
}

// MemberResp MemberResp
type MemberResp struct {
	Response
	MemberCount  int
	ChatRoomName string
	MemberList   []Member
	Seq          int
}

// ContractResponse ContractResponse
type ContractResponse struct {
	GroupMemberList []Member `json:"groupMembers"`
	PublicUserList  []Member `json:"-"`
	ContactList     []Member `json:"contacts"`
}

// SyncCheckResp sync check resp
type SyncCheckResp struct {
	RetCode  int `json:"retcode"`
	Selector int `json:"selector"`
}

// SyncParams sync params
type SyncParams struct {
	BaseRequest BaseRequest `json:"BaseRequest"`
	SyncKey     SyncKey     `json:"SyncKey"`
	RR          int64       `json:"rr"`
}

// SyncResp sync response
type SyncResp struct {
	Response
	SyncKey      SyncKey       `json:"SyncKey"`
	ContinueFlag int           `json:"ContinueFlag"`
	AddMsgList   []interface{} `json:"AddMsgList"`
}

// MediaResponse MediaResponse
type MediaResponse struct {
	Response
	MediaID           string `json:"MediaId"`
	StartPos          int    `json:"StartPos"`
	CDNThumbImgHeight int    `json:"CDNThumbImgHeight"`
	CDNThumbImgWidth  int    `json:"CDNThumbImgWidth"`
	EncryFileName     string `json:"EncryFileName"`
}
