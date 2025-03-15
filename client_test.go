package wcf_rpc_sdk

import (
	"path/filepath"
	"testing"
	"time"
)

// TestClient_Recv 持续接收消息
func TestClient_Recv(t *testing.T) {
	// 创建客户端实例
	cli := NewClient(10, false, false)

	// 启动客户端，这里假设不需要自动注入微信
	cli.Run(false)
	// 关闭客户端
	defer cli.Close()
	for msg := range cli.GetMsgChan() {
		t.Log(msg)
	}
}

func TestClient_SendTextAndGetMsg(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端，这里假设不需要自动注入微信
	client.Run(false)
	// 关闭客户端
	defer client.Close()
	// 等待客户端连接
	time.Sleep(5 * time.Second)

	// 测试 SendText
	testReceiver := "filehelper" // 微信文件助手
	testContent := "你好，这是一条测试消息"
	err := client.SendText(testReceiver, testContent)
	if err != nil {
		t.Fatalf("发送消息失败: %v", err)
	}

	// 测试 GetMsg
	msg, b := <-client.GetMsgChan()
	if !b {
		t.Error("chan closed!!")
	}

	// 打印接收到的消息
	t.Log(msg)

}

func TestClient_SendGroupTextAndAt(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	client.Run(false)
	defer client.Close()

	// 测试 SendText At
	testReceiver := "45959390469@chatroom" // 测试12
	testContent := "1222@23"               // 初始内容，不包含 @<Name>
	testAt := "wxid_jj4mhsji9tjk22"        // 替换为你要@的群成员的wxid

	err := client.SendText(testReceiver, testContent, testAt)
	if err != nil {
		t.Fatalf("发送群消息失败: %v", err)
	}

	// 测试 SendText notify@all
	testContent = "通知@所有人"
	err = client.SendText(testReceiver, testContent, "notify@all")
	if err != nil {
		t.Fatalf("发送群消息失败: %v", err)
	}
	testContent = "一般无at默认消息114514"
	err = client.SendText(testReceiver, testContent, "wxid_jj4mhsji9tjk22")
	if err != nil {
		t.Fatalf("发送群消息失败: %v", err)
	}
}

func TestClient_GetContacts(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)
	defer client.Close()

	// 启动客户端
	client.Run(false)

	// 测试 GetContacts
	contacts := client.wxClient.GetContacts()
	if len(contacts) == 0 {
		t.Fatalf("获取联系人列表失败: 列表空")
	}

	// 打印联系人列表
	for _, contact := range contacts {
		t.Logf("Wxid: %s, Code: %s, Remark: %s, Name: %s", contact.Wxid, contact.Code, contact.Remark, contact.Name)
	}

}

func TestClient_GetRoomMembers(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	client.Run(true)
	defer client.Close()

	roomId := "45959390469@chatroom"
	roomMembers, err := client.RoomMembers(roomId)
	if err != nil {
		t.Fatalf("GetRoomMemberID failed: %v", err)
	}

	// 打印解码后的字符串
	t.Logf("Decoded string for roomId %s: %v", roomId, roomMembers)
}

func TestClient_QueryRoomTable(t *testing.T) {
	c := NewClient(10, false, false)
	defer c.Close()
	c.Run(true)
	roomId := "45959390469@chatroom"
	contacts := c.wxClient.ExecDBQuery("MicroMsg.db", "SELECT * FROM ChatRoom WHERE ChatRoomName = '"+roomId+"';")
	t.Log(contacts)
}

func TestClient_ChatRoomOwner(t *testing.T) {
	c := NewClient(10, false, false)
	defer c.Close()
	c.Run(true)
	roomId := "45959390469@chatroom"
	owner := c.ChatRoomOwner(roomId)
	t.Logf("%#v", owner)
}

func TestClient_GetSelfInfo(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	client.Run(true)
	defer client.Close()

	info, ok := client.GetSelfInfo()
	if !ok {
		t.Errorf("GetSelfInfo failed: %v", info)
	}
	t.Logf("%#v", info)
}

func TestClient_GetSelfName(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	client.Run(false)
	defer client.Close()

	name, ok := client.GetSelfName()
	if !ok {
		t.Errorf("GetSelfName failed: %v", name)
	}
	t.Logf("Self Name: %s", name)
}

func TestClient_GetSelfWxId(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	client.Run(false)
	defer client.Close()

	wxid, ok := client.GetSelfWxId()
	if !ok {
		t.Errorf("GetSelfWxId failed: %v", wxid)
	}
	t.Logf("Self WxId: %s", wxid)
}

//func TestClient_GetFriend(t *testing.T) {
//	// 创建客户端实例
//	client := NewClient(10, false, false)
//
//	// 启动客户端
//	client.Run(false)
//	defer client.Close()
//
//	// 假设 "filehelper" 是一个已知的好友
//	friend, err := client.GetFriend("wxid_pagpb98c6nj722")
//	if err != nil {
//		t.Fatalf("getFriend failed: %v", err)
//	}
//
//	t.Logf("Friend Info: %#v", friend)
//}

