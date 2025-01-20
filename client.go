// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/1/13 下午8:49:00
// @Desc
package wcf_rpc_sdk

import (
	"context"
	"errors"
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/wcf-rpc-sdk/internal/manager"
	"github.com/Clov614/wcf-rpc-sdk/internal/wcf"
	"github.com/eatmoreapple/env"
	"github.com/rs/zerolog"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	ENVTcpAddr     = "TCP_ADDR"
	DefaultTcpAddr = "tcp://127.0.0.1:10086"
)

var (
	ErrNotLogin = errors.New("not login")
	ErrNull     = errors.New("null err")
)

type Client struct {
	ctx          context.Context
	stop         context.CancelFunc
	msgBuffer    *MessageBuffer
	cacheManager *manager.CacheFileManager
	wxClient     *wcf.Client
	addr         string // 接口地址
	self         *Self
	cacheUser    *CacheUserManager // 用户信息缓存
}

// Close 停止客户端
func (c *Client) Close() {
	c.stop()
	if c.cacheManager != nil {
		c.cacheManager.Close() // 清除缓存文件
	}
	if c.cacheUser != nil {
		c.cacheUser.Close() // 释放信息缓存
	}
	err := c.wxClient.Close()
	if err != nil {
		logging.ErrorWithErr(err, "停止wcf客户端发生了错误")
	}
}

func NewClient(msgChanSize int) *Client {
	addr := env.Name(ENVTcpAddr).StringOrElse(DefaultTcpAddr) // "tcp://127.0.0.1:10086"
	ctx, cancel := context.WithCancel(context.Background())
	wxclient, err := wcf.NewWCF(addr)
	if err != nil {
		logging.Fatal(fmt.Errorf("new wcf err: %w", err).Error(), 1001)
		panic(err)
	}
	return &Client{
		ctx:       ctx,
		stop:      cancel,
		msgBuffer: NewMessageBuffer(msgChanSize), // 消息缓冲区 <缓冲大小>
		wxClient:  wxclient,
		addr:      addr,
		cacheUser: NewCacheInfoManager(),
	}
}

// Run 运行tcp监听 以及 请求tcp监听信息 <是否debug> <是否自动注入微信（自动打开微信）> <是否开启sdk-debug>
func (c *Client) Run(debug bool, autoInject bool, sdkDebug bool) {
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logging.Debug("Debug mode enabled")
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	// 增加项目字段
	logging.SetField(map[string]interface{}{"sdk": "wcf-rpc-sdk"})
	var syncSignal = make(chan os.Signal, 1) // 同步信号 确保注入后处理消息
	if autoInject {                          // 自动注入
		port, err := strconv.Atoi(c.addr[strings.LastIndex(c.addr, ":")+1:])
		if err != nil {
			logging.ErrorWithErr(err, "the port is invalid, please check your address")
			logging.Fatal("canot auto inject!", 1000, map[string]interface{}{"port": port})
		}

		go func() {
			Inject(c.ctx, port, sdkDebug) // 调用sdk.dll 注入&启动微信
			syncSignal <- syscall.SIGINT
		}()

	}
	if autoInject { // todo test 待测试
		<-syncSignal
	}
	close(syncSignal) // 关闭同步
	go func() {       // 处理接收消息
		err := c.handleMsg(c.ctx)
		if err != nil {
			logging.Fatal(fmt.Errorf("handle msg err: %w", err).Error(), 1001)
		}
	}()
	go c.cyclicUpdateSelfInfo() // 启动定时更新
	go c.cyclicUpdateCacheInfo()
}

func (c *Client) IsLogin() bool {
	return c.wxClient.IsLogin()
}

