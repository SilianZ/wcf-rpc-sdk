# WeChatFerry RPC 接口文档

**WeChatFerry (Wcf)** 是一个用于操作微信的工具，提供了一系列 RPC 接口来执行各种微信操作。

## Wcf 类

**描述:** WeChatFerry 的核心类，提供各种微信操作的接口。

**参数:**

*   `host` (str): RPC 服务器地址，默认为 None，表示本地启动。
*   `port` (int): RPC 服务器端口，默认为 10086，接收消息会占用 `port+1` 端口。
*   `debug` (bool): 是否开启调试模式 (仅本地启动有效)，默认为 True。
*   `block` (bool): 是否阻塞等待微信登录，默认为 True。

**属性:**

*   `contacts` (list): 联系人缓存，调用 `get_contacts` 后更新。

**方法:**

### `accept_new_friend(v3: str, v4: str, scene: int = 30) -> int`

**描述:** 通过好友申请。

**参数:**

```json
{
  "v3": "str", // 加密用户名 (好友申请消息里 v3 开头的字符串)
  "v4": "str", // Ticket (好友申请消息里 v4 开头的字符串)
  "scene": "int, 默认30" // 申请方式 (好友申请消息里的 scene); 为了兼容旧接口，默认为扫码添加 (30)
}
```

**返回值:**

```json
{
  "status": "int" // 1 为成功，其他失败
}
```

### `add_chatroom_members(roomid: str, wxids: str) -> int`

**描述:** 添加群成员。

**参数:**

```json
{
  "roomid": "str", // 待加群的 id
  "wxids": "str" // 要加到群里的 wxid，多个用逗号分隔
}
```

**返回值:**

```json
{
  "status": "int" // 1 为成功，其他失败
}
```

### `cleanup() -> None`

**描述:** 关闭连接，回收资源。

### `decrypt_image(src: str, dir: str) -> str`

**描述:** 解密图片。这方法别直接调用，下载图片使用 `download_image`。

**参数:**

```json
{
  "src": "str", // 加密的图片路径
  "dir": "str" // 保存图片的目录
}
```

**返回值:**

```json
{
  "path": "str" // 解密图片的保存路径
}
```

### `del_chatroom_members(roomid: str, wxids: str) -> int`

**描述:** 删除群成员。

**参数:**

```json
{
  "roomid": "str", // 群的 id
  "wxids": "str" // 要删除成员的 wxid，多个用逗号分隔
}
```

**返回值:**

```json
{
  "status": "int" // 1 为成功，其他失败
}
```

### `disable_recv_msg() -> int`

**描述:** 停止接收消息。

**返回值:**

```json
{
  "status": "int" // 返回状态
}
```

### `download_attach(id: int, thumb: str, extra: str) -> int`

**描述:** 下载附件（图片、视频、文件）。这方法别直接调用，下载图片使用 `download_image`。

**参数:**

```json
{
  "id": "int", // 消息中 id
  "thumb": "str", // 消息中的 thumb
  "extra": "str" // 消息中的 extra
}
```

**返回值:**

```json
{
  "status": "int" // 0 为成功, 其他失败
}
```

### `download_image(id: int, extra: str, dir: str, timeout: int = 30) -> str`

**描述:** 下载图片。

**参数:**

```json
{
  "id": "int", // 消息中 id
  "extra": "str", // 消息中的 extra
  "dir": "str", // 存放图片的目录（目录不存在会出错）
  "timeout": "int, 默认30" // 超时时间（秒）
}
```

**返回值:**

```json
{
  "path": "str" // 成功返回存储路径；空字符串为失败
}
```

### `enable_receiving_msg(pyq=False) -> bool`

**描述:** 允许接收消息，成功后通过 `get_msg` 读取消息

### `enable_recv_msg(callback: Callable[[wcferry.wxmsg.WxMsg], None] = None) -> bool`

**描述:** （不建议使用）设置接收消息回调，消息量大时可能会丢失消息。自 3.7.0.30.13 版本弃用。

### `forward_msg(id: int, receiver: str) -> int`

**描述:** 转发消息。可以转发文本、图片、表情、甚至各种 XML； 语音也行，不过效果嘛，自己验证吧。

**参数:**

```json
{
  "id": "int", // 待转发消息的 id
  "receiver": "str" // 消息接收者，wxid 或者 roomid
}
```

**返回值:**

```json
{
  "status": "int" // 1 为成功，其他失败
}
```

### `get_alias_in_chatroom(wxid: str, roomid: str) -> str`

**描述:** 获取群名片。

**参数:**

```json
{
  "wxid": "str", // wxid
  "roomid": "str" // 群的 id
}
```

**返回值:**

```json
{
  "alias": "str" // 群名片
}
```

### `get_audio_msg(id: int, dir: str, timeout: int = 3) -> str`

**描述:** 获取语音消息并转成 MP3。

**参数:**

