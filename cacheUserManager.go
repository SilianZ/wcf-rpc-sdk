// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/1/17 上午11:39:00
// @Desc 用户缓存器
package wcf_rpc_sdk

import (
	"github.com/Clov614/logging"
	"sync"
)

// ContactInfoManager 缓存管理器
type ContactInfoManager struct {
	contactInfoCache map[string]*ContactInfo
	ciMu             sync.RWMutex
}

// NewCacheInfoManager 创建缓存管理器
func NewCacheInfoManager() *ContactInfoManager {
	return &ContactInfoManager{
		contactInfoCache: make(map[string]*ContactInfo),
	}
}

func (cm *ContactInfoManager) CacheContactInfo(c *ContactInfo) bool {
	cm.ciMu.Lock()
	defer cm.ciMu.Unlock()
	if c == nil || c.Wxid == "" {
		return false
	}
	cm.contactInfoCache[c.Wxid] = c
	return true
}

func (cm *ContactInfoManager) GetContactInfo(id string) (*ContactInfo, bool) {
	cm.ciMu.RLock()
	defer cm.ciMu.RUnlock()
	c, ok := cm.contactInfoCache[id]
	return c, ok
}

// Close 清理缓存
func (cm *ContactInfoManager) Close() {
	cm.contactInfoCache = nil // 释放引用
	logging.Warn("【wcf】close user cache")
}
