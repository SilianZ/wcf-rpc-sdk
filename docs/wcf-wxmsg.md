# wcferry.wxmsg 模块

该模块提供 `WxMsg` 类，用于表示微信消息。

## WxMsg 类

`WxMsg` 类封装了从微信接收到的消息数据。

**构造方法:**

```python
class wcferry.wxmsg.WxMsg(msg: wcferry.wcf_pb2.WxMsg)
```

**参数:**

*   `msg` (`wcferry.wcf_pb2.WxMsg`):  从 `wcf_pb2` 模块反序列化后的原始消息对象。

**属性:**

| 属性名    | 类型   | 描述                                                         |
| --------- | ------ | ------------------------------------------------------------ |
| `type`    | `int`  | 消息类型，可通过 `Wcf.get_msg_types()` 获取所有消息类型及其描述。 |
| `id`      | `str`  | 消息 ID                                                      |
| `xml`     | `str`  | 消息 XML 部分 (如果存在)                                      |
| `sender`  | `str`  | 消息发送者的 wxid                                             |
| `roomid`  | `str`  | 群聊消息的群 id (非群聊消息为空字符串)                        |
| `content` | `str`  | 消息内容                                                     |
| `thumb`   | `str`  | 视频或图片消息的缩略图路径                                   |
| `extra`   | `str`  | 视频或图片消息的路径                                         |
| `ts`      | `int`  | 消息时间戳                                                   |
| `sign`    | `str`  | 消息签名                                                     |

**方法:**

| 方法名         | 返回值   | 描述                                                                                                 |
| -------------- | -------- | ---------------------------------------------------------------------------------------------------- |
| `from_group()` | `bool`   | 是否为群聊消息                                                                                       |
| `from_self()`  | `bool`   | 是否为自己发送的消息                                                                                 |
| `is_at(wxid)`  | `bool`   | 是否被 @：群消息，在 @ 名单里，并且不是 @ 所有人。需要传入自己的 wxid 进行判断。                      |
| `is_text()`    | `bool`   | 是否为文本消息                                                                                       |

**示例:**

```python
from wcferry import Wcf

wcf = Wcf()  # 初始化 Wcf 对象

# ... 启用接收消息 ...
wcf.enable_receiving_msg()

while True:
    msg = wcf.get_msg()  # 获取消息
    if msg:
        print(f"收到消息: 类型={msg.type}, 发送者={msg.sender}, 内容={msg.content}")

        if msg.from_group():
            print(f"来自群聊: {msg.roomid}")
            if msg.is_at(wcf.get_self_wxid()):
                print("我被 @ 了")

        if msg.is_text():
            print("这是一条文本消息")
```

**说明:**

`WxMsg` 对象通常由 `wcferry.client.Wcf.get_msg()` 方法返回，你不需要手动创建 `WxMsg` 对象。
