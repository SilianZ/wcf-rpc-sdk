// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/1/17 上午11:39:00
// @Desc 用户缓存器
package wcf_rpc_sdk

import (
	"fmt"
	"github.com/Clov614/logging"
	"sync"
)

type InfoType uint32

const (
	friendType InfoType = iota
	roomType
	ghType
	memberType
)

// CacheUserManager 缓存管理器
type CacheUserManager struct {
	friendCache *sync.Map // 好友信息缓存，key: wxid, value: *Friend
	roomCache   *sync.Map // 群组信息缓存，key: room_id, value: *ChatRoom
	memberCache *sync.Map // todo 所有相关联系人（包括群组成员）缓存
	friendCount int       // 好友数量计数器
	roomCount   int       // 群组数量计数器
	memberCount int       // 所有联系人缓存计数器
	// todo 公众号
}

// NewCacheInfoManager 创建缓存管理器
func NewCacheInfoManager() *CacheUserManager {
	return &CacheUserManager{
		friendCache: &sync.Map{},
		roomCache:   &sync.Map{},
		memberCache: &sync.Map{},
		friendCount: 0,
		roomCount:   0,
	}
}

func (cm *CacheUserManager) Get(id string, isAll bool, t InfoType) (any, error) {
	switch t {
	case friendType:
		if isAll {
			return cm.getAllFriend()
		}
		return cm.getFriend(id)
	case roomType:
		if isAll {
			return cm.getAllChatRoom()
		}
		return cm.getChatRoom(id)
	case ghType: // todo
		logging.Warn("un support ghType")
		return nil, ErrNull
	case memberType:
		if isAll {
			return cm.GetAllMember()
		}
		return cm.GetMember(id)
	}
	return nil, ErrNull
}

// getFriend 获取用户信息
func (cm *CacheUserManager) getFriend(wxid string) (*Friend, error) {
	value, ok := cm.friendCache.Load(wxid)
	if !ok {
		return nil, fmt.Errorf("user not found: %s", wxid)
	}
	return value.(*Friend), nil
}

// updateFriend 更新用户信息
func (cm *CacheUserManager) updateFriend(friend *Friend) {
	if _, ok := cm.friendCache.Load(friend.Wxid); !ok {
		cm.friendCount++
	}
	cm.friendCache.Store(friend.Wxid, friend)
}

// getChatRoom 获取群组信息
func (cm *CacheUserManager) getChatRoom(chatRoomId string) (*ChatRoom, error) {
	value, ok := cm.roomCache.Load(chatRoomId)
	if !ok {
		return nil, fmt.Errorf("room not found: %s", chatRoomId)
	}
	return value.(*ChatRoom), nil
}

// updateChatRoom 更新群组信息
func (cm *CacheUserManager) updateChatRoom(chatRoom *ChatRoom) {
	if _, ok := cm.roomCache.Load(chatRoom.RoomID); !ok {
		cm.roomCount++
	}
	cm.roomCache.Store(chatRoom.RoomID, chatRoom)
}

// getAllFriend 获取全部好友
func (cm *CacheUserManager) getAllFriend() (*FriendList, error) {
	friendList := make([]*Friend, 0, cm.friendCount)
	cm.friendCache.Range(func(key, value interface{}) bool {
		friendList = append(friendList, value.(*Friend))
		return true
	})
	list := FriendList(friendList)
	return &list, nil
}

// getAllChatRoom 获取全部群组
func (cm *CacheUserManager) getAllChatRoom() (*ChatRoomList, error) {
	chatRoomList := make([]*ChatRoom, 0, cm.roomCount)
	cm.roomCache.Range(func(key, value interface{}) bool {
		chatRoomList = append(chatRoomList, value.(*ChatRoom))
		return true
	})
	list := ChatRoomList(chatRoomList)
	return &list, nil
}

// GetMember 获取联系人（包括群聊陌生群成员）
func (cm *CacheUserManager) GetMember(wxId string) (*ContactInfo, error) {
	value, ok := cm.memberCache.Load(wxId)
	if !ok {
		return nil, fmt.Errorf("contact not found: %s", wxId)
	}
	return value.(*ContactInfo), nil
}

// GetMemberByList 通过多个wxid 获取联系人列表（包括群聊陌生群成员）
func (cm *CacheUserManager) GetMemberByList(wxIdList ...string) ([]*ContactInfo, error) {
	var list = make([]*ContactInfo, 0, len(wxIdList))
	for _, wxId := range wxIdList {
		value, ok := cm.memberCache.Load(wxId)
		if !ok {
			continue
		}
		list = append(list, value.(*ContactInfo))
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("contact not found: %s", wxIdList)
	}
	return list, nil
}

// GetAllMember 获取全部联系人（包括群聊陌生群成员）
func (cm *CacheUserManager) GetAllMember() ([]*ContactInfo, error) {
	list := make([]*ContactInfo, 0, cm.memberCount)
	cm.memberCache.Range(func(key, value interface{}) bool {
		list = append(list, value.(*ContactInfo))
		return true
	})
	return list, nil
}

// UpdateMembers 更新所有联系人（包括群聊陌生群成员）
func (cm *CacheUserManager) UpdateMembers(list *[]*ContactInfo) {
	if nil == list {
		logging.Error("update contacts failed, list is nil")
		return
	}
	for _, info := range *list {
		if _, ok := cm.memberCache.Load(info.Wxid); !ok {
			cm.memberCount++
		}
		cm.memberCache.Store(info.Wxid, info)
	}
}

// Close 清理缓存
func (cm *CacheUserManager) Close() {
	cm.friendCache.Range(func(key, value interface{}) bool {
		cm.friendCache.Delete(key)
		return true
	})
	cm.roomCache.Range(func(key, value interface{}) bool {
		cm.roomCache.Delete(key)
		return true
	})
	cm.memberCache.Range(func(key, value any) bool {
		cm.memberCache.Delete(key)
		return true
	})
	cm.friendCount = 0
	cm.roomCount = 0
	cm.memberCount = 0
}
