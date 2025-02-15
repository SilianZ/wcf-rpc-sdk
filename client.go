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
	"html"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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
	ctx       context.Context
	stop      context.CancelFunc
	msgBuffer *MessageBuffer
	wxClient  *wcf.Client
	addr      string // 接口地址
	self      *Self
	cacheUser *CacheUserManager // 用户信息缓存
}

// Close 停止客户端
func (c *Client) Close() {
	c.stop()
	if c.cacheUser != nil {
		c.cacheUser.Close() // 释放信息缓存
	}
	err := c.wxClient.Close()
	if err != nil {
		logging.ErrorWithErr(err, "停止wcf客户端发生了错误")
	}
}

// NewClient <消息通道大小> <是否自动注入微信（自动打开微信）> <是否开启sdk-debug>
func NewClient(msgChanSize int, autoInject bool, sdkDebug bool) *Client {
	addr := env.Name(ENVTcpAddr).StringOrElse(DefaultTcpAddr) // "tcp://127.0.0.1:10086"
	ctx, cancel := context.WithCancel(context.Background())
	var syncSignal = make(chan struct{}) // 同步信号 确保注入后处理消息
	if autoInject {                      // 自动注入
		port, err := strconv.Atoi(addr[strings.LastIndex(addr, ":")+1:])
		if err != nil {
			logging.ErrorWithErr(err, "the port is invalid, please check your address")
			logging.Fatal("canot auto inject!", 1000, map[string]interface{}{"port": port})
		}

		go func() {
			Inject(ctx, port, sdkDebug, syncSignal) // 调用sdk.dll 注入&启动微信
		}()

	}
	if autoInject {
		<-syncSignal
	}
	close(syncSignal) // 关闭同步管道
	//logging.Warn("30s后启动wcf客户端，请确保登录微信")
	//time.Sleep(30 * time.Second)
	wxclient, err := wcf.NewWCF(addr)
	if err != nil {
		logging.Fatal(fmt.Errorf("new wcf err: %w", err).Error(), 1001)
		//panic(err)
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
		roomMemberIds, err := c.GetRoomMemberID(msg.Roomid)
		if err != nil {
			logging.ErrorWithErr(err, "GetRoomMemberID")
		} else {
			roomMembers, err = c.GetMember(roomMemberIds...)
			if err != nil {
				logging.ErrorWithErr(err, "GetMember")
			}
		}
	} else { // 不是群组消息
		msg.Roomid = "" // 置空
	}

	m := &Message{
		IsSelf:    msg.IsSelf,
		IsGroup:   msg.IsGroup,
		MessageId: msg.Id,
		Type:      MsgType(msg.Type),
		Ts:        msg.Ts,
		RoomId:    msg.Roomid,
		RoomData:  &RoomData{Members: roomMembers},
		Content:   msg.Content,
		WxId:      msg.Sender,
		Sign:      msg.Sign,
		Thumb:     msg.Thumb,
		Extra:     msg.Extra,
		Xml:       msg.Xml,
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
	}
	m.meta = metaData
	return m
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

// SendText 发送普通文本 <wxid or roomid> <文本内容> <艾特的人(wxid) 所有人:(notify@all)>
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
		friend, err := c.GetMember(wxid)
		if len(friend) == 0 || err != nil {
			logging.WarnWithErr(err, "SendText.GetMember err")
			names = append(names, wxid) // 如果获取失败，使用 wxid 代替
			atList = append(atList, wxid)
		} else {
			names = append(names, friend[0].NickName)
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
	res := c.wxClient.SendIMG(src, receiver)
	if res != 0 {
		logging.Debug("wxCliend.SendIMG", map[string]interface{}{"res": res, "receiver": receiver})
		return fmt.Errorf("wxClient.SendIMG err, code: %d", res)
	}
	return nil
}

// SendFile 发送图片 <wxid or roomid> <文件绝对路径>
func (c *Client) SendFile(receiver string, src string) error {
	res := c.wxClient.SendFile(src, receiver)
	if res != 0 {
		logging.Debug("wxCliend.SendFile", map[string]interface{}{"res": res, "receiver": receiver})
		return fmt.Errorf("wxClient.SendFile err, code: %d", res)
	}
	return nil
}

// GetRoomMemberID 获取群成员信息，返回解码后的字符串以及 wxid 列表
func (c *Client) GetRoomMemberID(roomId string) ([]string, error) {
	contacts := c.wxClient.ExecDBQuery("MicroMsg.db", "SELECT RoomData FROM ChatRoom WHERE ChatRoomName = '"+roomId+"';")
	logging.Debug("GetRoomMemberID", map[string]interface{}{"roomId": roomId, "contacts": contacts})

	if len(contacts) == 0 || len(contacts[0].GetFields()) == 0 {
		return nil, fmt.Errorf("no room data found for roomId: %s", roomId)
	}

	decodedString := string(contacts[0].GetFields()[0].Content)
	logging.Debug("GetRoomMemberID", map[string]interface{}{"roomId": roomId, "decodedString": decodedString}) // 打印解码后的字符串

	// 使用正则表达式提取 wxid
	re := regexp.MustCompile(`(wxid_[a-zA-Z0-9]+)`)
	matches := re.FindAllStringSubmatch(decodedString, -1)

	var wxids []string
	for _, match := range matches {
		wxids = append(wxids, match[1])
	}

	return wxids, nil
}

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
	self.FileStoragePath = filepath.Join(u.Home, u.Wxid, "FileStorage")
	c.self = self // 更新缓存
	return self
}

