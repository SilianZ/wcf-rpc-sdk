package wcf

import (
	"fmt"
	"testing"
	"time"
)

const testAddr = "tcp://192.168.150.128:10086" // 请修改为你的 WCF 服务端地址

func TestClient_IsLogin(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	if !c.IsLogin() {
		t.Errorf("IsLogin() = false, want true")
	}
}

func TestClient_GetSelfWXID(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	wxid := c.GetSelfWXID()
	if wxid == "" {
		t.Errorf("GetSelfWXID() = \"\", want non-empty string")
	}
	fmt.Println("SelfWXID:", wxid) // 打印获取到的 WXID
}

func TestClient_GetMsgTypes(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	types := c.GetMsgTypes()
	if len(types) == 0 {
		t.Errorf("GetMsgTypes() returned empty map")
	}
	fmt.Println("MsgTypes:", types) // 打印消息类型
}

func TestClient_GetContacts(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	contacts := c.GetContacts()
	if len(contacts) == 0 {
		t.Errorf("GetContacts() returned empty list")
	}
	fmt.Println("Contacts:", contacts) // 打印联系人列表
}

func TestClient_GetDBNames(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	dbNames := c.GetDBNames()
	if len(dbNames) == 0 {
		t.Errorf("GetDBNames() returned empty list")
	}
	fmt.Println("DB Names:", dbNames)
}

func TestClient_GetDBTables(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	dbNames := c.GetDBNames()
	if len(dbNames) == 0 {
		t.Skip("No databases found, skipping GetDBTables test")
	}

	tables := c.GetDBTables(dbNames[0]) // 获取第一个数据库的表
	if len(tables) == 0 {
		t.Errorf("GetDBTables() returned empty list")
	}
	fmt.Println("Tables:", tables)
}

func TestClient_ExecDBQuery(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	dbNames := c.GetDBNames()
	if len(dbNames) == 0 {
		t.Skip("No databases found, skipping ExecDBQuery test")
	}

	// 测试查询 Contact 表
	rows := c.ExecDBQuery("MicroMsg.db", "SELECT UserName, NickName FROM Contact limit 10;")
	if len(rows) == 0 {
		t.Errorf("ExecDBQuery() returned empty list")
	}
	fmt.Println("Query Result:", rows)
}

func TestClient_SendTxt(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	wxid := c.GetSelfWXID()
	if wxid == "" {
		t.Skip("SelfWXID is empty, skipping SendTxt test")
	}

	status := c.SendTxt("Test from Go", wxid, nil)
	if status != 0 {
		t.Errorf("SendTxt() = %v, want 0", status)
	}
}

func TestClient_EnableRecvTxt(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	status := c.EnableRecvTxt()
	if status != 0 {
		t.Errorf("EnableRecvTxt() = %v, want 0", status)
	}

	if !c.RecvTxt {
		t.Errorf("EnableRecvTxt() RecvTxt not set to true")
	}
}

func TestClient_DisableRecvTxt(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	c.EnableRecvTxt() // 先启用接收消息

	status := c.DisableRecvTxt()
	if status != 0 {
		t.Errorf("DisableRecvTxt() = %v, want 0", status)
	}

	if c.RecvTxt {
		t.Errorf("DisableRecvTxt() RecvTxt not set to false")
	}
}

func TestClient_OnMSG(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	c.EnableRecvTxt() // 启用接收消息

	msgChan := make(chan *WxMsg)
	go func() {
		err := c.OnMSG(func(msg *WxMsg) {
			msgChan <- msg
		})
		if err != nil {
			t.Errorf("OnMSG() error = %v", err)
		}
	}()

	// 等待接收消息，或者超时
	select {
	case msg := <-msgChan:
		fmt.Println("Received message:", msg)
	case <-time.After(20 * time.Second): // 设置一个合理的超时时间
		t.Errorf("OnMSG() timed out waiting for message")
	}
}

