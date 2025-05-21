package wcf_rpc_sdk

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/Clov614/logging"
	"github.com/Clov614/wcf-rpc-sdk/internal/utils/imgutil"
	"github.com/Clov614/wcf-rpc-sdk/internal/wcf"
	"github.com/antchfx/xmlquery"
	"github.com/eatmoreapple/env"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
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
	ctx         context.Context
	stop        context.CancelFunc
	msgBuffer   *MessageBuffer
	wxClient    *wcf.Client
	addr        string // 接口地址
	self        *Self
	cacheMember *ContactInfoManager // 用户信息缓存 fixme: 更改命名
	closeOnce   sync.Once
	memberLock  sync.Mutex // 查询member操作互斥锁
}

// Close 停止客户端
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.stop()
		if c.cacheMember != nil {
			c.cacheMember.Close() // 释放信息缓存
		}
		err := c.wxClient.Close()
		if err != nil {
			logging.ErrorWithErr(err, "停止wcf客户端发生了错误")
		}
	})
	logging.Warn("wcf-sdk closed!")
}

// NewClient <消息通道大小> <是否自动注入微信（自动打开微信）> <是否开启sdk-debug>
func NewClient(msgChanSize int, autoInject bool, sdkDebug bool) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return newClient(ctx, cancel, msgChanSize, autoInject, sdkDebug)
}

// NewClientWithCtx <上下文> <退出方法> <消息通道大小> <是否自动注入微信（自动打开微信）> <是否开启sdk-debug>
func NewClientWithCtx(ctx context.Context, cancel context.CancelFunc, msgChanSize int, autoInject bool, sdkDebug bool) *Client {
	if ctx == nil {
		panic("ctx is nil")
	}
	return newClient(ctx, cancel, msgChanSize, autoInject, sdkDebug)
}

func newClient(ctx context.Context, cancel context.CancelFunc, msgChanSize int, autoInject bool, sdkDebug bool) *Client {
	addr := env.Name(ENVTcpAddr).StringOrElse(DefaultTcpAddr) // "tcp://127.0.0.1:10086"
	var syncSignal = make(chan struct{})                      // 同步信号 确保注入后处理消息
	if autoInject {                                           // 自动注入
		port, err := strconv.Atoi(addr[strings.LastIndex(addr, ":")+1:])
		if err != nil {
			logging.ErrorWithErr(err, "the port is invalid, please check your address")
			logging.Fatal("canot auto inject!", 1000, map[string]interface{}{"port": port})
		}

		go func() {
			Inject(ctx, cancel, port, sdkDebug, syncSignal) // 调用sdk.dll 注入&启动微信
		}()
		<-syncSignal
	}
	close(syncSignal) // 关闭同步管道
	wxclient, err := wcf.NewWCF(addr)
	if err != nil {
		logging.Fatal(fmt.Errorf("new wcf err: %w", err).Error(), 1001)
		//panic(err)
	}
	return &Client{
		ctx:         ctx,
		stop:        cancel,
		msgBuffer:   NewMessageBuffer(msgChanSize), // 消息缓冲区 <缓冲大小>
		wxClient:    wxclient,
		self:        NewSelf(wxclient),
		addr:        addr,
		cacheMember: NewCacheInfoManager(),
	}
}

// Run 运行tcp监听 以及 请求tcp监听信息 <是否debug>
func (c *Client) Run(debug bool) {
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logging.Debug("Debug mode enabled")
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	// 增加项目字段
	logging.SetField(map[string]interface{}{"sdk": "wcf-rpc-sdk"})
	go func() { // 处理接收消息
		err := c.handleMsg(c.ctx)
		if err != nil {
			logging.Fatal(fmt.Errorf("handle msg err: %w", err).Error(), 1001)
		}
	}()
	go c.cyclicUpdateSelfInfo(true)  // 启动定时更新
	go c.cyclicUpdateCacheInfo(true) // 启动定时更新
}

func (c *Client) IsLogin() bool {
	return c.wxClient.IsLogin()
}

