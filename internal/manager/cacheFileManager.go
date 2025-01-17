// Package manager
// @Author Clover
// @Date 2025/1/8 下午4:13:00
// @Desc Cache Manager for handling files
package manager

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/Clov614/wcf-rpc-sdk/internal/utils"
	"github.com/Clov614/wcf-rpc-sdk/internal/utils/base64util"
	"github.com/Clov614/wcf-rpc-sdk/logging"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrInvalidDate = errors.New("invalid date")
	ErrFileExists  = errors.New("file already exists")
)

type ICacheFileManager interface {
	Save(fileName string, isImg bool, data []byte) (*FileInfo, error)
	GetFilePathByFileName(fileName string) (string, error)
	GetDataByFileName(fileName string) ([]byte, error)
	GetFileInfoBase64(fileName string) (*FileInfo, error)
}

type FileName2FileInfo map[string]*FileInfo // FileName-to-FileInfo mapping

type FileInfo struct {
	FilePath string // Full file path
	FileName string // File name including extension
	FileExt  string // File extension
	IsImg    bool   // Indicates if the file is an image
	Base64   string // 可选，非必须
}

// CacheFileManager is the implementation of ICacheFileManager
type CacheFileManager struct {
	mu                sync.RWMutex
	fileName2FileInfo FileName2FileInfo
	cache             *list.List // LRU 缓存列表
	capacity          int        // 缓存容量
}

var (
	cacheManager     *CacheFileManager
	cacheManagerOnce sync.Once
)

// newCacheManager creates a new CacheFileManager with the given cache size.
func newCacheManager(cacheSize int) *CacheFileManager {
	return &CacheFileManager{
		fileName2FileInfo: make(FileName2FileInfo, cacheSize),
		cache:             list.New(),
		capacity:          cacheSize,
	}
}

// GetCacheManager returns the singleton instance of CacheFileManager.
func GetCacheManager() ICacheFileManager {
	cacheManagerOnce.Do(func() {
		cacheManager = newCacheManager(30)
	})
	return cacheManager
}

// Save saves a file by its fileName and writes data to the file system.
func (cm *CacheFileManager) Save(fileName string, isImg bool, data []byte) (*FileInfo, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if the file already exists
	if _, exists := cm.fileName2FileInfo[fileName]; exists {
		return nil, fmt.Errorf("%w: %s", ErrFileExists, fileName)
	}

	// Generate file path
	filePath, err := utils.ConvertToWindows(fileName, isImg)
	if err != nil {
		return nil, fmt.Errorf("convert to windows file failed: %w", err)
	}

	// Write data to the file system
	if err := writeDataToFile(filePath, data); err != nil {
		return nil, fmt.Errorf("failed to write data to file: %w", err)
	}

	// Create and store FileInfo
	fileInfo := &FileInfo{
		FilePath: filePath,
		FileName: fileName,
		FileExt:  getFileExtension(fileName),
		IsImg:    isImg,
	}
	cm.fileName2FileInfo[fileName] = fileInfo
	cm.addToCache(fileName)

	return fileInfo, nil
}

// GetFilePathByFileName retrieves the file path by its file name.
func (cm *CacheFileManager) GetFilePathByFileName(fileName string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	fileInfo, exists := cm.fileName2FileInfo[fileName]
	if !exists {
		return "", os.ErrNotExist
	}
	cm.moveToFront(fileName)
	return fileInfo.FilePath, nil
}

// GetDataByFileName retrieves the file data by its file name.
func (cm *CacheFileManager) GetDataByFileName(fileName string) ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	fileInfo, exists := cm.fileName2FileInfo[fileName]
	if !exists {
		return nil, os.ErrNotExist
	}

	// Read the data from the file
	data, err := os.ReadFile(fileInfo.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	cm.moveToFront(fileName)
	return data, nil
}

// GetFileInfoBase64 retrieves the FileInfo and the base64 encoded data of the file by its file name.
func (cm *CacheFileManager) GetFileInfoBase64(fileName string) (*FileInfo, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	fileInfo, exists := cm.fileName2FileInfo[fileName]
	if !exists {
		return nil, os.ErrNotExist
	}

	// Read the data from the file
	data, err := os.ReadFile(fileInfo.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// Encode the data to base64
	fileInfo.Base64 = base64util.EncodeBase64(data)

	cm.moveToFront(fileName)
	return fileInfo, nil
}

// writeDataToFile writes data to the given file path.
func writeDataToFile(filePath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write data to file: %w", err)
	}

	return nil
}

// getFileExtension extracts the file extension from the file name.
func getFileExtension(fileName string) string {
	if i := strings.LastIndex(fileName, "."); i >= 0 {
		return fileName[i+1:]
	}
	return ""
}

// addToCache adds a fileName to the cache.
func (cm *CacheFileManager) addToCache(fileName string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.cache.Len() >= cm.capacity {
		// Remove the least recently used element
		back := cm.cache.Back()
		if back != nil {
			// 获取需要删除的文件名
			fileNameToRemove := back.Value.(string)

			// 删除对应的文件信息
			fileInfoToRemove, exists := cm.fileName2FileInfo[fileNameToRemove]
			if exists {
				// 删除系统文件
				if err := os.Remove(fileInfoToRemove.FilePath); err != nil {
					// 处理删除文件失败的错误，例如记录日志
					logging.Error(fmt.Sprintf("failed to remove file: %s, error: %v\n", fileInfoToRemove.FilePath, err))
				}
			}

			// 从缓存映射和链表中删除
			delete(cm.fileName2FileInfo, fileNameToRemove)
			cm.cache.Remove(back)
		}
	}
	cm.cache.PushFront(fileName)
}

// moveToFront moves a fileName to the front of the cache.
func (cm *CacheFileManager) moveToFront(fileName string) {
	for e := cm.cache.Front(); e != nil; e = e.Next() {
		if e.Value.(string) == fileName {
			cm.cache.MoveToFront(e)
			return
		}
	}
}

// Close cleans up the cache files and releases resources.
func (cm *CacheFileManager) Close() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Iterate through all cached files and remove them
	for fileName, fileInfo := range cm.fileName2FileInfo {
		if err := os.Remove(fileInfo.FilePath); err != nil {
			logging.Error(fmt.Sprintf("failed to remove file: %s, error: %v\n", fileInfo.FilePath, err))
		}
		delete(cm.fileName2FileInfo, fileName)
	}

	// Clear the cache list
	cm.cache.Init()

	logging.Info("清除缓存并删除临时文件成功")
}