// GetSelfName 获取机器人昵称
func (c *Client) GetSelfName() string {
	if c.self == nil || c.self.Name == "" {
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

// GetSelfFileStoragePath 获取机器人文件存储路径
func (c *Client) GetSelfFileStoragePath() string {
	if c.self == nil || c.self.FileStoragePath == "" {
		c.GetSelfInfo() // 更新缓存
	}
	if c.self == nil {
		return ""
	}
	return c.self.FileStoragePath
}

// cyclicUpdateSelfInfo 定时更新机器人信息 <immediate 立即执行一次>
func (c *Client) cyclicUpdateSelfInfo(immediate bool) {
	if immediate {
		c.GetSelfInfo()
	}
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
		if t == memberType || t == roomType {
			c.updateCacheInfo(true, retry <= 0)
		} else {
			c.updateCacheInfo(false, retry <= 0)
		}
		return c.getInfo(wxid, isAll, t, retry-1, f)
	}
	return result, err
}

// GetFriend 根据wxid获取好友信息
func (c *Client) GetFriend(wxid string) (*Friend, error) {
	info, err := c.getInfo(wxid, false, friendType, 3, c.cacheUser.Get)
	if err != nil {
		return nil, err
	}
	res, _ := info.(*Friend)
	return res, nil
}

// GetAllFriend 获取所有好友信息
func (c *Client) GetAllFriend() (*FriendList, error) {
	info, err := c.getInfo("", true, friendType, 3, c.cacheUser.Get)
	if err != nil {
		logging.ErrorWithErr(err, "get all friend err")
		return nil, err
	}
	res, _ := info.(*FriendList)
	return res, nil
}

// GetChatRoom 根据roomId获取群组信息 todo 完善ChatRoom字段
func (c *Client) GetChatRoom(roomId string) (*ChatRoom, error) {
	info, err := c.getInfo(roomId, false, roomType, 3, c.cacheUser.Get)
	if err != nil {
		logging.ErrorWithErr(err, "get all friend err")
		return nil, err
	}
	res, _ := info.(*ChatRoom)
	return res, nil
}

// GetAllChatRoom 获取所有群组信息
func (c *Client) GetAllChatRoom() (*ChatRoomList, error) {
	info, err := c.getInfo("", true, roomType, 3, c.cacheUser.Get)
	if err != nil {
		logging.ErrorWithErr(err, "get all friend err")
		return nil, err
	}
	res, _ := info.(*ChatRoomList)
	return res, nil
}

// GetMember 根据wxid获取成员（包括群组陌生人）
func (c *Client) GetMember(wxidList ...string) ([]*ContactInfo, error) {
	var result = make([]*ContactInfo, 0, len(wxidList))
	for _, wxid := range wxidList {
		info, err := c.getInfo(wxid, false, memberType, 3, c.cacheUser.Get)
		if err != nil {
			logging.ErrorWithErr(err, "get member by wxid err")
			continue
		}
		result = append(result, info.(*ContactInfo))
	}
	if len(result) == 0 {
		return nil, ErrNull
	}
	return result, nil
}

// GetAllMember 获取全部成员（包括群组陌生人）
func (c *Client) GetAllMember() ([]*ContactInfo, error) {
	info, err := c.getInfo("", true, memberType, 3, c.cacheUser.Get)
	if err != nil {
		logging.ErrorWithErr(err, "get all member err")
		return nil, err
	}
	res, _ := info.([]*ContactInfo)
	return res, nil
}

// 定时更新用户信息 <immediate 立即执行一次>
func (c *Client) cyclicUpdateCacheInfo(immediate bool) {
	if immediate {
		c.updateCacheInfo(true, true)
	}
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.updateCacheInfo(true, true) // 每分钟更新一次
		}
	}
}