// GetMsgChan 返回消息的管道
func (c *Client) GetMsgChan() <-chan *Message {
	return c.msgBuffer.msgCH
}

// SendText 发送普通文本 <wxid or roomid> <文本内容> <艾特的人(wxid) 所有人:(notify@all)> todo test 重构后待测试
func (c *Client) SendText(receiver string, content string, ats ...string) error {
	// 根据 wxid 获取对应的 Name
	names := make([]string, 0, len(ats))
	atList := make([]string, 0, len(ats))
	for _, wxid := range ats {
		if wxid == "notify@all" {
			names = append(names, "所有人")
			atList = append(atList, "notify@all")
			continue
		}
		m := c.GetMember(wxid, true)
		if m.NickName == "" && m.Alias == "" {
			logging.Debug("sendText NickName && Alias null", map[string]interface{}{"wxid": wxid, "info": m})
			names = append(names, wxid) // 如果获取失败，使用 wxid 代替
			atList = append(atList, wxid)
		} else {
			if m.Alias != "" {
				names = append(names, m.Alias)
			} else if m.NickName != "" {
				names = append(names, m.NickName)
			}
			atList = append(atList, wxid)
		}
	}

	hasAt := strings.Contains(content, "@")

	// 如果内容中不包含 @ 符号，则在开头添加 @<Name>
	if !hasAt {
		for _, name := range names {
			content = "@" + name + " " + content
		}
	} else {
		// 替换 @ 符号
		for _, name := range names {
			content = strings.Replace(content, "@", "@"+name+" ", 1)
		}
	}

	// 发送文本
	res := c.wxClient.SendTxt(content, receiver, atList)
	if res != 0 {
		logging.Debug("wxCliend.SendTxt", map[string]interface{}{"res": res, "receiver": receiver, "content": content, "ats": ats})
		return fmt.Errorf("wxClient.SendTxt err, code: %d", res)
	}
	return nil
}

// SendImage 发送图片 <wxid or roomid> <图片绝对路径>
func (c *Client) SendImage(receiver string, src string) error {
	var tmpFile *os.File    //  声明 tmpFile 变量
	if imgutil.IsURL(src) { // 网络地址
		bytes, err := imgutil.ImgFetch(src)
		if err != nil {
			logging.ErrorWithErr(err, "imgutil.ImgFetch")
			return err
		}
		// 创建临时文件
		tmpFile, err = imgutil.CreateTempFile(".jpg")
		if err != nil {
			logging.ErrorWithErr(err, "imgutil.CreateTempFile")
			return err
		}
		defer func() { // 使用闭包处理 tmpFile.Close() 的错误
			if closeErr := tmpFile.Close(); closeErr != nil {
				logging.ErrorWithErr(closeErr, "tmpFile.Close error in defer")
			}
		}()

		// 写入临时文件
		_, err = tmpFile.Write(bytes)
		if err != nil {
			logging.ErrorWithErr(err, "tmpFile.Write")
			return err
		}
		src = tmpFile.Name() // 使用临时文件路径
	}
	res := c.wxClient.SendIMG(src, receiver)
	if imgutil.IsURL(src) && tmpFile != nil { //  只有网络图片才删除临时文件, 并且确保 tmpFile 不为 nil
		if removeErr := imgutil.RemoveTempFile(tmpFile.Name()); removeErr != nil {
			logging.ErrorWithErr(removeErr, "imgutil.RemoveTempFile error")
		}
	}
	if res != 0 {
		logging.Debug("wxCliend.SendIMG", map[string]interface{}{"res": res, "receiver": receiver, "src": src}) // 打印 src 方便debug
		return fmt.Errorf("wxClient.SendIMG err, code: %d", res)
	}
	return nil
}

