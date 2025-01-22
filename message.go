// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/1/13 下午8:48:00
// @Desc
package wcf_rpc_sdk

import (
	"context"
	"errors"
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/wcf-rpc-sdk/internal/manager"
)

var (
	ErrBufferFull = errors.New("the message buffer is full")
)

type IMeta interface {
	ReplyText(content string, ats ...string) error
	IsSendByFriend() bool
}

// 用于回调
type meta struct {
	rawMsg *Message
	sender string
	cli    *Client
}

// ReplyText 回复文本
func (m *meta) ReplyText(content string, ats ...string) error {
	return m.cli.SendText(m.sender, content, ats...)
}

// IsSendByFriend 是否好友发送的消息
func (m *meta) IsSendByFriend() bool {
	if m.rawMsg.IsSelf {
		return false
	}
	friend, _ := m.cli.GetFriend(m.rawMsg.WxId)
	return friend != nil
}

type Message struct {
	meta      IMeta             // 用于实现对客户端操作
	IsSelf    bool              `json:"is_self,omitempty"`
	IsGroup   bool              `json:"is_group,omitempty"`
	MessageId uint64            `json:"message_id,omitempty"`
	Type      uint32            `json:"type,omitempty"`
	Ts        uint32            `json:"ts,omitempty"`
	RoomId    string            `json:"room_id,omitempty"`
	RoomData  *RoomData         `json:"room_data,omitempty"`
	Content   string            `json:"content,omitempty"`
	WxId      string            `json:"wx_id,omitempty"`
	Sign      string            `json:"sign,omitempty"`
	Thumb     string            `json:"thumb,omitempty"`
	Extra     string            `json:"extra,omitempty"`
	Xml       string            `json:"xml,omitempty"`
	FileInfo  *manager.FileInfo `json:"-"` // 图片保存信息

	//UserInfo *UserInfo `json:"user_info,omitempty"` todo
	//Contacts *Contacts `json:"contact,omitempty"`
}

// ReplyText 回复文本
func (m *Message) ReplyText(content string, ats ...string) error {
	return m.meta.ReplyText(content, ats...)
}

// IsSendByFriend 是否为好友的消息
func (m *Message) IsSendByFriend() bool {
	return m.meta.IsSendByFriend()
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

type GenderType uint32

const (
	UnKnown GenderType = iota
	Boy
	Girl
)

// User 用户抽象通用结构
type User struct {
	Wxid     string     `json:"wxid,omitempty"`    // 微信ID (wxid_xxx gh_xxxx  xxxx@chatroom)
	Code     string     `json:"code,omitempty"`    // 微信号
	Remark   string     `json:"remark,omitempty"`  // 对其的备注
	Name     string     `json:"name,omitempty"`    // 用户名\公众号名\群名
	Country  string     `json:"country,omitempty"` // 国家代码
	Province string     `json:"province,omitempty"`
	City     string     `json:"city,omitempty"`
	Gender   GenderType `json:"gender,omitempty"` // 性别
}

type Self struct { // 机器人自己
	User
	Mobile string `json:"mobile,omitempty"` // 个人信息时携带
	Home   string `json:"home,omitempty"`   // 个人信息时携带
}

// FriendList 联系人
type FriendList []*Friend
type ChatRoomList []*ChatRoom
type GhList []*GH

type Friend User
type ChatRoom struct { // 群聊
	User
	RoomID           string    `json:"room_id"`
	RoomData         *RoomData `json:"room_data"`                   // 群聊成员
	RoomHeadImgURL   *string   `json:"room_head_img_url,omitempty"` // 群聊头像
	RoomAnnouncement *string   `json:"room_announcement,omitempty"` // 公告
}

type RoomData struct {
	Members []*ContactInfo `json:"members,omitempty"` // 成员列表
}

func (rd *RoomData) GetMembers(wxidList ...string) ([]*ContactInfo, error) {
	var contactInfoList = make([]*ContactInfo, 0, len(rd.Members))
	if wxidList == nil || len(wxidList) == 0 {
		return nil, ErrNull
	}
	for _, w := range wxidList {
		for _, member := range rd.Members {
			if member.Wxid == w {
				contactInfoList = append(contactInfoList, member)
			}
		}
	}
	if len(contactInfoList) == 0 {
		return nil, ErrNull
	}

	return contactInfoList, nil
}

func (rd *RoomData) GetMembersNickNameById(wxidList ...string) ([]string, error) {
	var nicknameList = make([]string, 0, len(wxidList))
	if len(wxidList) == 0 {
		return nil, ErrNull
	}
	for _, wxid := range wxidList {
		for _, member := range rd.Members {
			if member.Wxid == wxid {
				nicknameList = append(nicknameList, member.NickName)
				break // 找到一个匹配的 wxid 就跳出内层循环
			}
		}
	}
	if len(nicknameList) == 0 {
		return nil, ErrNull
	}
	return nicknameList, nil
}

func (rd *RoomData) GetMembersByNickName(nicknameList ...string) ([]*ContactInfo, error) {
	var contactInfoList = make([]*ContactInfo, 0, len(nicknameList))
	if len(nicknameList) == 0 {
		return nil, ErrNull
	}
	for _, nickname := range nicknameList {
		for _, member := range rd.Members {
			if member.NickName == nickname {
				contactInfoList = append(contactInfoList, member)
			}
		}
	}
	if len(contactInfoList) == 0 {
		return nil, ErrNull
	}
	return contactInfoList, nil
}

type ContactInfo struct {
	// 微信ID
	Wxid string `json:"wxid"`
	// 微信号
	Alias string `json:"alias,omitempty"`
	// 删除标记
	DelFlag uint8 `json:"del_flag"`
	// 类型
	ContactType uint8 `json:"contact_type"`
	// 备注
	Remark string `json:"remark,omitempty"`
	// 昵称
	NickName string `json:"nick_name,omitempty"`
	// 昵称拼音首字符
	PyInitial string `json:"py_initial,omitempty"`
	// 昵称全拼
	QuanPin string `json:"quan_pin,omitempty"`
	// 备注拼音首字母
	RemarkPyInitial string `json:"remark_py_initial,omitempty"`
	// 备注全拼
	RemarkQuanPin string `json:"remark_quan_pin,omitempty"`
	// 小头像
	SmallHeadURL string `json:"small_head_url,omitempty"`
	// 大头像
	BigHeadURL string `json:"big_head_url,omitempty"`
}

type GH User // todo 公众号

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