//func TestClient_GetAllFriend(t *testing.T) {
//	// 创建客户端实例
//	client := NewClient(10, false, false)
//
//	// 启动客户端
//	client.Run(true)
//	defer client.Close()
//
//	friends, err := client.GetAllFriend()
//	if err != nil {
//		t.Fatalf("getAllFriend failed: %v", err)
//	}
//
//	for _, friend := range *friends {
//		t.Logf("Friend Info: %#v", friend)
//	}
//}

//func TestClient_GetChatRoom(t *testing.T) {
//	// 创建客户端实例
//	client := NewClient(10, false, false)
//
//	// 启动客户端
//	client.Run(false)
//	defer client.Close()
//
//	// 假设 "45959390469@chatroom" 是一个已知的群聊
//	chatroom := client.GetMember("45959390469@chatroom", true)
//	if chatroom == nil {
//		t.Fatalf("GetChatRoom failed: %v", chatroom)
//	}
//
//	t.Logf("ChatRoom Info: %#v", *chatroom)
//	t.Logf("ChatRoom RoomData: %#v", *chatroom.RoomData)
//	t.Logf("ChatRoom RoomHeadImgURL: %#v", *chatroom.RoomHeadImgURL)
//}

//func TestClient_GetAllChatRoom(t *testing.T) {
//	// 创建客户端实例
//	client := NewClient(10, false, false)
//
//	// 启动客户端
//	client.Run(true)
//	defer client.Close()
//
//	chatrooms, err := client.GetAllChatRoom()
//	if err != nil {
//		t.Fatalf("getAllChatRoom failed: %v", err)
//	}
//
//	for _, chatroom := range *chatrooms {
//		t.Logf("ChatRoom Info: %#v", chatroom)
//	}
//}

func TestClient_ReplyText(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	client.Run(true)
	defer client.Close()

	msg, b := <-client.GetMsgChan()
	if !b {
		t.Error("chan closed!!")
	}
	t.Logf("收到消息: %+v\n", msg)

	// 如果是文本消息，则回复
	if msg.Content == "ping" {
		err := msg.ReplyText("pong")
		if err != nil {
			t.Error("回复消息错误", err)
		}
	}
}

func TestClient_IsSendByFriend(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	client.Run(false)
	defer client.Close()

	msg, b := <-client.GetMsgChan()
	if !b {
		t.Error("chan closed!!")
	}
	t.Logf("收到消息: %+v\n", msg)
	if nil != msg {
		isSendByFriend := msg.IsSendByFriend()
		t.Logf("isFriend: %t", isSendByFriend)
	} else {
		t.Fatalf("msg is nil")
	}
}

func TestClient_GetMemberByCache(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	client.Run(false)
	defer client.Close()

	// 假设 "wxid_xxx" 是一个已知的成员
	member := client.GetMember("45959390469@chatroom", true) // 45959390469@chatroom wxid_qyutq6wnee2f22

	if member.Wxid == "" {
		t.Errorf("GetMember failed: %v", member)
	}
	t.Logf("GetMember Info: %#v", member)
}

func TestClient_GetMemberDirectly(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	//client.Run(false)
	defer client.Close()

	// 假设 "wxid_xxx" 是一个已知的成员
	member := client.GetMember("45959390469@chatroom", false) // 45959390469@chatroom wxid_qyutq6wnee2f22

	if member.Wxid == "" {
		t.Errorf("GetMember failed: %v", member)
	}
	t.Logf("GetMember Info: %#v", member)
}

func TestClient_GetAllMember(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	client.Run(true)
	defer client.Close()

	members := client.getAllMember()
	if members == nil || len(*members) == 0 {
		t.Errorf("GetAllMember failed: %v", members)
	}

	t.Log("members: ", members)
}

// TestClient_RecvAndDecodeImageMsg 持续接收消息, 并测试图片消息数据是否携带, 解码并保存图片
func TestClient_RecvAndDecodeImageMsg(t *testing.T) {
	// 创建客户端实例
	cli := NewClient(10, false, false)

	// 启动客户端，这里假设不需要自动注入微信
	cli.Run(true)
	// 关闭客户端
	defer cli.Close()

	for msg := range cli.GetMsgChan() {
		t.Logf("收到消息，消息类型: %v", msg.Type)
		if msg.Type == MsgTypeImage {
			if msg.FileInfo == nil {
				t.Errorf("图片消息 FileInfo 为 nil，图片数据未携带")
			}

			// 新增: 检查 RelativePathAfterMsgAttach 是否为空
			if msg.FileInfo != nil && msg.FileInfo.RelativePathAfterMsgAttach == "" {
				t.Errorf("图片消息 FileInfo.RelativePathAfterMsgAttach 为空，相对路径未提取")
			} else if msg.FileInfo != nil {
				t.Logf("图片消息 RelativePathAfterMsgAttach: %v", msg.FileInfo.RelativePathAfterMsgAttach) // 打印相对路径
			}

			t.Logf("图片消息测试通过，FileInfo: %+v", msg.FileInfo)
			return // 接收到图片消息并测试通过，结束测试
		} else {
			t.Logf("非图片消息，忽略: %v", msg.Type)
		}
	}
}

