// Package utils
// @Author Clover
// @Data 2025/1/8 下午4:05:00
// @Desc
package utils

import (
	"fmt"
	"github.com/eatmoreapple/env"
	"os"
	"path/filepath"
)

// TempDir for go:linkname
func TempDir() string {
	return env.Name("TEMP_DIR").StringOrElse(os.TempDir())
}

// ConvertToWindows 根据给定的文件名生成一个在 Windows 操作系统中安全的文件路径。
// 它将一个可选的子目录（'img' 或 'file'）添加到临时目录中。
//
// 参数：
// fileName: 要转换的文件名。
// isImg: 一个布尔值，指示文件是否为图像。如果为 true，则将 'img' 子目录添加到路径中；否则，添加 'file' 子目录。
//
// 返回值：
// string: 转换后的 Windows 文件路径。
// error: 如果获取临时目录或创建目录时出错，则返回错误。
func ConvertToWindows(fileName string, isImg bool) (string, error) {
	// 根据 isImg 参数确定子目录
	var subDir string
	if isImg {
		subDir = "img"
	} else {
		subDir = "file"
	}

	// 构建完整路径
	dir := filepath.Join(TempDir(), subDir)

	// 确保目录存在
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// 返回完整路径
	return filepath.Join(dir, fileName), nil
}
