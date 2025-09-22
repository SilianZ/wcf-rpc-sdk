# wcf-rpc-sdk

[![Go Reference](https://pkg.go.dev/badge/github.com/Clov614/wcf-rpc-sdk.svg)](https://pkg.go.dev/github.com/Clov614/wcf-rpc-sdk)

一个简单的 Go 语言 SDK，用于与 [WCF (WeChat Ferry)](https://github.com/lich0821/WeChatFerry) RPC 服务进行交互。

## Python 版本

本仓库同时提供 Python 实现，位于 `python/wcf_rpc_sdk` 目录。该实现与 Go 版本保持相同的 RPC 接口，并提供熟悉
的消息缓冲与联系人缓存体验。

### 安装

```bash
pip install pynng pytest  # 运行示例或测试前需要的依赖
pip install -e ./python   # 可选：以开发模式安装
```

### 快速开始

```python
from wcf_rpc_sdk import Client


client = Client(msg_buffer_size=10)
client.run()

if not client.is_login():
    print("请先在电脑端登录微信")

friends = client.get_all_friend()
print("好友数量", len(friends))

client.send_text("filehelper", "你好，这是一条来自 Python SDK 的消息")

message = client.get_message(timeout=5)
print("收到消息:", message.content)

client.close()
```

> **提示**：Python 版本需要本地已经启动的 WCF RPC 服务。若系统中未安装 `pynng`，SDK 会在运行时给出明确的依赖提示。

## 特别鸣谢

- 感谢[lich0821](https://github.com/lich0821)大佬的[WeChatFerry](https://github.com/lich0821/WeChatFerry)项目


## 安装

```bash
go get github.com/Clov614/wcf-rpc-sdk
```

## 快速开始

以下是一个简单的示例，演示如何使用此 SDK 连接微信，接收消息，发送消息以及获取联系人列表：

```go
package main

import (
	"fmt"
	wcf "github.com/Clov614/wcf-rpc-sdk"
	"time"
)

func main() {
	// 创建客户端实例，设置消息缓冲区大小为 10
	cli := wcf.NewClient(10)

	// 启动客户端，不开启调试模式，不自动注入微信，不开启 SDK 调试
	cli.Run(false, false, false)

	// 延时 5 秒等待连接
	time.Sleep(5 * time.Second)

	// 确保连接成功
	if !cli.IsLogin() {
		fmt.Println("微信未登录，请扫码登录")
		// 这里可以添加一个循环，等待用户扫码登录
		for !cli.IsLogin() {
			time.Sleep(1 * time.Second)
		}
	}

	// 获取当前账号的个人信息
	selfInfo := cli.GetSelfInfo()
	fmt.Printf("当前账号信息: %+v\n", selfInfo)

	// 获取好友列表
	friends, err := cli.GetAllFriend()
	if err != nil {
		fmt.Println("获取好友列表失败:", err.Error())
	} else {
		fmt.Println("好友列表:")
		for _, friend := range *friends {
			fmt.Printf("  Wxid: %s, Name: %s, Remark: %s\n", friend.Wxid, friend.Name, friend.Remark)
		}
	}

	// 获取群组列表
	chatRooms, err := cli.GetAllChatRoom()
	if err != nil {
		fmt.Println("获取群组列表失败:", err.Error())
	} else {
		fmt.Println("群组列表:")
		for _, room := range *chatRooms {
			fmt.Printf("  RoomId: %s, Name: %s\n", room.Wxid, room.Name)
		}
	}

	// 发送消息给文件助手
	err = cli.SendText("filehelper", "你好，这是一条测试消息")
	if err != nil {
		fmt.Println("发送消息失败:", err.Error())
	}

	// 发送群消息并 @ 指定成员
	err = cli.SendText("your_group_id@chatroom", "这是一条群消息 @user_name", "wxid_xxxxxxx") // 替换为你的群ID和要@的成员的wxid
	if err != nil {
		fmt.Println("发送群消息失败:", err.Error())
	}

	// 循环接收消息
	for {
		msg, err := cli.GetMsg()
		if err != nil {
			fmt.Println("接收消息失败:", err.Error())
			continue
		}
		fmt.Printf("收到消息: %+v\n", msg)
	}
}
```

**说明:**

1. **`cli := wcf.NewClient(10)`**: 创建一个 `Client` 实例，并设置消息缓冲区大小为 10。这意味着客户端可以缓存最多 10 条未处理的消息。
2. **`cli.Run(false, false, false)`**: 启动客户端。
    *   第一个 `false` 表示不开启调试模式。
    *   第二个 `false` 表示不自动注入微信（需要手动打开微信并扫码登录）。
    *   第三个 `false` 表示不开启 SDK 调试。
3. **`time.Sleep(5 * time.Second)`**: 等待 5 秒，让客户端有足够的时间连接到微信。
4. **`cli.IsLogin()`**: 检查微信是否已经登录。如果未登录，示例代码会打印提示信息并循环等待登录。
5. **`cli.GetSelfInfo()`**: 获取当前登录微信账号的个人信息。
6. **`cli.GetAllFriend()`**: 获取当前登录微信账号的好友列表。
7. **`cli.GetAllChatRoom()`**: 获取当前登录微信账号的群组列表。
8. **`cli.SendText("filehelper", "你好，这是一条测试消息")`**: 向微信的文件助手 (filehelper) 发送一条文本消息。
9. **`cli.SendText("your_group_id@chatroom", "这是一条群消息 your_name", "wxid_xxxxxx")`**: 向指定的群聊 (your\_group\_id@chatroom) 发送一条文本消息，并 @ 群成员 (wxid\_xxxxxx)。**注意：你需要将 `your_group_id@chatroom` 和 `wxid_xxxxxx` 替换为实际的群 ID 和成员 wxid。同时，你需要在消息内容中明确写出 `@成员昵称`，例如 `@<YourName>`。**
10. **`cli.GetMsg()`**: 循环调用 `GetMsg()` 方法来接收消息。当接收到新消息时，会打印消息内容。

**改进:**

*   添加了 `cli.IsLogin()` 判断，确保在微信登录后才执行后续操作。
*   添加了循环接收消息的示例。
*   对每个步骤添加了更详细的注释说明。
*   强调了发送群消息并 @ 成员时需要替换的参数和注意事项。
*   **新增了获取当前账号的个人信息、好友列表以及群组列表的示例和说明。**