// 更新缓存用户信息 <isAsync GetAllMember是否异步> <是否输出错误日志>
func (c *Client) updateCacheInfo(IsGetMember bool, isLogErr bool) {
	contacts := c.wxClient.GetContacts()
	if len(contacts) == 0 {
		logging.ErrorWithErr(ErrNull, "get contacts err")
		return
	}
	if IsGetMember {
		c.cacheUser.UpdateMembers(c.getAllMember()) // 查询数据库获取全部联系人并更新
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
		case ChatRoom: // todo 完善chatRoom字段
			logging.Debug("updateCacheInfo", map[string]interface{}{"user": user, "chatroom": v})
			v.RoomID = v.Wxid
			roomMemberIds, err := c.GetRoomMemberID(v.RoomID)
			if err != nil {
				if isLogErr {
					logging.WarnWithErr(err, "get room member id err")
				}
			} else {
				members, err := c.cacheUser.GetMemberByList(roomMemberIds...)
				if err != nil {
					if isLogErr {
						logging.WarnWithErr(err, "get room member err")
					}
				} else {
					v.RoomData = &RoomData{Members: members}
				}
			}
			room, err := c.cacheUser.GetMember(v.RoomID)
			if err != nil {
				if isLogErr {
					logging.WarnWithErr(err, "get room err")
				}

			} else {
				v.RoomHeadImgURL = &room.SmallHeadURL
				//todo 公告字段 v.RoomAnnouncement
			}
			c.cacheUser.updateChatRoom(&v)
		case GH:
			logging.Debug("updateCacheInfo", map[string]interface{}{"user": user, "gh": v})
		// todo cache GH
		case User: // User 类型为除了上方类型的特殊类型，如文件助手、漂流瓶等
			logging.Debug("updateCacheInfo", map[string]interface{}{"user": user})
		default: // 未知类型
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
		return ChatRoom{User: user}
	case strings.HasPrefix(ct.Wxid, "gh_"):
		return GH(user)
	default:
		specialUserType := GetSpecialUserType(ct.Wxid)
		if specialUserType != SpecialUserTypeUnknown {
			return user
		}
		logging.Warn("unknown contact type", map[string]interface{}{"type": ct.Wxid})
		return nil
	}
}

// getAllMember 获取所有的联系人（包括群聊中的陌生群成员）
func (c *Client) getAllMember() *[]*ContactInfo {
	contacts := c.wxClient.ExecDBQuery("MicroMsg.db", "select * from Contact;")
	if len(contacts) == 0 {
		logging.Error("client.getAllMember: contact not found")
		return nil
	}
	var memberList = make([]*ContactInfo, 0, len(contacts))
	for _, contact := range contacts {
		var cInfo = &ContactInfo{}
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
		memberList = append(memberList, cInfo)
	}
	logging.Debug("client.getAllMember()", map[string]interface{}{"memberList": memberList})
	return &memberList
}

// GetFullFilePathFromRelativePath 通过相对路径获取完整文件路径
func (c *Client) GetFullFilePathFromRelativePath(relativePath string) string {
	fileStoragePath := c.GetSelfFileStoragePath()
	if fileStoragePath == "" {
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