func TestClient_GetSelfFileStoragePath(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	//client.Run(false)
	defer client.Close()

	fileStoragePath, ok := client.GetSelfFileStoragePath()
	if !ok {
		t.Fatalf("client.GetSelfFileStoragePath failed: %v", ok)
	}
	t.Logf("Self FileStoragePath: %s", fileStoragePath)
}

// TestClient_GetFullFilePathFromRelativePath 测试通过相对路径获取完整文件路径
func TestClient_GetFullFilePathFromRelativePath(t *testing.T) {
	// 创建客户端实例
	cli := NewClient(10, false, false)

	// 启动客户端
	cli.Run(false)
	// 关闭客户端
	defer cli.Close()

	for msg := range cli.GetMsgChan() {
		t.Logf("收到消息，消息类型: %v", msg.Type)
		if msg.Type == MsgTypeImage {
			if msg.FileInfo == nil {
				t.Errorf("图片消息 FileInfo 为 nil，图片数据未携带")
				continue // 如果 FileInfo 为 nil，继续接收下一条消息
			}
			// 测试 GetFullFilePathFromRelativePath
			fullPathFromMethod := cli.GetFullFilePathFromRelativePath(msg.FileInfo.ExtractRelativePath())

			// 使用 filepath.ToSlash 将两种路径都转换为正斜杠形式再比较
			expectedPath := filepath.ToSlash(msg.FileInfo.FilePath)
			actualPath := filepath.ToSlash(fullPathFromMethod)

			if actualPath != expectedPath {
				t.Errorf("GetFullFilePathFromRelativePath() 路径不匹配: \nExpected: %v\nGot: %v", expectedPath, actualPath)
			} else {
				t.Logf("GetFullFilePathFromRelativePath 测试通过，路径: %s", fullPathFromMethod)
			}
			return // 接收到图片消息并测试通过，结束测试
		} else {
			t.Logf("非图片消息，忽略: %v", msg.Type)
		}
	}
}

// TestClient_SendImage 测试发送图片消息 (需手动验证)
func TestClient_SendImage(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	//client.Run(false)
	defer client.Close()

	// 等待客户端启动完成
	time.Sleep(5 * time.Second)

	// 接收者 wxid, 默认为文件助手，可修改为其他好友或群
	testReceiver := "filehelper"

	// **测试发送本地图片**
	//  请修改为你的本地图片路径，不存在则跳过本地图片测试
	localImagePath := "C:\\image\\test01.png" // 修改为你的本地图片路径

	err := client.SendImage(testReceiver, localImagePath)
	if err != nil {
		t.Fatalf("发送本地图片失败: %v", err)
	}
	t.Logf("本地图片发送成功, 请在微信中查看: %s", localImagePath)

	// **测试发送网络图片**
	//  使用网络图片 URL
	networkImageUrl := "https://cdn.jsdelivr.net/gh/Xiao-yi123/WebImageFiles/146d33e6f92e89b5ae54a193ab2f7959.jpg" // 示例网络图片URL
	err = client.SendImage(testReceiver, networkImageUrl)
	if err != nil {
		t.Fatalf("发送网络图片失败: %v", err)
	}
	t.Logf("网络图片发送成功, 请在微信中查看: %s", networkImageUrl)

	// **手动测试步骤**
	t.Log("\n**请手动测试:**")
	t.Log("1. 检查微信是否收到 **本地图片** 和 **网络图片**")
	t.Log("2. 确认图片内容正确")
	t.Log("3. 如都收到且显示正常，则手动测试通过")

	time.Sleep(5 * time.Second) //  等待 5 秒以便查看微信消息
	t.Log("SendImage 测试完成, 请检查手动测试结果")
}

func TestClient_getAllMember(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	//client.Run(true)
	defer client.Close()
	var flag = false
	infos := *client.getAllMember()
	for _, info := range infos {
		if info.Wxid == "wxid_qyutq6wnee2f22" {
			flag = true
			t.Log("存在")
		}
	}
	if flag == false {
		t.Logf("不存在")
		t.Fail()
	}

}

func TestClient_updateCacheInfo(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10, false, false)

	// 启动客户端
	//client.Run(true)
	defer client.Close()

	client.updateCacheInfo(false)
}