// SendImageBytes 发送图片字节数据 <wxid or roomid> <图片字节>
func (c *Client) SendImageBytes(receiver string, imgBytes []byte) error {
	// 创建临时文件
	tmpFile, err := imgutil.CreateTempFile(".jpg") // 假设图片格式为 jpg，如果需要支持其他格式，可以调整
	if err != nil {
		logging.ErrorWithErr(err, "imgutil.CreateTempFile for SendImageBytes")
		return err
	}
	defer func() {
		// 关闭文件
		if closeErr := tmpFile.Close(); closeErr != nil {
			logging.ErrorWithErr(closeErr, "tmpFile.Close error in SendImageBytes defer")
		}
		// 删除临时文件
		if removeErr := imgutil.RemoveTempFile(tmpFile.Name()); removeErr != nil {
			logging.ErrorWithErr(removeErr, "imgutil.RemoveTempFile error in SendImageBytes defer")
		}
	}()

	// 写入临时文件
	_, err = tmpFile.Write(imgBytes)
	if err != nil {
		logging.ErrorWithErr(err, "tmpFile.Write for SendImageBytes")
		return err
	}

	// 获取临时文件路径
	src := tmpFile.Name()

	// 发送图片
	res := c.wxClient.SendIMG(src, receiver)
	if res != 0 {
		logging.Debug("wxCliend.SendIMG from SendImageBytes", map[string]interface{}{"res": res, "receiver": receiver, "src_len": len(imgBytes)}) // 打印字节长度方便debug
		return fmt.Errorf("wxClient.SendIMG from SendImageBytes err, code: %d", res)
	}
	return nil
}

// SendFile 发送图片 <wxid or roomid> <文件绝对路径> todo 支持网络地址发送文件
func (c *Client) SendFile(receiver string, src string) error {
	res := c.wxClient.SendFile(src, receiver)
	if res != 0 {
		logging.Debug("wxCliend.SendFile", map[string]interface{}{"res": res, "receiver": receiver})
		return fmt.Errorf("wxClient.SendFile err, code: %d", res)
	}
	return nil
}

// CardMessage 卡片消息结构体
type CardMessage struct {
	Name     string `json:"name"`      // 卡片名称
	Account  string `json:"account"`   // 账号
	Title    string `json:"title"`     // 标题
	Digest   string `json:"digest"`    // 摘要
	URL      string `json:"url"`       // 链接
	ThumbURL string `json:"thumb_url"` // 缩略图链接
}

// SendCardMessage 发送卡片消息
func (c *Client) SendCardMessage(receiver string, card CardMessage) error {
	res := c.wxClient.SendRichText(card.Name, card.Account, card.Title, card.Digest, card.URL, card.ThumbURL, receiver)
	if res != 1 {
		logging.Debug("wxClient.SendRichText", map[string]interface{}{"res": res, "receiver": receiver, "card": card})
		return fmt.Errorf("wxClient.SendRichText err, code: %d", res)
	}
	return nil
}

// AcceptNewFriend 通过好友请求
func (c *Client) AcceptNewFriend(req NewFriendReq) bool {
	return 1 == c.wxClient.AcceptFriend(req.V3, req.V4, req.Scene) // 1 为成功
}

// CtFriends 获取通讯录所有好友
func (c *Client) CtFriends() ([]Friend, error) {
	fs, b := c.self.CtFriends()
	if !b {
		return nil, fmt.Errorf("self.CtFriends err")
	}
	return fs, nil
}

// CtChatRooms 获取通讯录所有群聊
func (c *Client) CtChatRooms() ([]ChatRoom, error) {
	cr, b := c.self.ChatRooms()
	if !b {
		return nil, fmt.Errorf("self.ChatRooms err")
	}
	return cr, nil
}

// CtGHs 获取通讯录所有公众号
func (c *Client) CtGHs() ([]GH, error) {
	ghs, b := c.self.CtGHs()
	if !b {
		return nil, fmt.Errorf("self.CtGHs err")
	}
	return ghs, nil
}

