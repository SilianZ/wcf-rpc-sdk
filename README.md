# wcf-rpc-sdk

[![Go Reference](https://pkg.go.dev/badge/github.com/Clov614/wcf-rpc-sdk.svg)](https://pkg.go.dev/github.com/Clov614/wcf-rpc-sdk)

一个简单的 Go 语言 SDK，用于与 WCF (WeChat Ferry) RPC 服务进行交互。

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

	// 获取联系人列表
	contacts, err := cli.GetContacts()
	if err != nil {
		fmt.Println("获取联系人列表失败:", err.Error())
	} else {
		fmt.Println("联系人列表:")
		for _, contact := range contacts {
			fmt.Printf("  Wxid: %s, Name: %s, Remark: %s\n", contact.Wxid, contact.Name, contact.Remark)
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
5. **`cli.GetContacts()`**: 获取当前登录微信账号的联系人列表。
6. **`cli.SendText("filehelper", "你好，这是一条测试消息")`**: 向微信的文件助手 (filehelper) 发送一条文本消息。
7. **`cli.SendText("your_group_id@chatroom", "这是一条群消息 your_name", "wxid_xxxxxx")`**: 向指定的群聊 (your\_group\_id@chatroom) 发送一条文本消息，并 @ 群成员 (wxid\_jj4mhsji9tjk22)。**注意：你需要将 `your_group_id@chatroom` 和 `wxid_jj4mhsji9tjk22` 替换为实际的群 ID 和成员 wxid。同时，你需要在消息内容中明确写出 `@成员昵称`，例如 `@AkiAoi-evil`。**
8. **`cli.GetMsg()`**: 循环调用 `GetMsg()` 方法来接收消息。当接收到新消息时，会打印消息内容。

**改进:**

*   添加了 `cli.IsLogin()` 判断，确保在微信登录后才执行后续操作。
*   添加了循环接收消息的示例。
*   对每个步骤添加了更详细的注释说明。
*   强调了发送群消息并 @ 成员时需要替换的参数和注意事项。