func TestClient_GetUserInfo(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	ui := c.GetUserInfo()
	if ui == nil {
		t.Errorf("GetUserInfo() returned nil")
	}
	fmt.Println("User Info:", ui)
}

func TestClient_RefreshPYQ(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	status := c.RefreshPYQ()
	if status != 1 {
		t.Errorf("RefreshPYQ() = %v, want 1", status)
	}
}

func TestClient_AddChatRoomMembers(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	contacts := c.GetContacts()
	if len(contacts) < 2 {
		t.Skip("Not enough contacts to test AddChatRoomMembers, skipping")
	}

	// 使用前两个联系人作为测试
	wxids := []string{"wxid_pagpb98c6nj722", "wxid_jj4mhsji9tjk22"}

	// 假设你有一个测试群，请替换为你的测试群 ID
	roomID := "45959390469@chatroom" // 请替换为你的测试群 ID

	status := c.AddChatRoomMembers(roomID, wxids)
	if status != 1 {
		t.Errorf("AddChatRoomMembers() = %v, want 1", status)
	}
}

func TestClient_InvChatRoomMembers(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	contacts := c.GetContacts()
	if len(contacts) < 2 {
		t.Skip("Not enough contacts to test InvChatRoomMembers, skipping")
	}

	// 使用前两个联系人作为测试
	wxids := []string{"wxid_jj4mhsji9tjk22"}

	// 假设你有一个测试群，请替换为你的测试群 ID
	roomID := "45959390469@chatroom" // 请替换为你的测试群 ID

	status := c.InvChatRoomMembers(roomID, wxids)
	if status != 1 {
		t.Errorf("InvChatRoomMembers() = %v, want 1", status)
	}
}

func TestClient_DelChatRoomMembers(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	contacts := c.GetContacts()
	if len(contacts) < 2 {
		t.Skip("Not enough contacts to test DelChatRoomMembers, skipping")
	}

	// 使用前两个联系人作为测试
	wxids := []string{"wxid_jj4mhsji9tjk22"}

	// 假设你有一个测试群，请替换为你的测试群 ID
	roomID := "45959390469@chatroom" // 请替换为你的测试群 ID

	status := c.DelChatRoomMembers(roomID, wxids)
	if status != 1 {
		t.Errorf("DelChatRoomMembers() = %v, want 1", status)
	}
}

func TestClient_AcceptFriend(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	// 这些值需要根据实际的好友请求消息来填写
	v3 := "v3_020b3826fd0301000000000049781a096f1ed4000000501ea9a3dba12f95f6b60a0536a1adb60fbd06afd0a1b0587bc4a43c37cf1bfeec110f30c755650c5493fbc713ce869828f7645b3a1600f18d507e13170bac48665cd047f69348941a1c819c6b@stranger"                                                                                                             // 请替换为实际的 v3 值
	v4 := "v4_000b708f0b04000001000000000057f883922de57cd70175217187671000000050ded0b020927e3c97896a09d47e6e9e550a019a56af47b5593b6812f6d50bc071a677c39ff103d9c04656371318849ecaa951b561b37cd20b073b0340f14bbfb0df83dfb565a29ee993d2bca44ae89aadd1bb17a3d6d96f496025e66b7c019b747dbd069744b68ec3386563add41ff12af5dded467e88920d@stranger" // 请替换为实际的 v4 值
	scene := int32(30)                                                                                                                                                                                                                                                                                                                     // 请根据实际情况修改

	status := c.AcceptFriend(v3, v4, scene)
	if status != 1 {
		t.Errorf("AcceptFriend() = %v, want 1", status)
	}
}

func TestClient_ReceiveTransfer(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	// 这些值需要根据实际的转账消息来填写
	wxid := "transfer_sender_wxid" // 请替换为实际的转账发送者 wxid
	tfid := "transfer_id"          // 请替换为实际的 transferid
	taid := "transaction_id"       // 请替换为实际的 transactionid

	status := c.ReceiveTransfer(wxid, tfid, taid)
	if status != 1 {
		t.Errorf("ReceiveTransfer() = %v, want 1", status)
	}
}

