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
)

// CacheUserManager 缓存管理器
type CacheUserManager struct {
	friendCache *sync.Map // 好友信息缓存，key: wxid, value: *Friend
	roomCache   *sync.Map // 群组信息缓存，key: room_id, value: *ChatRoom
	friendCount int       // 好友数量计数器
	roomCount   int       // 群组数量计数器
	// todo 公众号
}

// NewCacheInfoManager 创建缓存管理器
func NewCacheInfoManager() *CacheUserManager {
	return &CacheUserManager{
		friendCache: &sync.Map{},
		roomCache:   &sync.Map{},
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
	if _, ok := cm.roomCache.Load(chatRoom.Wxid); !ok {
		cm.roomCount++
	}
	cm.roomCache.Store(chatRoom.Wxid, chatRoom)
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
	cm.friendCount = 0
	cm.roomCount = 0
}
