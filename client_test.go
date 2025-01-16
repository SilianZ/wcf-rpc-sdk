package wcf_rpc_sdk

import (
	"testing"
	"time"
)

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
