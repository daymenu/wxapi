package wechat

import (
	"testing"
)

func TestParseWechatResult(t *testing.T) {
	result := `window.code=200;
	window.redirect_uri="https://wx.qq.com/cgi-bin/mmwebwx-bin/webwxnewloginpage?ticket=AZIpMcRh4hpDS7IUqoQhHhqe@qrticket_0&uuid=IcyiwdKOXg==&lang=zh-CN&scan=1574520693";`
	wr := ParseJsResult([]byte(result))
	if v := wr.Get("window.code"); v != "200" {
		t.Fatalf("%s", v)
	}
	if v := wr.Get("window.redirect_uri"); v != "https://wx.qq.com/cgi-bin/mmwebwx-bin/webwxnewloginpage?ticket=AZIpMcRh4hpDS7IUqoQhHhqe@qrticket_0&uuid=IcyiwdKOXg==&lang=zh-CN&scan=1574520693" {
		t.Fatalf("%#v", wr)
	}
}
