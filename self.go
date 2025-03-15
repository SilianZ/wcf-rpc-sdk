// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/3/15 下午2:39:00
// @Desc
package wcf_rpc_sdk

import (
	"github.com/Clov614/logging"
	"github.com/Clov614/wcf-rpc-sdk/internal/wcf"
	"path/filepath"
	"strings"
	"sync"
)

type Self struct { // 机器人自己
	cli *wcf.Client
	User
	Mobile          string `json:"mobile,omitempty"` // 个人信息时携带
	Home            string `json:"home,omitempty"`   // C:/Users/Administrator/Documents/WeChat Files/
	FileStoragePath string `json:"fileStoragePath"`  // C:/Users/Administrator/Documents/WeChat Files/wxid_p5z4fuhnbdgs22/FileStorage/
	// below contact Field
	Friends FriendMp   `json:"-"` // 朋友列表
	Rooms   ChatRoomMp `json:"-"` // 加入的群列表
	GHs     GHMp       `json:"-"` // 关注的公众号列表
	mu      sync.RWMutex
}

type SelfInfo struct { // 保护隐藏self信息
	Wxid            string `json:"wxid,omitempty"`
	Name            string `json:"name,omitempty"`
	Mobile          string `json:"mobile,omitempty"`
	Home            string `json:"home,omitempty"`  // C:/Users/Administrator/Documents/WeChat Files/
	FileStoragePath string `json:"fileStoragePath"` // C:/Users/Administrator/Documents/WeChat Files/wxid_p5z4fuhnbdgs22/FileStorage/
}

func NewSelf(cli *wcf.Client) *Self {
	return &Self{cli: cli, Friends: make(FriendMp), Rooms: make(ChatRoomMp), GHs: make(GHMp)}
}

// GetSelfInfo 获取个人账号信息 <getLatest true: 缓存获取到时是否异步获取>
func (s *Self) GetSelfInfo() (info SelfInfo, ok bool) {
	info = s.getSelfInfo(false)
	if info.Wxid == "" {
		info = s.getSelfInfo(true)
	}
	if info.Wxid == "" { // double check
		return info, false
	}
	return info, true
}

func (s *Self) getSelfInfo(getLatest bool) SelfInfo {
	if getLatest {
		info := s.cli.GetUserInfo()
		s.Wxid = info.Wxid
		s.Name = info.Name
		s.Home = info.Home
		s.Mobile = info.Mobile
		s.FileStoragePath = filepath.Join(info.Home, info.Wxid, "FileStorage")
	}
	return SelfInfo{
		Wxid:            s.Wxid,
		Name:            s.Name,
		Mobile:          s.Mobile,
		Home:            s.Home,
		FileStoragePath: s.FileStoragePath,
	}
}

type IsType uint8

const (
	IsFriend = iota
	IsRoom
	IsGH
)

func (s *Self) Is(id string, t IsType) (ok bool) {
	return s.is(id, t, 1)
}

func (s *Self) is(id string, t IsType, retry int) (ok bool) {
	var b bool
	switch t {
	case IsFriend:
		ok, b = s.IsMyFriend(id)
	case IsRoom:
		ok, b = s.IsInRoom(id)
	case IsGH:
		ok, b = s.IsFollowGH(id)
	}
	if !b { // 获取缓存失败
		s.UpdateContact() // 更新通讯录
	}
	if retry == 0 || b {
		return ok
	}
	return s.is(id, t, retry-1)
}

func (s *Self) IsMyFriend(wxId string) (isFriend bool, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Friends == nil || len(s.Friends) == 0 {
		return false, false
	}
	_, ok = s.Friends[wxId]
	return ok, true
}
func (s *Self) IsInRoom(roomId string) (isInRoom bool, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Rooms == nil || len(s.Rooms) == 0 {
		return false, false
	}
	_, ok = s.Rooms[roomId]
	return ok, true
}
func (s *Self) IsFollowGH(ghId string) (isFollow bool, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.GHs == nil || len(s.GHs) == 0 {
		return false, false
	}
	_, ok = s.GHs[ghId]
	return ok, true
}

func (s *Self) IsSendByFriend(wxid string) (isFriend bool) {
	return s.Is(wxid, IsFriend)
}

func (s *Self) UpdateInfo() (success bool) {
	if !s.mu.TryLock() {
		logging.Debug("try UpdateInfo give up! cause: Busy")
		return false
	}
	defer s.mu.Unlock()
	info := s.cli.GetUserInfo()
	if info == nil {
		logging.Debug("self.UpdateInfo() s.cli.GetUserInfo nil")
		return false
	}
	s.Wxid = info.Wxid
	s.Name = info.Name
	s.Mobile = info.Mobile
	s.Home = info.Home
	s.FileStoragePath = filepath.Join(info.Home, info.Wxid, "FileStorage")
	return true
}

func (s *Self) UpdateContact() (success bool) {
	if !s.mu.TryLock() {
		logging.Debug("try UpdateContact failed cause: Busy!")
		return false
	}
	defer s.mu.Unlock()
	contacts := s.cli.GetContacts()
	for _, ct := range contacts {
		u := User{
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
			s.Friends[u.Wxid] = Friend(u)
		case strings.HasSuffix(ct.Wxid, "@chatroom"):
			s.Rooms[u.Wxid] = ChatRoom{User: u}
		case strings.HasPrefix(ct.Wxid, "gh_"):
			s.GHs[u.Wxid] = GH(u)
		}
	}
	return true
}