func TestClient_DecryptImage(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	// 请替换为实际的加密图片路径和目标路径
	src := "C:/Users/aimi/Documents/WeChat Files/wxid_p5z4fuhnbdgs22/FileStorage/MsgAttach/84d8449549662bc200b18aabcf977f3a/Image/2025-01/31d611357fdf46f144cea07cd9abd2c6.dat" // 请替换为实际的加密图片路径
	dst := "C:/Users/aimi/Documents/WeChat Files/wxid_p5z4fuhnbdgs22/FileStorage/MsgAttach/84d8449549662bc200b18aabcf977f3a/Image/2025-01/test.jpg"                             // 请替换为实际的解密图片保存路径

	decryptedPath := c.DecryptImage(src, dst)
	if decryptedPath == "" {
		t.Errorf("DecryptImage() returned empty string")
	}
	fmt.Println("Decrypted image path:", decryptedPath)
}

func TestClient_SendIMG(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	wxid := c.GetSelfWXID()
	if wxid == "" {
		t.Skip("SelfWXID is empty, skipping SendIMG test")
	}

	// 请替换为实际的图片路径
	imgPath := "C:\\images\\test.jpg" // 请替换为你的图片路径

	status := c.SendIMG(imgPath, wxid)
	if status != 0 {
		t.Errorf("SendIMG() = %v, want 0", status)
	}
}

func TestClient_SendFile(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	wxid := c.GetSelfWXID()
	if wxid == "" {
		t.Skip("SelfWXID is empty, skipping SendFile test")
	}

	// 请替换为实际的文件路径
	filePath := "path/to/your/file.txt" // 请替换为你的文件路径

	status := c.SendFile(filePath, wxid)
	if status != 0 {
		t.Errorf("SendFile() = %v, want 0", status)
	}
}

func TestClient_SendRichText(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	wxid := c.GetSelfWXID()
	if wxid == "" {
		t.Skip("SelfWXID is empty, skipping SendRichText test")
	}

	// 请根据实际情况修改参数
	status := c.SendRichText("Name", "gh_account", "Title", "Digest", "https://example.com", "https://example.com/thumb.jpg", wxid)
	if status != 1 {
		t.Errorf("SendRichText() = %v, want 0", status)
	}
}

func TestClient_SendXml(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	wxid := c.GetSelfWXID()
	if wxid == "" {
		t.Skip("SelfWXID is empty, skipping SendXml test")
	}

	// 请根据实际情况修改参数
	xmlContent := "<appmsg appid=\\\"wx8dd6ecd81906fd84\\\" sdkver=\\\"0\\\">\\n\\t\\t<title>北宇治四重奏 第4番 トランペット (北宇治四重奏 第四章 小号)</title>\\n\\t\\t<des>安済知佳 - TVアニメ『響け！ユーフォニアム』キャラクターソング Vol.4</des>\\n\\t\\t<type>5</type>\\n\\t\\t<url>https://y.music.163.com/m/song?id=33789233&amp;fx-wxqd=c&amp;playerUIModeId=76001&amp;userid=3893734548&amp;app_version=9.2.30&amp;shareToken=38937345481736867627_86fabb98f86cce8cf26572d81c6cbd90&amp;fx-wechatnew=t1&amp;fx-wordtest=t4&amp;fx-listentest=t3&amp;PlayerStyles_SynchronousSharing=t3&amp;dlt=0846&amp;H5_DownloadVIPGift=</url>\\n\\t\\t<appattach>\\n\\t\\t\\t<cdnthumburl>3057020100044b304902010002045192ec6902032dd343020419161c6f020467867f32042464333834393239622d643931652d343433612d383935372d6433633761323063356639310204011408030201000405004c550700</cdnthumburl>\\n\\t\\t\\t<cdnthumbmd5>8ec20afe57f1e23f669f9fdc311bb27a</cdnthumbmd5>\\n\\t\\t\\t<cdnthumblength>8437</cdnthumblength>\\n\\t\\t\\t<cdnthumbwidth>135</cdnthumbwidth>\\n\\t\\t\\t<cdnthumbheight>135</cdnthumbheight>\\n\\t\\t\\t<cdnthumbaeskey>e3c87be21e79f2e956891618bf18f0be</cdnthumbaeskey>\\n\\t\\t\\t<aeskey>e3c87be21e79f2e956891618bf18f0be</aeskey>\\n\\t\\t\\t<encryver>0</encryver>\\n\\t\\t\\t<filekey>wxid_p5z4fuhnbdgs22_108_1736867633</filekey>\\n\\t\\t</appattach>\\n\\t\\t<md5>8ec20afe57f1e23f669f9fdc311bb27a</md5>\\n\\t\\t<statextstr>GhQKEnd4OGRkNmVjZDgxOTA2ZmQ4NA==</statextstr>\\n\\t</appmsg>" // 请替换为你的 XML 内容
	status := c.SendXml("", xmlContent, wxid, 49)
	if status != 0 {
		t.Errorf("SendXml() = %v, want 0", status)
	}
}