// GetMsg 获取消息 !!!不推荐使用!!!
// Deprecated
func (c *Client) GetMsg() (*Message, error) {
	if !c.wxClient.IsLogin() {
		logging.Warn("客户端并未登录成功，请稍重试")
		return nil, ErrNotLogin
	}
	msg, err := c.msgBuffer.Get(c.ctx)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// GetMsgChan 返回消息的管道
func (c *Client) GetMsgChan() <-chan *Message {
	return c.msgBuffer.msgCH
}

func (c *Client) handleMsg(ctx context.Context) (err error) {
	var handler wcf.MsgHandler = func(msg *wcf.WxMsg) error { // 回调函数
		// todo 处理图片消息以及其他消息
		covertedMsg := covertMsg(c, msg)
		if covertedMsg == nil {
			return ErrNull
		}
		err = c.msgBuffer.Put(c.ctx, covertedMsg) // 缓冲消息（内存中）
		if err != nil {
			return fmt.Errorf("MessageHandler err: %w", err)
		}
		return nil
	}
	go func() {
		c.wxClient.EnableRecvTxt()           // 允许接收消息
		err = c.wxClient.OnMSG(ctx, handler) // 当消息到来时，处理消息
		if err != nil {
			logging.ErrorWithErr(err, "handlerMsg err")
		}
	}()
	return nil
}

func covertMsg(cli *Client, msg *wcf.WxMsg) *Message {
	if msg == nil {
		logging.ErrorWithErr(ErrNull, "internal msg is nil")
		return nil
	}
	if !msg.IsGroup { // 不是群组消息
		msg.Roomid = "" // 置空
	}
	return &Message{
		meta: &meta{ // meta用于让消息可以直接调用回复
			sender:   msg.Sender,
			sendText: cli.SendText,
		},
		IsSelf:    msg.IsSelf,
		IsGroup:   msg.IsGroup,
		MessageId: msg.Id,
		Type:      msg.Type,
		Ts:        msg.Ts,
		RoomId:    msg.Roomid,
		Content:   msg.Content,
		WxId:      msg.Sender,
		Sign:      msg.Sign,
		Thumb:     msg.Thumb,
		Extra:     msg.Extra,
		Xml:       msg.Xml,
	}
}

// SendText 发送普通文本 <wxid or roomid> <文本内容> <艾特的人(wxid) 所有人:(notify@all)>
func (c *Client) SendText(receiver string, content string, ats ...string) error {
	// todo test 需要手动在content里添加上 @<Name>    2025/1/17 可以将@按顺序插入文本中，ats也相应顺序 自动查询出<Name>替换入文本
	// todo 可能需要搭配 根据wxid查询到对应的Name
	// todo 增加一个wxid的全局cache
	res := c.wxClient.SendTxt(content, receiver, ats)
	if res != 0 {
		logging.Debug("wxCliend.SendTxt", map[string]interface{}{"res": res, "receiver": receiver, "content": content, "ats": ats})
		return fmt.Errorf("wxClient.SendTxt err, code: %d", res)
	}
	return nil
}

// GetRoomMember 获取群成员信息，返回解码后的字符串以及 wxid 列表
func (c *Client) GetRoomMember(roomId string) ([]string, error) {
	contacts := c.wxClient.ExecDBQuery("MicroMsg.db", "SELECT RoomData FROM ChatRoom WHERE ChatRoomName = '"+roomId+"';")
	logging.Debug("GetRoomMember", map[string]interface{}{"roomId": roomId, "contacts": contacts})

	if len(contacts) == 0 || len(contacts[0].GetFields()) == 0 {
		return nil, fmt.Errorf("no room data found for roomId: %s", roomId)
	}

	decodedString := string(contacts[0].GetFields()[0].Content)
	logging.Debug("GetRoomMember", map[string]interface{}{"roomId": roomId, "decodedString": decodedString}) // 打印解码后的字符串

	// 使用正则表达式提取 wxid
	re := regexp.MustCompile(`\n\x19\n\x13(wxid_[a-zA-Z0-9]+)\x12\x00\x18`)
	matches := re.FindAllStringSubmatch(decodedString, -1)

	var wxids []string
	for _, match := range matches {
		wxids = append(wxids, match[1])
	}

	return wxids, nil
}

// todo GetAllRoomMember

// GetSelfInfo 获取账号个人信息
func (c *Client) GetSelfInfo() *Self {
	u := c.wxClient.GetUserInfo()
	if u == nil {
		logging.ErrorWithErr(ErrNull, "get self info err")
		return c.self
	}
	self := &Self{}
	self.Wxid = u.Wxid
	self.Name = u.Name
	self.Mobile = u.Mobile
	self.Home = u.Home
	c.self = self // 更新缓存
	return self
}

// GetSelfName 获取机器人昵称
func (c *Client) GetSelfName() string {
	if c.self.Name == "" {
		c.GetSelfInfo() // 更新缓存
	}
	if c.self == nil {
		return ""
	}
	return c.self.Name
}

// GetSelfWxId 获取机器人微信ID
func (c *Client) GetSelfWxId() string {
	if c.self == nil || c.self.Wxid == "" {
		c.GetSelfInfo() // 更新缓存
	}
	if c.self == nil {
		return ""
	}
	return c.self.Wxid
}

// cyclicUpdateSelfInfo 定时更新机器人信息
func (c *Client) cyclicUpdateSelfInfo() {
	ticker := time.NewTicker(time.Minute * 2)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.GetSelfInfo() // 每2分钟更新一次
		}
	}
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)

	// 先处理接口类型，获取其内部的实际值
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Ptr | reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return false
	}
}

