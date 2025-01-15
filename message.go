// Package wcf_rpc_sdk
// @Author Clover
// @Data 2025/1/13 下午8:48:00
// @Desc
package wcf_rpc_sdk

import "wcf-rpc-sdk/internal/manager"

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
}