// RoomMembers 获取群成员信息
func (c *Client) RoomMembers(roomId string) ([]*ContactInfo, error) {
	contacts := c.wxClient.ExecDBQuery("MicroMsg.db", "SELECT RoomData FROM ChatRoom WHERE ChatRoomName = '"+roomId+"';")
	logging.Debug("GetRoomMemberID", map[string]interface{}{"roomId": roomId, "contacts": contacts})

	if len(contacts) == 0 || len(contacts[0].GetFields()) == 0 {
		return nil, fmt.Errorf("no room data found for roomId: %s", roomId)
	}

	roomDataBytes := contacts[0].GetFields()[0].Content

	roomData := &wcf.RoomData{}

	err := proto.Unmarshal(roomDataBytes, roomData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal RoomData: %w", err)
	}
	var roomMembers = make([]*ContactInfo, len(roomData.GetMembers()))
	for i, member := range roomData.GetMembers() {
		roomMembers[i] = c.GetMember(member.Wxid, true)
		roomMembers[i].Wxid = member.Wxid
		roomMembers[i].Alias = member.Name
	}

	return roomMembers, nil
}

// ChatRoomOwner 获取群主
func (c *Client) ChatRoomOwner(roomId string) *ContactInfo {
	res := c.wxClient.ExecDBQuery("MicroMsg.db", "SELECT Reserved2 FROM ChatRoom WHERE ChatRoomName = '"+roomId+"';")
	if res == nil || len(res) == 0 || len(res[0].GetFields()) == 0 {
		logging.Debug("获取群组错误", map[string]interface{}{"roomId": roomId, "res": res})
		return nil
	}
	Reserved2 := res[0].GetFields()[0].Content
	wxid := string(Reserved2)
	info, ok := c.cacheMember.GetContactInfo(wxid)
	if ok || info != nil {
		return info // 返回群主信息
	}
	return nil
}

// GetSelfInfo 获取账号个人信息
func (c *Client) GetSelfInfo() (info SelfInfo, ok bool) { // fixme: 重构
	return c.self.GetSelfInfo()
}

// GetSelfName 获取机器人昵称
func (c *Client) GetSelfName() (string, bool) {
	info, ok := c.self.GetSelfInfo()
	return info.Name, ok
}

// GetSelfWxId 获取机器人微信ID
func (c *Client) GetSelfWxId() (string, bool) {
	info, ok := c.self.GetSelfInfo()
	return info.Wxid, ok
}

// GetSelfFileStoragePath 获取机器人文件存储路径
func (c *Client) GetSelfFileStoragePath() (string, bool) {
	info, ok := c.self.GetSelfInfo()
	return info.FileStoragePath, ok
}

func (c *Client) GetMember(id string, byCache bool) *ContactInfo {
	if byCache { // 走缓存
		info, b := c.cacheMember.GetContactInfo(id)
		if b {
			return info
		}
	}
	var cInfo = &ContactInfo{}
	contacts := c.wxClient.ExecDBQuery("MicroMsg.db", fmt.Sprintf("select * from Contact where UserName = '%s';", id)) // 注意 原字段 UserName指的就是 wxid
	if len(contacts) != 0 {
		c.nomalize(contacts[0], cInfo)
	}
	return cInfo
}

// cyclicUpdateSelfInfo 定时更新机器人信息 <immediate 立即执行一次>
func (c *Client) cyclicUpdateSelfInfo(immediate bool) {
	if immediate {
		c.self.UpdateInfo()
	}
	ticker := time.NewTicker(time.Hour * 2)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.self.UpdateInfo() // 每30 分钟更新一次
			c.self.UpdateContact()
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

// 定时更新用户信息 <immediate 立即执行一次>
func (c *Client) cyclicUpdateCacheInfo(immediate bool) {
	if immediate {
		c.updateCacheInfo(false)
	}
	ticker := time.NewTicker(time.Minute * 30)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.updateCacheInfo(true)
		}
	}
}

// 更新缓存用户信息 <isAsync GetAllMember是否异步>
func (c *Client) updateCacheInfo(isAsync bool) {
	if !c.wxClient.IsLogin() { // fixme: 登入后运行时扔可能获取到登录错误
		logging.WarnWithErr(ErrNotLogin, "[尚未登陆]跳过更新联系人信息")
		return
	}
	if isAsync {
		go c.getAllMember()
	} else {
		c.getAllMember()
	}
}