```json
{
  "id": "int", // 语音消息 id
  "dir": "str", // MP3 保存目录（目录不存在会出错）
  "timeout": "int, 默认3" // 超时时间（秒）
}
```

**返回值:**

```json
{
  "path": "str" // 成功返回存储路径；空字符串为失败
}
```

### `get_chatroom_members(roomid: str) -> Dict`

**描述:** 获取群成员。

**参数:**

```json
{
  "roomid": "str" // 群的 id
}
```

**返回值:**

```json
{
  "members": {
    "wxid1": "昵称1",
    "wxid2": "昵称2",
    "...": "..."
  }
}
```

### `get_contacts() -> List[Dict]`

**描述:** 获取完整通讯录。

**返回值:**

```json
[
  {
    // 联系人信息
  },
  // ...
]
```

### `get_dbs() -> List[str]`

**描述:** 获取所有数据库。

**返回值:**

```json
[
  "db1",
  "db2",
  "..."
]
```

### `get_friends() -> List[Dict]`

**描述:** 获取好友列表。

**返回值:**

```json
[
  {
    // 好友信息
  },
  // ...
]
```

### `get_info_by_wxid(wxid: str) -> dict`

**描述:** 通过 wxid 查询微信号昵称等信息。

**参数:**

```json
{
  "wxid": "str" // 联系人 wxid
}
```

**返回值:**

```json
{
  "wxid": "str",
  "code": "str",
  "name": "str",
  "gender": "int"
}
```

### `get_msg(block=True) -> wcferry.wxmsg.WxMsg`

**描述:** 从消息队列中获取消息。

**参数:**

```json
{
  "block": "bool, 默认True" // 是否阻塞
}
```

**返回值:**

```json
{
  // WxMsg 对象
}
```

**抛出:**

*   `Empty`: 如果阻塞并且超时，抛出空异常，需要用户自行捕获。

### `get_msg_types() -> Dict`

**描述:** 获取所有消息类型。

**返回值:**

```json
{
  "type1": "描述1",
  "type2": "描述2",
  "...": "..."
}
```

### `get_ocr_result(extra: str, timeout: int = 2) -> str`

**描述:** 获取 OCR 结果。鸡肋，需要图片能自动下载；通过下载接口下载的图片无法识别。

**参数:**

```json
{
  "extra": "str", // 待识别的图片路径，消息里的 extra
  "timeout": "int, 默认2" // 超时时间
}
```

**返回值:**

```json
{
  "result": "str" // OCR 结果
}
```

### `get_qrcode() -> str`

**描述:** 获取登录二维码，已经登录则返回空字符串。

**返回值:**

```json
{
  "qrcode": "str" // 二维码 base64 字符串
}
```

### `get_self_wxid() -> str`

**描述:** 获取登录账户的 wxid。

**返回值:**

```json
{
  "wxid": "str" // 登录账户的 wxid
}
```

### `get_tables(db: str) -> List[Dict]`

**描述:** 获取 db 中所有表。

**参数:**

```json
{
  "db": "str" // 数据库名（可通过 get_dbs 查询）
}
```

**返回值:**

```json
[
  {
    "name": "table1",
    "sql": "CREATE TABLE ..."
  },
  {
    "name": "table2",
    "sql": "CREATE TABLE ..."
  },
  // ...
]
```

### `get_user_info() -> Dict`

**描述:** 获取登录账号个人信息。

**返回值:**

```json
{
  // 个人信息
}
```

### `invite_chatroom_members(roomid: str, wxids: str) -> int`

**描述:** 邀请群成员。

**参数:**

```json
{
  "roomid": "str", // 群的 id
  "wxids": "str" // 要邀请成员的 wxid, 多个用逗号`,`分隔
}
```

**返回值:**

```json
{
  "status": "int" // 1 为成功，其他失败
}
```

### `is_login() -> bool`

**描述:** 是否已经登录。

**返回值:**

```json
{
  "login": "bool" // True 已登录，False 未登录
}
```

### `is_receiving_msg() -> bool`

**描述:** 是否已启动接收消息功能。

**返回值:**

```json
{
  "receiving": "bool" // True 正在接收消息，False 未接收消息
}
```

### `keep_running()`

**描述:** 阻塞进程，让 RPC 一直维持连接。

### `query_sql(db: str, sql: str) -> List[Dict]`

**描述:** 执行 SQL，如果数据量大注意分页，以免 OOM。

**参数:**

```json
{
  "db": "str", // 要查询的数据库
  "sql": "str" // 要执行的 SQL
}
```

**返回值:**

```json
[
  {
    "column1": "value1",
    "column2": "value2",
    "...": "..."
  },
  // ...
]
```

### `receive_transfer(wxid: str, transferid: str, transactionid: str) -> int`

**描述:** 接收转账。

**参数:**

```json
{
  "wxid": "str", // 转账消息里的发送人 wxid
  "transferid": "str", // 转账消息里的 transferid
  "transactionid": "str" // 转账消息里的 transactionid
}
```

**返回值:**

