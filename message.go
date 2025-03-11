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
	"github.com/Clov614/wcf-rpc-sdk/internal/utils/imgutil"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrBufferFull = errors.New("the message buffer is full")
)

type IMeta interface {
	ReplyText(content string, ats ...string) error
	ReplyImage(src string) error
	ReplyFile(src string) error
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

// ReplyImage 回复图片
func (m *meta) ReplyImage(src string) error {
	return m.cli.SendImage(m.sender, src)
}

// ReplyFile 回复文件
func (m *meta) ReplyFile(src string) error {
	return m.cli.SendFile(m.sender, src)
}

// IsSendByFriend 是否好友发送的消息
func (m *meta) IsSendByFriend() bool {
	if m.rawMsg.IsSelf {
		return false
	}
	friend, _ := m.cli.GetFriend(m.rawMsg.WxId) // todo 这边本意是区分群聊、公众号、好友 但是群聊消息中的wxId不为空，当前意思是判断是否好友的消息
	return friend != nil
}

type Message struct {
	meta      IMeta     // 用于实现对客户端操作
	IsSelf    bool      `json:"is_self,omitempty"`
	IsGroup   bool      `json:"is_group,omitempty"`
	IsGH      bool      `json:"is_gh,omitempty"` // 是否公众号
	MessageId uint64    `json:"message_id,omitempty"`
	Type      MsgType   `json:"type,omitempty"`
	Ts        uint32    `json:"ts,omitempty"`
	RoomId    string    `json:"room_id,omitempty"`
	RoomData  *RoomData `json:"room_data,omitempty"`
	Content   string    `json:"content,omitempty"`
	WxId      string    `json:"wx_id,omitempty"`
	Sign      string    `json:"sign,omitempty"`
	Thumb     string    `json:"thumb,omitempty"`
	Extra     string    `json:"extra,omitempty"`
	Xml       string    `json:"xml,omitempty"`
	FileInfo  *FileInfo `json:"file_info,omitempty"` // 图片保存信息
	Quote     *QuoteMsg `json:"quote,omitempty"`     // 引用消息

	//UserInfo *UserInfo `json:"user_info,omitempty"` todo
	//Contacts *Contacts `json:"contact,omitempty"`
}

type FileInfo struct {
	FilePath                   string `json:"file_path,omitempty"`                      // Full file path
	RelativePathAfterMsgAttach string `json:"relative_path_after_msg_attach,omitempty"` // MsgAttach 之后的相对路径
	FileName                   string `json:"file_name,omitempty"`                      // File name including extension
	FileExt                    string `json:"file_ext,omitempty"`                       // File extension
	IsImg                      bool   `json:"is_img,omitempty"`                         // Indicates if the file is an image
	Data                       []byte `json:"-"	`                                       // 图片数据
}

// DecryptImg 解析图片信息
func (fi *FileInfo) DecryptImg() (err error) {
	fi.Data, err = imgutil.DecodeDatFileToBytes(fi.FilePath)
	if err != nil {
		return fmt.Errorf("decrypt img error: %w", err)
	}
	fileType, err := imgutil.DetectFileType(fi.Data)
	if err != nil {
		logging.WarnWithErr(err, "detect file type")
	}
	fi.FileExt = string(fileType)
	return nil
}

// ExtractRelativePath 提取路径后缀
func (fi *FileInfo) ExtractRelativePath() string {
	if fi.RelativePathAfterMsgAttach != "" {
		return fi.RelativePathAfterMsgAttach
	}
	msgAttachIndex := strings.Index(fi.FilePath, "MsgAttach")
	if msgAttachIndex != -1 {
		relativePath := fi.FilePath[msgAttachIndex+len("MsgAttach"):]
		// 使用 filepath.Join 确保路径以 / 开头 (如果需要) 并进行清理
		extractedPath := filepath.Join("/", relativePath) //  "/" 作为第一个参数，确保路径是绝对路径 (相对 MsgAttach 而言)
		// 注意: filepath.Join 会清理路径，例如将 "//" 变为 "/"
		fi.RelativePathAfterMsgAttach = filepath.ToSlash(extractedPath) // 使用 filepath.ToSlash 转换为正斜杠
		return fi.RelativePathAfterMsgAttach
	}
	return ""
}

// ReplyText 回复文本
func (m *Message) ReplyText(content string, ats ...string) error {
	return m.meta.ReplyText(content, ats...)
}

// ReplyImage 回复图片
func (m *Message) ReplyImage(src string) error {
	return m.meta.ReplyImage(src)
}

// ReplyFile 回复文件
func (m *Message) ReplyFile(src string) error {
	return m.meta.ReplyFile(src)
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
			logging.Debug("put message to buffer", map[string]interface{}{"msg": msg})
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
		logging.Debug("retrieved message pair from buffer", map[string]interface{}{"msg": msg})
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
	Mobile          string `json:"mobile,omitempty"` // 个人信息时携带
	Home            string `json:"home,omitempty"`   // C:/Users/Administrator/Documents/WeChat Files/
	FileStoragePath string `json:"fileStoragePath"`  // C:/Users/Administrator/Documents/WeChat Files/wxid_p5z4fuhnbdgs22/FileStorage/
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
	Members       []*ContactInfo `json:"members,omitempty"`     // 成员列表
	AtedMSequence []*ContactInfo `json:"at_sequence,omitempty"` // 被艾特的顺序
	IsAtSelf      bool           `json:"is_at_self"`            // 是否艾特自己
}

// AnalyseMemberAt 检查并生成成员@情况
func (rd *RoomData) AnalyseMemberAt(selfWxid string, content string) {
	if selfWxid == "" {
		logging.Error("analyse member at wxid error", map[string]interface{}{"wxid": selfWxid})
		return
	}
	if rd.Members == nil || len(rd.Members) == 0 {
		return
	}
	// 获取消息中艾特成员的成员名
	re := regexp.MustCompile(`@([^\s]+?) `)
	matches := re.FindAllStringSubmatch(content, -1)
	atNameList := make([]string, len(matches))
	rd.AtedMSequence = make([]*ContactInfo, len(matches))
	for i, match := range matches {
		if len(match) > 1 {
			atNameList[i] = match[1]
			// 检查 msg.RoomData 是否为 nil
			infos, err := rd.GetMembersByNickName(match[1])
			if err != nil {
				logging.WarnWithErr(err, "RoomData.GetMembersByNickName fail")
				continue
			}
			rd.AtedMSequence[i] = infos[0]
			if selfWxid == infos[0].Wxid { // fixme: 可能的空指针错误
				rd.IsAtSelf = true
			}
		}
	}
}

func (rd *RoomData) GetMembers(wxidList ...string) ([]*ContactInfo, error) {
	var contactInfoList = make([]*ContactInfo, len(rd.Members))
	if wxidList == nil || len(wxidList) == 0 {
		return nil, ErrNull
	}
	for i, w := range wxidList {
		for _, member := range rd.Members {
			if member.Wxid == w {
				contactInfoList[i] = member
			}
		}
	}
	if len(contactInfoList) == 0 {
		return nil, ErrNull
	}

	return contactInfoList, nil
}

func (rd *RoomData) GetMembersNickNameById(wxidList ...string) ([]string, error) {
	var nicknameList = make([]string, len(wxidList))
	if len(wxidList) == 0 {
		return nil, ErrNull
	}
	for i, wxid := range wxidList {
		for _, member := range rd.Members {
			if member.Wxid == wxid {
				nicknameList[i] = member.NickName
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
	var contactInfoList = make([]*ContactInfo, len(nicknameList))
	if len(nicknameList) == 0 {
		return nil, ErrNull
	}
	for i, nickname := range nicknameList {
		for _, member := range rd.Members {
			if member == nil {
				continue
			}
			if member.NickName == nickname {
				contactInfoList[i] = member
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
	MsgTypeMoments           MsgType = 0       // 朋友圈消息
	MsgTypeText              MsgType = 1       // 文字
	MsgTypeImage             MsgType = 3       // 图片
	MsgTypeVoice             MsgType = 34      // 语音
	MsgTypeFriendConfirm     MsgType = 37      // 好友确认
	MsgTypePossibleFriend    MsgType = 40      // POSSIBLEFRIEND_MSG
	MsgTypeBusinessCard      MsgType = 42      // 名片
	MsgTypeVideo             MsgType = 43      // 视频
	MsgTypeRockPaperScissors MsgType = 47      // 石头剪刀布 | 表情图片
	MsgTypeLocation          MsgType = 48      // 位置
	MsgTypeXML               MsgType = 49      // 共享实时位置、文件、转账、链接、应用消息
	MsgTypeXMLQuote          MsgType = 4901    // 引用消息
	MsgTypeXMLImage          MsgType = 4903    // XML 中的图片消息
	MsgTypeXMLFile           MsgType = 4906    // XML 中的文件消息
	MsgTypeXMLLink           MsgType = 4916    // XML 中的链接消息
	MsgTypeVoip              MsgType = 50      // VOIPMSG
	MsgTypeWechatInit        MsgType = 51      // 微信初始化
	MsgTypeVoipNotify        MsgType = 52      // VOIPNOTIFY
	MsgTypeVoipInvite        MsgType = 53      // VOIPINVITE
	MsgTypeShortVideo        MsgType = 62      // 小视频
	MsgTypeRedPacket         MsgType = 66      // 微信红包 // 436207665
	MsgTypeSysNotice         MsgType = 9999    // SYSNOTICE
	MsgTypeSystem            MsgType = 10000   // 红包、系统消息
	MsgTypeRevoke            MsgType = 10002   // 撤回消息
	MsgTypeSogouEmoji        MsgType = 1048625 // 搜狗表情
	//MsgTypeLink              MsgType = 16777265   // 链接
	//MsgTypeWechatRedPacket   MsgType = 436207665  // 微信红包 // 重复定义
	MsgTypeRedPacketCover    MsgType = 536936497 // 红包封面
	MsgTypeVideoChannelVideo MsgType = 754974769 // 视频号视频
	MsgTypeVideoChannelCard  MsgType = 771751985 // 视频号名片
	//MsgTypeQuote             MsgType = 822083633  // 引用消息
	MsgTypePat               MsgType = 922746929  // 拍一拍
	MsgTypeVideoChannelLive  MsgType = 973078577  // 视频号直播
	MsgTypeProductLink       MsgType = 974127153  // 商品链接
	MsgTypeVideoChannelLive2 MsgType = 975175729  // 视频号直播 // 重复定义
	MsgTypeMusicLink         MsgType = 1040187441 // 音乐链接
	MsgTypeFile              MsgType = 1090519089 // 文件
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
	MsgTypeXML:               "xml消息",
	MsgTypeXMLQuote:          "引用消息",
	MsgTypeXMLImage:          "XML图片",
	MsgTypeXMLFile:           "XML文件",
	MsgTypeXMLLink:           "XML链接",
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
	//MsgTypeLink:              "链接",
	//MsgTypeWechatRedPacket:   "微信红包", // 与 MsgTypeRedPacket 重复
	MsgTypeRedPacketCover:    "红包封面",
	MsgTypeVideoChannelVideo: "视频号视频",
	MsgTypeVideoChannelCard:  "视频号名片",
	//MsgTypeQuote:             "引用消息",
	MsgTypePat:               "拍一拍",
	MsgTypeVideoChannelLive:  "视频号直播",
	MsgTypeProductLink:       "商品链接",
	MsgTypeVideoChannelLive2: "视频号直播", // 与上面的 MsgTypeVideoChannelLive 重复
	MsgTypeMusicLink:         "音乐链接",
	MsgTypeFile:              "文件",
}

// QuoteMsg 引用消息
type QuoteMsg struct {
	Type       int    `xml:"type" json:"type"`
	SvrId      string `xml:"svrid" json:"svrId"`
	FromUser   string `xml:"fromusr" json:"fromUser"`
	ChatUser   string `xml:"chatusr" json:"chatUser"`
	CreateTime int64  `xml:"createtime" json:"createTime"`
	MsgSource  string `xml:"msgsource" json:"msgSource"`
	XMLSource  string `xml:"-" json:"-"`
	Content    string `xml:"content" json:"content"`
}

// ReferMsg 引用的消息
type ReferMsg struct {
	Quote QuoteMsg `xml:"refermsg"`
}

// FileMsg 文件消息
type FileMsg struct {
	Title     string `xml:"title"`
	FileExt   string `xml:"appattach>fileext"`
	AppAttach struct {
		TotalLen string `xml:"totallen"`
	} `xml:"appattach"`
}

type SpecialUserType int

const (
	SpecialUserTypeUnknown     SpecialUserType = iota
	SpecialUserTypeMediaNote                   // 影音号
	SpecialUserTypeFloatBottle                 // 漂流瓶
	SpecialUserTypeFileHelper                  // 文件助手
	SpecialUserTypeFMessage                    // 朋友推荐消息
)

var SpecialUserTypeNames = map[SpecialUserType]string{
	SpecialUserTypeUnknown:     "未知类型",
	SpecialUserTypeMediaNote:   "影音号",
	SpecialUserTypeFloatBottle: "漂流瓶",
	SpecialUserTypeFileHelper:  "文件助手",
	SpecialUserTypeFMessage:    "朋友推荐消息",
}

var SpecialUserTypeValues = map[string]SpecialUserType{
	"medianote":   SpecialUserTypeMediaNote,
	"floatbottle": SpecialUserTypeFloatBottle,
	"filehelper":  SpecialUserTypeFileHelper,
	"fmessage":    SpecialUserTypeFMessage,
}

func GetSpecialUserType(name string) SpecialUserType {
	if value, ok := SpecialUserTypeValues[name]; ok {
		return value
	}
	return SpecialUserTypeUnknown
}
