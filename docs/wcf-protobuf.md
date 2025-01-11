```markdown
# wcferry.wcf_pb2 模块

该模块包含 WeChatFerry 使用的 Protocol Buffer 定义，由 `.proto` 文件自动生成。

**注意:**  通常情况下，你不需要直接使用这个模块，而是通过 `wcferry.client.Wcf` 类来与微信进行交互。

## 概览

`wcf_pb2` 模块定义了 WeChatFerry RPC 通信中使用的数据结构和功能枚举。

## 数据结构 (Generated Classes)

以下列出了模块中定义的主要数据结构，这些数据结构用于 RPC 请求和响应的消息体。

| 类名          | 描述                                     |
| ------------- | ---------------------------------------- |
| `AttachMsg`   | 附件消息                                 |
| `AudioMsg`    | 语音消息                                 |
| `DbField`     | 数据库字段                               |
| `DbNames`     | 数据库名称列表                           |
| `DbQuery`     | 数据库查询请求                           |
| `DbRow`       | 数据库查询结果行                         |
| `DbRows`      | 数据库查询结果集                         |
| `DbTable`     | 数据库表信息                             |
| `DbTables`    | 数据库表信息列表                         |
| `DecPath`     | 解密路径                                 |
| `Empty`       | 空消息                                   |
| `ForwardMsg`  | 转发消息                                 |
| `MemberMgmt`  | 群成员管理 (添加、删除、邀请)            |
| `OcrMsg`      | OCR 识别请求                             |
| `PatMsg`      | 拍一拍消息                               |
| `PathMsg`     | 路径消息 (用于文件、图片等)              |
| `Request`     | RPC 请求消息                             |
| `Response`    | RPC 响应消息                             |
| `RichText`    | 富文本消息                               |
| `RoomData`    | 群信息                                   |
| `RpcContact`  | 单个联系人信息                           |
| `RpcContacts` | 联系人列表                               |
| `TextMsg`     | 文本消息                                 |
| `Transfer`    | 转账消息                                 |
| `UserInfo`    | 用户信息                                 |
| `Verification`| 好友验证消息 (v3, v4)                    |
| `WxMsg`       | 微信消息 (通用结构)                       |
| `XmlMsg`      | XML 消息                                 |

**详细的数据结构定义请参考 `wcf_pb2.py` 文件中的源码或使用 `help()` 函数查看。**

例如：

```python
import wcferry.wcf_pb2 as wcf_pb2

help(wcf_pb2.WxMsg)
```

## 功能枚举 (Functions)

`Functions` 枚举定义了所有可用的 RPC 功能号。这些功能号用于在 `Request` 消息中指定要执行的操作。

以下列出部分功能号及其对应的功能:

| 功能号 (十进制) | 功能名称                 | 描述                                      |
| --------------- | ----------------------- | ----------------------------------------- |
| 0               | `FUNC_RESERVED`         | 保留                                      |
| 1               | `FUNC_IS_LOGIN`         | 是否已登录                                |
| 16              | `FUNC_GET_SELF_WXID`    | 获取登录账户的 wxid                        |
| 17              | `FUNC_GET_MSG_TYPES`    | 获取所有消息类型                          |
| 18              | `FUNC_GET_CONTACTS`     | 获取完整通讯录                            |
| 19              | `FUNC_GET_DB_NAMES`     | 获取所有数据库名称                        |
| 20              | `FUNC_GET_DB_TABLES`    | 获取指定数据库的所有表                    |
| 21              | `FUNC_GET_USER_INFO`    | 获取登录账号个人信息                      |
| 22              | `FUNC_GET_AUDIO_MSG`    | 获取语音消息并转成 MP3                    |
| 32              | `FUNC_SEND_TXT`         | 发送文本消息                              |
| 33              | `FUNC_SEND_IMG`         | 发送图片                                  |
| 34              | `FUNC_SEND_FILE`        | 发送文件                                  |
| 35              | `FUNC_SEND_XML`         | 发送 XML                                  |
| 36              | `FUNC_SEND_EMOTION`     | 发送表情                                  |
| 37              | `FUNC_SEND_RICH_TXT`    | 发送富文本消息                            |
| 38              | `FUNC_SEND_PAT_MSG`     | 发送拍一拍消息                            |
| 39              | `FUNC_FORWARD_MSG`      | 转发消息                                  |
| 48              | `FUNC_ENABLE_RECV_TXT`  | 启用接收消息                              |
| 64              | `FUNC_DISABLE_RECV_TXT` | 禁用接收消息                              |
| 80              | `FUNC_EXEC_DB_QUERY`    | 执行 SQL 查询                             |
| 81              | `FUNC_ACCEPT_FRIEND`    | 通过好友申请                              |
| 82              | `FUNC_RECV_TRANSFER`    | 接收转账                                  |
| 83              | `FUNC_REFRESH_PYQ`      | 刷新朋友圈                                |
| 84              | `FUNC_DOWNLOAD_ATTACH`  | 下载附件                                  |
| 85              | `FUNC_GET_CONTACT_INFO` | 通过 wxid 获取联系人信息                  |
| 86              | `FUNC_REVOKE_MSG`       | 撤回消息                                  |
| 87              | `FUNC_REFRESH_QRCODE`   | 刷新二维码                                |
| 96              | `FUNC_DECRYPT_IMAGE`    | 解密图片                                  |
| 97              | `FUNC_EXEC_OCR`         | 执行 OCR 识别                             |
| 112             | `FUNC_ADD_ROOM_MEMBERS` | 添加群成员                                |
| 113             | `FUNC_DEL_ROOM_MEMBERS` | 删除群成员                                |
| 114             | `FUNC_INV_ROOM_MEMBERS` | 邀请群成员                                |

**完整的枚举值列表请参考 `wcf_pb2.py` 文件中的源码。**

## 属性 (Attributes)

*   `DESCRIPTOR`:  FileDescriptor 对象，包含对该 `.proto` 文件的完整描述。

## 使用示例 (仅供参考)

```python
import wcferry.wcf_pb2 as wcf_pb2

# 创建一个 Request 消息
req = wcf_pb2.Request()
req.func = wcf_pb2.FUNC_GET_SELF_WXID

# 序列化消息
data = req.SerializeToString()

# ... 通过某种方式将 data 发送到 RPC 服务器 ...

# ... 从 RPC 服务器接收响应数据 response_data ...

# 反序列化响应消息
resp = wcf_pb2.Response()
resp.ParseFromString(response_data)

# 处理响应
if resp.status == 0:  # 成功
    print(f"My wxid: {resp.str}")
else:
    print(f"Error: {resp.msg}")
```

**再次强调：通常情况下，你不需要直接操作 `wcf_pb2` 模块，而是使用 `wcferry.client.Wcf` 类提供的更高级别的 API。**
```