```json
{
  "status": "int" // 1 为成功，其他失败
}
```

### `refresh_pyq(id: int = 0) -> int`

**描述:** 刷新朋友圈。

**参数:**

```json
{
  "id": "int, 默认0" // 开始 id，0 为最新页
}
```

**返回值:**

```json
{
  "status": "int" // 1 为成功，其他失败
}
```

### `revoke_msg(id: int = 0) -> int`

**描述:** 撤回消息。

**参数:**

```json
{
  "id": "int" // 待撤回消息的 id
}
```

**返回值:**

```json
{
  "status": "int" // 1 为成功，其他失败
}
```

### `send_emotion(path: str, receiver: str) -> int`

**描述:** 发送表情。

**参数:**

```json
{
  "path": "str", // 本地表情路径，如：C:/Projs/WeChatRobot/emo.gif
  "receiver": "str" // 消息接收人，wxid 或者 roomid
}
```

**返回值:**

```json
{
  "status": "int" // 0 为成功，其他失败
}
```

### `send_file(path: str, receiver: str) -> int`

**描述:** 发送文件，非线程安全。

**参数:**

```json
{
  "path": "str", // 本地文件路径，如：C:/Projs/WeChatRobot/README.MD 或 https://raw.githubusercontent.com/lich0821/WeChatFerry/master/README.MD
  "receiver": "str" // 消息接收人，wxid 或者 roomid
}
```

**返回值:**

```json
{
  "status": "int" // 0 为成功，其他失败
}
```

### `send_image(path: str, receiver: str) -> int`

**描述:** 发送图片，非线程安全。

**参数:**

```json
{
  "path": "str", // 图片路径，如：C:/Projs/WeChatRobot/TEQuant.jpeg 或 https://raw.githubusercontent.com/lich0821/WeChatFerry/master/assets/TEQuant.jpg
  "receiver": "str" // 消息接收人，wxid 或者 roomid
}
```

**返回值:**

```json
{
  "status": "int" // 0 为成功，其他失败
}
```

### `send_pat_msg(roomid: str, wxid: str) -> int`

**描述:** 拍一拍群友。

**参数:**

```json
{
  "roomid": "str", // 群 id
  "wxid": "str" // 要拍的群友的 wxid
}
```

**返回值:**

```json
{
  "status": "int" // 1 为成功，其他失败
}
```

### `send_rich_text(name: str, account: str, title: str, digest: str, url: str, thumburl: str, receiver: str) -> int`

**描述:** 发送富文本消息。

**参数:**

```json
{
  "name": "str", // 左下显示的名字
  "account": "str", // 填公众号 id 可以显示对应的头像（gh_ 开头的）
  "title": "str", // 标题，最多两行
  "digest": "str", // 摘要，三行
  "url": "str", // 点击后跳转的链接
  "thumburl": "str", // 缩略图的链接
  "receiver": "str" // 接收人, wxid 或者 roomid
}
```

**返回值:**

```json
{
  "status": "int" // 0 为成功，其他失败
}
```

### `send_text(msg: str, receiver: str, aters: str | None = '') -> int`

**描述:** 发送文本消息。

**参数:**

```json
{
  "msg": "str", // 要发送的消息，换行使用 \n （单杠）；如果 @ 人的话，需要带上跟 aters 里数量相同的 @
  "receiver": "str", // 消息接收人，wxid 或者 roomid
  "aters": "str, 默认''" // 要 @ 的 wxid，多个用逗号分隔；@所有人 只需要 notify@all
}
```

**返回值:**

```json
{
  "status": "int" // 0 为成功，其他失败
}
```

### `send_xml(receiver: str, xml: str, type: int, path: str = None) -> int`

**描述:** 发送 XML。

**参数:**

```json
{
  "receiver": "str", // 消息接收人，wxid 或者 roomid
  "xml": "str", // xml 内容
  "type": "int", // xml 类型，如：0x21 为小程序
  "path": "str, 默认None" // 封面图片路径
}
```

**返回值:**

```json
{
  "status": "int" // 0 为成功，其他失败
}
```

## WxMsg 类

**描述:** 微信消息结构。

**属性:**

*   `type` (int): 消息类型，可通过 `get_msg_types` 获取。
*   `id` (str): 消息 id。
*   `xml` (str): 消息 xml 部分。
*   `sender` (str): 消息发送人。
*   `roomid` (str): （仅群消息有）群 id。
*   `content` (str): 消息内容。
*   `thumb` (str): 视频或图片消息的缩略图路径。
*   `extra` (str): 视频或图片消息的路径。
*   `ts` (int): 消息时间戳
*   `sign` (str): 消息签名

**方法:**

*   `from_group() -> bool`: 是否群聊消息。
*   `from_self() -> bool`: 是否自己发的消息。
*   `is_at(wxid) -> bool`: 是否被 @：群消息，在 @ 名单里，并且不是 @ 所有人。
*   `is_text() -> bool`: 是否文本消息。