// getAllMember 获取所有的联系人（包括群聊中的陌生群成员）
func (c *Client) getAllMember() *[]*ContactInfo {
	if !c.memberLock.TryLock() {
		return nil
	}
	defer c.memberLock.Unlock()
	contacts := c.wxClient.ExecDBQuery("MicroMsg.db", "select * from Contact;")
	if len(contacts) == 0 {
		logging.Error("client.getAllMember: queryDB res is nil")
		return nil
	}
	var memberList = make([]*ContactInfo, 0, len(contacts))
	for _, contact := range contacts {
		var cInfo = &ContactInfo{}
		c.nomalize(contact, cInfo)
		memberList = append(memberList, cInfo)
	}
	//logging.Debug("client.getAllMember()", map[string]interface{}{"memberList": memberList})
	return &memberList
}

// 解析 ContactInfo
func (c *Client) nomalize(contact *wcf.DbRow, cInfo *ContactInfo) {
	for _, field := range contact.Fields {
		switch field.Column {
		case "UserName":
			cInfo.Wxid = string(field.Content)
		case "Alias":
			cInfo.Alias = string(field.Content)
		case "DelFlag":
			if num, err := strconv.ParseUint(string(field.Content), 10, 8); err == nil {
				cInfo.DelFlag = uint8(num)
			} else {
				logging.WarnWithErr(err, "error parsing DelFlag")
				cInfo.DelFlag = 0 // todo 或者其他默认值
			}
		case "Type":
			if num, err := strconv.ParseUint(string(field.Content), 10, 8); err == nil {
				cInfo.ContactType = uint8(num)
			} else {
				cInfo.ContactType = 0
			}
		case "Remark":
			cInfo.Remark = string(field.Content)
		case "NickName":
			cInfo.NickName = string(field.Content)
		case "PYInitial":
			cInfo.PyInitial = string(field.Content)
		case "QuanPin":
			cInfo.QuanPin = string(field.Content)
		case "RemarkPYInitial":
			cInfo.RemarkPyInitial = string(field.Content)
		case "RemarkQuanPin":
			cInfo.RemarkQuanPin = string(field.Content)
		case "SmallHeadImgUrl":
			cInfo.SmallHeadURL = string(field.Content)
		case "BigHeadImgUrl":
			cInfo.BigHeadURL = string(field.Content)
		}
	}
	// 查询小头像和大头像
	if cInfo.Wxid != "" {
		query := c.wxClient.ExecDBQuery("MicroMsg.db", fmt.Sprintf("select * from ContactHeadImgUrl where usrName = '%s';", cInfo.Wxid))
		for _, row := range query {
			for _, field := range row.Fields {
				switch field.Column {
				case "smallHeadImgUrl":
					cInfo.SmallHeadURL = string(field.Content)
				case "bigHeadImgUrl":
					cInfo.BigHeadURL = string(field.Content)
				}
			}
		}
	}
	c.cacheMember.CacheContactInfo(cInfo) // 更新缓存
}

// GetFullFilePathFromRelativePath 通过相对路径获取完整文件路径
func (c *Client) GetFullFilePathFromRelativePath(relativePath string) string {
	fileStoragePath, ok := c.GetSelfFileStoragePath()
	if fileStoragePath == "" || !ok {
		logging.Error("GetFullFilePathFromRelativePath: FileStoragePath is empty")
		return "" // 或者返回错误
	}
	// todo 后续可能支持其他的dat解析
	// MsgAttach
	fullFilePath := filepath.Join(fileStoragePath, "MsgAttach", relativePath)
	return filepath.ToSlash(fullFilePath) // 使用 filepath.ToSlash 转换为正斜杠
}

// DecodeDatFileToBytes 解码 .dat 文件为图片, 并返回字节数组
func (c *Client) DecodeDatFileToBytes(datPath string) []byte {
	bytes, err := imgutil.DecodeDatFileToBytes(datPath)
	if err != nil {
		logging.ErrorWithErr(err, "DecodeDatFileToBytes", nil)
		return nil
	}
	return bytes
}

