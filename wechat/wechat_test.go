package wechat

import "testing"

func TestUploadMedia(t *testing.T) {
	wechat := NewWechat(GetLogger())
	wechat.Request.BaseRequest.PassTicket = "hahaha"
	wechat.uuID = "1234"
	mediaID, err := wechat.UploadMedia("/home/madison/Downloads/timg.jpeg")
	if err != nil {
		t.Errorf("%s : %+v", mediaID, err)
	}
	t.Fail()
}