func (c *Client) getInfo(wxid string, isAll bool, t InfoType, retry int, f func(id string, isAll bool, t InfoType) (interface{}, error)) (interface{}, error) {
	result, err := f(wxid, isAll, t)
	if retry > 0 && isNil(result) {
		c.updateCacheInfo()
		return c.getInfo(wxid, isAll, t, retry-1, f)
	}
	return result, err
}

// GetFriend 根据wxid获取好友信息
func (c *Client) GetFriend(wxid string) (*Friend, error) {
	info, err := c.getInfo(wxid, false, friendType, 3, c.cacheUser.Get)
	if err != nil {
		logging.ErrorWithErr(err, "get friend err", map[string]interface{}{"wxid": wxid})

	}
	res, _ := info.(*Friend)
	return res, nil
}

// GetAllFriend 获取所有好友信息
func (c *Client) GetAllFriend() (*FriendList, error) {
	info, err := c.getInfo("", true, friendType, 3, c.cacheUser.Get)
	if err != nil {
		logging.ErrorWithErr(err, "get all friend err")

	}
	res, _ := info.(*FriendList)
	return res, nil
}

// GetChatRoom 根据roomId获取群组信息
func (c *Client) GetChatRoom(roomId string) (*ChatRoom, error) {
	info, err := c.getInfo(roomId, false, roomType, 3, c.cacheUser.Get)
	if err != nil {
		logging.ErrorWithErr(err, "get all friend err")

	}
	res, _ := info.(*ChatRoom)
	return res, nil
}

// GetAllChatRoom 获取所有群组信息
func (c *Client) GetAllChatRoom() (*ChatRoomList, error) {
	info, err := c.getInfo("", true, roomType, 3, c.cacheUser.Get)
	if err != nil {
		logging.ErrorWithErr(err, "get all friend err")

	}
	res, _ := info.(*ChatRoomList)
	return res, nil
}

// 定时更新用户信息
func (c *Client) cyclicUpdateCacheInfo() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.updateCacheInfo() // 每分钟更新一次
		}
	}
}

// 更新缓存用户信息
func (c *Client) updateCacheInfo() {
	contacts := c.wxClient.GetContacts()
	if len(contacts) == 0 {
		logging.ErrorWithErr(ErrNull, "get contacts err")
		return
	}
	for _, contact := range contacts {
		user := c.getUser(contact)
		if user == nil {
			continue
		}
		switch v := user.(type) {
		case Friend:
			logging.Debug("updateCacheInfo", map[string]interface{}{"user": user, "friend": v})
			c.cacheUser.updateFriend(&v)
		case ChatRoom:
			logging.Debug("updateCacheInfo", map[string]interface{}{"user": user, "chatroom": v})
			c.cacheUser.updateChatRoom(&v)
		case GH:
			logging.Debug("updateCacheInfo", map[string]interface{}{"user": user, "gh": v})
		// todo cache GH
		default:
			logging.Warn("unknown user type", map[string]interface{}{"user": user})
		}
	}
}

// getWxIdType 判断 wxid 类型
func (c *Client) getUser(ct *wcf.RpcContact) interface{} {
	user := User{
		Wxid:     ct.Wxid,
		Code:     ct.Code,
		Remark:   ct.Remark,
		Name:     ct.Name,
		Country:  ct.Country,
		Province: ct.Province,
		City:     ct.City,
		Gender:   GenderType(ct.Gender),
	}
	switch true {
	case strings.HasPrefix(ct.Wxid, "wxid_"):
		return Friend(user)
	case strings.HasSuffix(ct.Wxid, "@chatroom"):
		return ChatRoom(user)
	case strings.HasPrefix(ct.Wxid, "gh_"):
		return GH(user)
	default:
		logging.Warn("unknown contact type", map[string]interface{}{"type": ct.Wxid})
		return nil
	}
}

// todo 图片解码模块

// todo 发送图片

// todo 对应消息的回复 message.Reply(xx)
