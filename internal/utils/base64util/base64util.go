// Package base64util
// @Author Clover
// @Date 2025/1/9 下午3:00:00
// @Desc Base64 工具类
package base64util

import (
	"encoding/base64"
	"fmt"
)

// EncodeBase64 encodes data to base64 string.
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 string to data.
func DecodeBase64(base64Str string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("DecodeBase64: %w", err)
	}
	return data, nil
}