func (c *Client) handleMsg(ctx context.Context) (err error) {
	var handler wcf.MsgHandler = func(msg *wcf.WxMsg) error { // 回调函数
		covertedMsg := c.covertMsg(msg)
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
		//c.wxClient.DisableRecvTxt()          // 重置可能的状态
		c.wxClient.EnableRecvTxt()           // 允许接收消息
		err = c.wxClient.OnMSG(ctx, handler) // 当消息到来时，处理消息
		if err != nil {
			logging.ErrorWithErr(err, "handlerMsg err")
		}
	}()
	return nil
}

func (c *Client) covertMsg(msg *wcf.WxMsg) *Message {
	if msg == nil {
		logging.ErrorWithErr(ErrNull, "internal msg is nil")
		return nil
	}
	var roomMembers []*ContactInfo
	if msg.IsGroup { // 群聊消息
		member, err := c.RoomMembers(msg.Roomid)
		if err != nil {
			logging.Debug("get room member err", map[string]interface{}{"err": err.Error()})
		}
		roomMembers = member
	} else { // 不是群组消息
		msg.Roomid = "" // 置空
	}
	rd := &RoomData{Members: roomMembers}
	id, b := c.GetSelfWxId()
	if b {
		rd.AnalyseMemberAt(id, msg.Content)
	}
	m := &Message{
		IsSelf:    msg.IsSelf,
		IsGroup:   msg.IsGroup,
		IsGH:      strings.HasPrefix(msg.Sender, "gh_"), // 是否为公众号
		MessageId: msg.Id,
		Type:      MsgType(msg.Type),
		Ts:        msg.Ts,
		RoomId:    msg.Roomid,
		RoomData:  rd,
		Content:   msg.Content,
		WxId:      msg.Sender,
		Sign:      msg.Sign,
		Thumb:     msg.Thumb,
		Extra:     msg.Extra,
		Xml:       msg.Xml,
	}
	// 好友申请解析
	if m.Type == MsgTypeFriendConfirm {
		fillNewFriendReq(m)
	}

	// 图片数据解析
	if m.Type == MsgTypeImage {
		time.Sleep(50 * time.Microsecond)
		c.wxClient.DownloadAttach(m.MessageId, m.Thumb, m.Extra) // 下载图片
		m.FileInfo = &FileInfo{FilePath: filepath.ToSlash(m.Extra), IsImg: true}
	}

	// 解析XML
	if msg.Type == uint32(MsgTypeXML) { // 49
		if strings.Contains(msg.Content, "<refermsg>") {
			referMsg, content, err := parseReferMsg(msg.Content)
			if err != nil {
				logging.Debug("parseReferMsg", map[string]interface{}{"err": err, "xml": msg.Xml})
			} else {
				m.Type = MsgTypeXMLQuote
				m.Quote = &referMsg.Quote
				m.Content = content
			}
		} else if strings.Contains(msg.Content, "<recorditem>") { // 新增的转发消息解析逻辑
			forwardMsg, err := parseForwardMsg(msg.Content)
			if err != nil {
				logging.Debug("parseForwardMsg", map[string]interface{}{"err": err, "xml": msg.Xml})
			} else {
				m.Type = MsgTypeXMLForward // 假设您已经定义了这个新的消息类型
				m.Forward = forwardMsg
			}
		} else {
			// 检查是否是文件类型
			fileMsg := &FileMsg{}
			err := xml.Unmarshal([]byte(msg.Content), fileMsg)
			if err != nil {
				logging.Debug("xml.Unmarshal fileMsg", map[string]interface{}{"err": err, "xml": msg.Xml})
			} else {
				if fileMsg.FileExt != "" {
					m.Type = MsgTypeXMLFile
					m.Content = fileMsg.Title
				}
			}
		}
	}

	var sender = m.WxId
	if m.IsGroup { // 群组则回复消息至群组
		sender = m.RoomId
	}
	metaData := &meta{ // meta用于让消息可以直接调用回复
		rawMsg: m,
		sender: sender,
		cli:    c,
		self:   c.self,
	}
	m.meta = metaData
	return m
}

