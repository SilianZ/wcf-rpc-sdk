package wcf_rpc_sdk

import (
	"testing"
	"time"
)

// TestClient_Recv 持续接收消息
func TestClient_Recv(t *testing.T) {
	// 创建客户端实例
	cli := NewClient(10)

	// 启动客户端，这里假设不需要自动注入微信
	cli.Run(false, false, false)
	// 关闭客户端
	defer cli.Close()
	for msg := range cli.GetMsgChan() {
		t.Log(msg)
	}
}

func TestClient_SendTextAndGetMsg(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端，这里假设不需要自动注入微信
	client.Run(false, false, false)
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
	msg, err := client.GetMsg()
	if err != nil {
		t.Fatalf("接收消息失败: %v", err)
	}

	// 打印接收到的消息
	t.Log(msg)

}

func TestClient_SendGroupTextAndAt(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(false, false, false)
	defer client.Close()

	// 测试 SendText At
	testReceiver := "45959390469@chatroom" // 测试12
	testContent := "@AkiAoi-evil 123"      // todo test 需要手动在content里添加上 @<Name>
	testAt := "wxid_jj4mhsji9tjk22"        // 替换为你要@的群成员的wxid
	err := client.SendText(testReceiver, testContent, testAt)
	if err != nil {
		t.Fatalf("发送群消息失败: %v", err)
	}
}

func TestClient_GetContacts(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)
	defer client.Close()

	// 启动客户端
	client.Run(false, false, false)

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

func TestClient_GetRoomMemberID(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(true, false, false)
	defer client.Close()

	roomId := "45959390469@chatroom"
	wxids, err := client.GetRoomMemberID(roomId)
	if err != nil {
		t.Fatalf("GetRoomMemberID failed: %v", err)
	}

	// 打印解码后的字符串
	t.Logf("Decoded string for roomId %s: %v", roomId, wxids)
}

func TestClient_GetSelfInfo(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(true, false, false)
	defer client.Close()

	info := client.GetSelfInfo()
	t.Logf("%#v", info)
}

func TestClient_GetSelfName(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(false, false, false)
	defer client.Close()

	name := client.GetSelfName()
	t.Logf("Self Name: %s", name)
}

func TestClient_GetSelfWxId(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(false, false, false)
	defer client.Close()

	wxid := client.GetSelfWxId()
	t.Logf("Self WxId: %s", wxid)
}

func TestClient_GetFriend(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(false, false, false)
	defer client.Close()

	// 假设 "filehelper" 是一个已知的好友
	friend, err := client.GetFriend("wxid_pagpb98c6nj722")
	if err != nil {
		t.Fatalf("getFriend failed: %v", err)
	}

	t.Logf("Friend Info: %#v", friend)
}

func TestClient_GetAllFriend(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(true, false, false)
	defer client.Close()

	friends, err := client.GetAllFriend()
	if err != nil {
		t.Fatalf("getAllFriend failed: %v", err)
	}

	for _, friend := range *friends {
		t.Logf("Friend Info: %#v", friend)
	}
}

func TestClient_GetChatRoom(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(false, false, false)
	defer client.Close()

	// 假设 "45959390469@chatroom" 是一个已知的群聊
	chatroom, err := client.GetChatRoom("45959390469@chatroom")
	if err != nil {
		t.Fatalf("getChatRoom failed: %v", err)
	}

	t.Logf("ChatRoom Info: %#v", *chatroom)
	t.Logf("ChatRoom RoomData: %#v", *chatroom.RoomData)
	t.Logf("ChatRoom RoomHeadImgURL: %#v", *chatroom.RoomHeadImgURL)
}

func TestClient_GetAllChatRoom(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(true, false, false)
	defer client.Close()

	chatrooms, err := client.GetAllChatRoom()
	if err != nil {
		t.Fatalf("getAllChatRoom failed: %v", err)
	}

	for _, chatroom := range *chatrooms {
		t.Logf("ChatRoom Info: %#v", chatroom)
	}
}

func TestClient_ReplyText(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(true, false, false)
	defer client.Close()

	msg, err := client.GetMsg()
	if err != nil {
		t.Error("接收消息失败:", err.Error())
	}
	t.Logf("收到消息: %+v\n", msg)

	// 如果是文本消息，则回复
	if msg.Content == "ping" {
		err = msg.ReplyText("pong")
		if err != nil {
			t.Error("回复消息错误", err)
		}
	}
}

func TestClient_IsSendByFriend(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(false, false, false)
	defer client.Close()

	msg, err := client.GetMsg()
	if err != nil {
		t.Error("接收消息失败:", err.Error())
	}
	t.Logf("收到消息: %+v\n", msg)

	isSendByFriend := msg.IsSendByFriend()
	t.Logf("isFriend: %t", isSendByFriend)
}

func TestClient_GetMember(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(false, false, false)
	defer client.Close()

	// 假设 "wxid_xxx" 是一个已知的成员
	memberList, err := client.GetMember("45959390469@chatroom") // 45959390469@chatroom wxid_qyutq6wnee2f22
	if err != nil {
		t.Fatalf("getMember failed: %v", err)
	}

	for _, member := range memberList {
		t.Logf("Member Info: %#v", member)
	}
}

func TestClient_GetAllMember(t *testing.T) {
	// 创建客户端实例
	client := NewClient(10)

	// 启动客户端
	client.Run(true, false, false)
	defer client.Close()

	members, err := client.GetAllMember()
	if err != nil {
		t.Fatalf("getAllMember failed: %v", err)
	}

	t.Log("members: ", members)
}
