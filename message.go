// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/1/13 下午8:48:00
// @Desc
package wcf_rpc_sdk

import (
	"context"
	"errors"
	"fmt"
	"github.com/Clov614/wcf-rpc-sdk/internal/manager"
	"github.com/Clov614/wcf-rpc-sdk/logging"
)

var (
	ErrBufferFull = errors.New("the message buffer is full")
)

type Message struct {
	IsSelf    bool              `json:"is_self,omitempty"`
	IsGroup   bool              `json:"is_group,omitempty"`
	MessageId uint64            `json:"message_id,omitempty"`
	Type      uint32            `json:"type,omitempty"`
	Ts        uint32            `json:"ts,omitempty"`
	RoomId    string            `json:"room_id,omitempty"`
	Content   string            `json:"content,omitempty"`
	WxId      string            `json:"wx_id,omitempty"`
	Sign      string            `json:"sign,omitempty"`
	Thumb     string            `json:"thumb,omitempty"`
	Extra     string            `json:"extra,omitempty"`
	Xml       string            `json:"xml,omitempty"`
	FileInfo  *manager.FileInfo `json:"-"` // 图片保存信息

	UserInfo *UserInfo `json:"user_info,omitempty"`
	Contacts *Contacts `json:"contact,omitempty"`
}

type MessageBuffer struct {
	msgCH chan *Message // 原始消息输入通道
}

// NewMessageBuffer 创建消息缓冲区 <缓冲大小>
func NewMessageBuffer(bufferSize int) *MessageBuffer {
	mb := &MessageBuffer{
		msgCH: make(chan *Message, bufferSize),
	}
	return mb
}

// Put 向缓冲区中添加消息
func (mb *MessageBuffer) Put(ctx context.Context, msg *Message) error {
	retries := 3
	for i := 0; i < retries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case mb.msgCH <- msg:
			logging.Info("put message to buffer")
			return nil
		default:
			logging.Warn("message buffer is full, retrying", map[string]interface{}{fmt.Sprintf("%d", i+1): retries})
		}

		//// Optional: add a small delay before retrying to prevent busy-waiting
		//time.Sleep(time.Millisecond * 100)
	}
	return ErrBufferFull
}

// Get 获取消息（阻塞等待）
func (mb *MessageBuffer) Get(ctx context.Context) (*Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-mb.msgCH:
		logging.Info("retrieved message pair from buffer")
		return msg, nil
	}
}

// UserInfo 用户信息（当前用户信息）
type UserInfo struct {
	Wxid   string `json:"wxid,omitempty"`
	Name   string `json:"name,omitempty"`
	Mobile string `json:"mobile,omitempty"`
	Home   string `json:"home,omitempty"`
}

// Contacts 联系人
type Contacts []*Contact

type Contact struct {
	Wxid     string `json:"wxid,omitempty"`
	Code     string `json:"code,omitempty"`
	Remark   string `json:"remark,omitempty"`
	Name     string `json:"name,omitempty"`
	Country  string `json:"country,omitempty"`
	Province string `json:"province,omitempty"`
	City     string `json:"city,omitempty"`
	Gender   int32  `json:"gender,omitempty"`
}

type MsgType int

const (
	MsgTypeMoments           MsgType = iota // 朋友圈消息
	MsgTypeText                             // 文字
	MsgTypeImage                            // 图片
	MsgTypeVoice                            // 语音
	MsgTypeFriendConfirm                    // 好友确认
	MsgTypePossibleFriend                   // POSSIBLEFRIEND_MSG
	MsgTypeBusinessCard                     // 名片
	MsgTypeVideo                            // 视频
	MsgTypeRockPaperScissors                // 石头剪刀布 | 表情图片
	MsgTypeLocation                         // 位置
	MsgTypeShare                            // 共享实时位置、文件、转账、链接
	MsgTypeVoip                             // VOIPMSG
	MsgTypeWechatInit                       // 微信初始化
	MsgTypeVoipNotify                       // VOIPNOTIFY
	MsgTypeVoipInvite                       // VOIPINVITE
	MsgTypeShortVideo                       // 小视频
	MsgTypeRedPacket                        // 微信红包
	MsgTypeSysNotice                        // SYSNOTICE
	MsgTypeSystem                           // 红包、系统消息
	MsgTypeRevoke                           // 撤回消息
	MsgTypeSogouEmoji                       // 搜狗表情
	MsgTypeLink                             // 链接
	MsgTypeWechatRedPacket                  // 微信红包
	MsgTypeRedPacketCover                   // 红包封面
	MsgTypeVideoChannelVideo                // 视频号视频
	MsgTypeVideoChannelCard                 // 视频号名片
	MsgTypeQuote                            // 引用消息
	MsgTypePat                              // 拍一拍
	MsgTypeVideoChannelLive                 // 视频号直播
	MsgTypeProductLink                      // 商品链接
	MsgTypeMusicLink                        // 音乐链接
	MsgTypeFile                             // 文件
)

var MsgTypeNames = map[MsgType]string{
	MsgTypeMoments:           "朋友圈消息",
	MsgTypeText:              "文字",
	MsgTypeImage:             "图片",
	MsgTypeVoice:             "语音",
	MsgTypeFriendConfirm:     "好友确认",
	MsgTypePossibleFriend:    "POSSIBLEFRIEND_MSG",
	MsgTypeBusinessCard:      "名片",
	MsgTypeVideo:             "视频",
	MsgTypeRockPaperScissors: "石头剪刀布 | 表情图片",
	MsgTypeLocation:          "位置",
	MsgTypeShare:             "共享实时位置、文件、转账、链接",
	MsgTypeVoip:              "VOIPMSG",
	MsgTypeWechatInit:        "微信初始化",
	MsgTypeVoipNotify:        "VOIPNOTIFY",
	MsgTypeVoipInvite:        "VOIPINVITE",
	MsgTypeShortVideo:        "小视频",
	MsgTypeRedPacket:         "微信红包",
	MsgTypeSysNotice:         "SYSNOTICE",
	MsgTypeSystem:            "红包、系统消息",
	MsgTypeRevoke:            "撤回消息",
	MsgTypeSogouEmoji:        "搜狗表情",
	MsgTypeLink:              "链接",
	MsgTypeWechatRedPacket:   "微信红包",
	MsgTypeRedPacketCover:    "红包封面",
	MsgTypeVideoChannelVideo: "视频号视频",
	MsgTypeVideoChannelCard:  "视频号名片",
	MsgTypeQuote:             "引用消息",
	MsgTypePat:               "拍一拍",
	MsgTypeVideoChannelLive:  "视频号直播",
	MsgTypeProductLink:       "商品链接",
	MsgTypeMusicLink:         "音乐链接",
	MsgTypeFile:              "文件",
}