func fillNewFriendReq(m *Message) {
	if m.Content != "" { // 确保 Content 不为空
		doc, err := xmlquery.Parse(strings.NewReader(m.Content))
		if err != nil {
			logging.ErrorWithErr(err, "Failed to parse friend request XML", map[string]interface{}{"messageId": m.MessageId, "content": m.Content})
		} else {
			msgNode := xmlquery.FindOne(doc, "/msg") // 查找根节点 <msg>
			if msgNode != nil {
				v3 := msgNode.SelectAttr("encryptusername") // 提取 v3 (encryptusername)
				v4 := msgNode.SelectAttr("ticket")          // 提取 v4 (ticket)
				sceneStr := msgNode.SelectAttr("scene")     // 提取 scene 字符串

				var sceneVal int64
				if sceneStr != "" {
					sceneVal, err = strconv.ParseInt(sceneStr, 10, 64) // 解析 scene 为 int
					if err != nil {
						logging.ErrorWithErr(err, "Failed to parse scene attribute in friend request", map[string]interface{}{"messageId": m.MessageId, "sceneStr": sceneStr})
						// 解析失败，可以设置默认值或保持为0
						sceneVal = 0
					}
				} else {
					// scene 属性可能不存在或为空
					logging.Warn("Scene attribute missing or empty in friend request", map[string]interface{}{"messageId": m.MessageId})
					sceneVal = 0 // 默认值
				}

				// 创建并填充 NewFriendReq 结构体
				m.NewFriendReq = &NewFriendReq{
					V3:    v3,
					V4:    v4,
					Scene: sceneVal, // 转换为 int32
				}
				logging.Debug("Parsed friend request", map[string]interface{}{"v3": v3, "v4_len": len(v4), "scene": m.NewFriendReq.Scene}) // 打印 V4 长度避免日志过长
			} else {
				logging.Error("Could not find <msg> node in friend request XML", map[string]interface{}{"messageId": m.MessageId, "content": m.Content})
			}
		}
	} else {
		logging.Warn("Friend request message content is empty", map[string]interface{}{"messageId": m.MessageId})
	}
}

func parseReferMsg(xmlStr string) (*ReferMsg, string, error) {
	doc, err := xmlquery.Parse(strings.NewReader(xmlStr))
	if err != nil {
		return nil, "", fmt.Errorf("xmlquery.Parse error: %w", err)
	}

	// 使用 XPath 查找最内层的 refermsg 节点
	referNode := xmlquery.FindOne(doc, "//refermsg[not(refermsg)]")
	if referNode == nil {
		return nil, "", nil // 没有找到 refermsg 节点
	}

	// 提取并反转义每个字段的值
	referMsg := &ReferMsg{
		Quote: QuoteMsg{
			Type:       getInt(referNode, "type"),
			SvrId:      getString(referNode, "svrid"),
			FromUser:   getString(referNode, "fromusr"),
			ChatUser:   getString(referNode, "chatusr"),
			CreateTime: getInt64(referNode, "createtime"),
			MsgSource:  getString(referNode, "msgsource"), // 可能需要进一步处理
			XMLSource:  getString(referNode, "content"),
			Content:    getReferMsgContentTitle(referNode), // 获取引用的消息的 title
		},
	}

	return referMsg, getString(doc, "//appmsg/title"), nil
}

// 辅助函数：提取字符串并进行反转义
func getString(node *xmlquery.Node, xpath string) string {
	strNode := xmlquery.FindOne(node, xpath)
	if strNode == nil {
		return ""
	}
	return html.UnescapeString(strNode.InnerText())
}

// 辅助函数：提取整数
func getInt(node *xmlquery.Node, xpath string) int {
	str := getString(node, xpath) // 复用 getString 函数
	if str == "" {
		return 0
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return 0 // 或者根据需要处理错误
	}
	return val
}