func TestClient_SendEmotion(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	wxid := c.GetSelfWXID()
	if wxid == "" {
		t.Skip("SelfWXID is empty, skipping SendEmotion test")
	}

	// 请替换为实际的表情图片路径
	emotionPath := "path/to/your/emotion.gif" // 请替换为你的表情图片路径

	status := c.SendEmotion(emotionPath, wxid)
	if status != 0 {
		t.Errorf("SendEmotion() = %v, want 0", status)
	}
}

func TestClient_SendPat(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	// 假设你有一个测试群和群成员，请替换为你的测试群 ID 和成员 wxid
	roomID := "your_test_room_id" // 请替换为你的测试群 ID
	wxid := "member_wxid_in_room" // 请替换为你要拍的群成员 wxid

	status := c.SendPat(roomID, wxid)
	if status != 1 {
		t.Errorf("SendPat() = %v, want 1", status)
	}
}

func TestClient_DownloadAttach(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	// 这些值需要根据实际的消息来填写
	msgID := uint64(5837917763151228085)                                                                                                                                            // 请替换为实际的消息 ID
	thumb := "C:/Users/aimi/Documents/WeChat Files/wxid_p5z4fuhnbdgs22/FileStorage/MsgAttach/84d8449549662bc200b18aabcf977f3a/Thumb/2025-01/5eefe5f664d26314675c45e643a32dd5_t.dat" // 请替换为实际的 thumb 路径
	extra := "C:/Users/aimi/Documents/WeChat Files/wxid_p5z4fuhnbdgs22/FileStorage/MsgAttach/84d8449549662bc200b18aabcf977f3a/Image/2025-01/39cd1240e4aa4609ffad8481c3d91584.dat"   // 请替换为实际的 extra 信息

	status := c.DownloadAttach(msgID, thumb, extra)
	if status != 0 {
		t.Errorf("DownloadAttach() = %v, want 0", status)
	}
}

func TestClient_ForwardMsg(t *testing.T) {
	c, err := NewWCF(testAddr)
	if err != nil {
		t.Fatalf("NewWCF() error = %v", err)
	}
	defer c.Close()

	// 这些值需要根据实际的消息来填写
	msgID := uint64(12345)      // 请替换为实际的消息 ID
	receiver := "receiver_wxid" // 请替换为实际的消息接收者 wxid

	status := c.ForwardMsg(msgID, receiver)
	if status != 1 {
		t.Errorf("ForwardMsg() = %v, want 1", status)
	}
}
