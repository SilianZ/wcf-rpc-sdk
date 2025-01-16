// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/1/13 下午8:49:00
// @Desc
package wcf_rpc_sdk

import (
	"context"
	"errors"
	"fmt"
	"github.com/Clov614/wcf-rpc-sdk/internal/manager"
	"github.com/Clov614/wcf-rpc-sdk/internal/wcf"
	"github.com/Clov614/wcf-rpc-sdk/logging"
	"github.com/eatmoreapple/env"
	"github.com/rs/zerolog"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const (
	ENVTcpAddr     = "TCP_ADDR"
	DefaultTcpAddr = "tcp://127.0.0.1:10086"
)

var (
	ErrNotLogin = errors.New("not login")
	ErrNull     = errors.New("null")
)

type Client struct {
	ctx          context.Context
	stop         context.CancelFunc
	msgBuffer    *MessageBuffer
	cacheManager *manager.CacheManager
	wxClient     *wcf.Client
	addr         string // 接口地址
}

// Close 停止客户端
func (c *Client) Close() {
	c.stop()
	if c.cacheManager != nil {
		c.cacheManager.Close() // 清除缓存文件
	}
	err := c.wxClient.Close()
	if err != nil {
		logging.ErrorWithErr(err, "停止wcf客户端发生了错误")
	}
}

// GetMsg 获取消息
func (c *Client) GetMsg() (*Message, error) {
	if !c.wxClient.IsLogin() {
		logging.Warn("客户端并未登录成功，请稍重试")
		return nil, ErrNotLogin
	}
	msgPair, err := c.msgBuffer.Get(c.ctx)
	if err != nil {
		return nil, err
	}
	return msgPair, nil
}

func (c *Client) handleMsg(ctx context.Context) (err error) {
	var handler wcf.MsgHandler = func(msg *wcf.WxMsg) error { // 回调函数
		// todo 处理图片消息以及其他消息
		err = c.msgBuffer.Put(c.ctx, covertMsg(msg)) // 缓冲消息（内存中）
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

func covertMsg(msg *wcf.WxMsg) *Message {
	return &Message{
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
}

// SendText 发送普通文本 <wxid or roomid> <文本内容> <艾特的人(wxid) 所有人:(notify@all)>
func (c *Client) SendText(receiver string, content string, ats ...string) error {
	// todo test 需要手动在content里添加上 @<Name>
	// todo 可能需要搭配 根据wxid查询到对应的Name
	// todo 增加一个wxid的全局cache
	res := c.wxClient.SendTxt(content, receiver, ats)
	if res != 0 {
		logging.Debug("wxCliend.SendTxt", map[string]interface{}{"res": res, "receiver": receiver, "content": content, "ats": ats})
		return fmt.Errorf("wxClient.SendTxt err, code: %d", res)
	}
	return nil
}

func (c *Client) GetContacts() (Contacts, error) {
	contacts := c.wxClient.GetContacts()
	if len(contacts) == 0 {
		return nil, fmt.Errorf("get contacts err: %w", ErrNull)
	}
	contactList := make(Contacts, 0, len(contacts))
	for _, contact := range contacts {
		contactList = append(contactList, &Contact{
			Wxid:     contact.Wxid,
			Code:     contact.Code,
			Remark:   contact.Remark,
			Name:     contact.Name,
			Country:  contact.Country,
			Province: contact.Province,
			City:     contact.City,
			Gender:   contact.Gender,
		})
	}
	return contactList, nil
}

// todo 图片解码模块

// todo 发送图片

// todo 对应消息的回复 message.Reply(xx)