// 辅助函数：提取 int64
func getInt64(node *xmlquery.Node, xpath string) int64 {
	str := getString(node, xpath) // 复用 getString 函数
	if str == "" {
		return 0
	}
	val, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0 // 或者根据需要处理错误
	}
	return val
}

// 辅助函数：提取 refermsg 中 content 字段内的 title
func getReferMsgContentTitle(referNode *xmlquery.Node) string {
	content := getString(referNode, "content")
	if content == "" {
		return ""
	}

	// 循环反转义
	for strings.Contains(content, "&amp;") {
		content = html.UnescapeString(content)
	}

	// 解析为 XML
	contentDoc, err := xmlquery.Parse(strings.NewReader(content))
	if err != nil {
		return "" // 或者根据需要处理错误
	}

	// 提取 title
	titleNode := xmlquery.FindOne(contentDoc, "//title")
	if titleNode == nil {
		return ""
	}

	return titleNode.InnerText()
}

func parseForwardMsg(xmlStr string) (*ForwardMsg, error) {
	// 循环反转义
	for strings.Contains(xmlStr, "&amp;") {
		xmlStr = html.UnescapeString(xmlStr)
	}

	doc, err := xmlquery.Parse(strings.NewReader(xmlStr))
	if err != nil {
		return nil, fmt.Errorf("xmlquery.Parse error: %w", err)
	}

	// 新增: 获取 fromusername
	fromUsernameNode := xmlquery.FindOne(doc, "//fromusername")
	fromUsername := ""
	if fromUsernameNode != nil {
		fromUsername = fromUsernameNode.InnerText()
	}

	// 查找 recorditem 节点
	recordItemNode := xmlquery.FindOne(doc, "//recorditem")
	if recordItemNode == nil {
		return nil, fmt.Errorf("recorditem node not found")
	}

	// 获取 recorditem 节点的 InnerText 并进行 HTML 反转义
	recordInfoStr := html.UnescapeString(recordItemNode.InnerText())

	// 使用反转义后的字符串创建一个新的 XML 文档
	recordInfoDoc, err := xmlquery.Parse(strings.NewReader(recordInfoStr))
	if err != nil {
		return nil, fmt.Errorf("xmlquery.Parse recordinfo error: %w", err)
	}

	// 查找 recordinfo 节点
	recordInfoNode := xmlquery.FindOne(recordInfoDoc, "//recordinfo")
	if recordInfoNode == nil {
		return nil, fmt.Errorf("recordinfo node not found")
	}

	forwardMsg := &ForwardMsg{
		Title:        getString(recordInfoNode, "title"),
		Desc:         getString(recordInfoNode, "desc"),
		DataList:     []ForwardMsgDataItem{},
		FromUsername: fromUsername, // 设置 FromUsername
	}

	// 查找 datalist 下的所有 dataitem
	dataItemNodes := xmlquery.Find(recordInfoNode, "datalist/dataitem")
	for _, dataItemNode := range dataItemNodes {
		dataItem := ForwardMsgDataItem{
			DataId:        getString(dataItemNode, "@dataid"),
			DataType:      getInt(dataItemNode, "@datatype"),
			DataDesc:      getString(dataItemNode, "datadesc"),
			SourceName:    getString(dataItemNode, "sourcename"),
			SourceTime:    getString(dataItemNode, "sourcetime"),
			SourceHeadURL: getString(dataItemNode, "sourceheadurl"),
			FromNewMsgId:  getInt64(dataItemNode, "fromnewmsgid"),
			CdnDataUrl:    getString(dataItemNode, "cdndataurl"),
			CdnThumbUrl:   getString(dataItemNode, "cdnthumburl"),
			DataFmt:       getString(dataItemNode, "datafmt"),
			FullMd5:       getString(dataItemNode, "fullmd5"),
			ThumbFullMd5:  getString(dataItemNode, "thumbfullmd5"),
			CdnThumbKey:   getString(dataItemNode, "cdnthumbkey"),
			CdnDataKey:    getString(dataItemNode, "cdndatakey"),
		}
		forwardMsg.DataList = append(forwardMsg.DataList, dataItem)
	}

	return forwardMsg, nil
}
