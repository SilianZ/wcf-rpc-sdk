package manager

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
	"wcf-rpc-sdk/internal/utils"
	"wcf-rpc-sdk/internal/utils/base64util"
)

var i int = 0

// 创建临时目录并设置环境变量 TEMP_DIR
// 每次生成唯一的临时目录名
func setupTempDir(t *testing.T) string {
	i++
	t.Helper()
	// 生成唯一的临时目录名
	tempDir := filepath.Join(t.TempDir(), fmt.Sprintf("testdata_%s_%d", time.Now().Format("20060102_150405"), i))
	t.Logf("tempDir: %s", tempDir)
	// 创建目录
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	os.Setenv("TEMP_DIR", tempDir)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	return tempDir
}

// TestCacheManagerSave 测试 Save 方法
func TestCacheManagerSave(t *testing.T) {
	manager := GetCacheManager()
	_ = setupTempDir(t)

	// 测试保存文件
	fileName := "testfile1.txt"
	fileData := []byte("test data")
	fileInfo, err := manager.Save(fileName, false, fileData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 显式生成期望的文件路径
	expectedFilePath, err := utils.ConvertToWindows(fileName, false)
	if err != nil {
		t.Fatalf("ConvertToWindows failed: %v", err)
	}

	// 验证 FileInfo
	if fileInfo.FileName != fileName {
		t.Errorf("expected fileName %s, got %s", fileName, fileInfo.FileName)
	}
	if fileInfo.FileExt != "txt" {
		t.Errorf("expected fileExt txt, got %s", fileInfo.FileExt)
	}
	if fileInfo.FilePath != expectedFilePath {
		t.Errorf("expected filePath %s, got %s", expectedFilePath, fileInfo.FilePath)
	}
}

// TestCacheManagerGetData 测试 GetDataByFileName 方法
func TestCacheManagerGetData(t *testing.T) {
	manager := GetCacheManager()
	_ = setupTempDir(t)

	// 保存测试文件
	fileName := "testfile2.txt"
	fileData := []byte("Hello, world!")
	_, err := manager.Save(fileName, false, fileData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 测试读取数据
	data, err := manager.GetDataByFileName(fileName)
	if err != nil {
		t.Fatalf("GetDataByFileName failed: %v", err)
	}
	expectedData := "Hello, world!"
	if string(data) != expectedData {
		t.Errorf("expected data %s, got %s", expectedData, string(data))
	}
}

// TestCacheManagerFilePath 测试 GetFilePathByFileName 方法
func TestCacheManagerFilePath(t *testing.T) {
	// 设置环境变量 TEMP_DIR，用于指定测试数据的根目录
	tempDir := setupTempDir(t)
	testDataDir := filepath.Join(tempDir)
	err := os.Setenv("TEMP_DIR", testDataDir)
	if err != nil {
		t.Fatalf("Failed to set TEMP_DIR: %v", err)
	}

	manager := GetCacheManager()

	// 保存测试文件 - 图片
	imgFileName := "testimage.jpg"
	imgFileData := []byte("test image data")
	_, err = manager.Save(imgFileName, true, imgFileData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 测试获取图片文件路径
	imgFilePath, err := manager.GetFilePathByFileName(imgFileName)
	if err != nil {
		t.Fatalf("GetFilePathByFileName failed: %v", err)
	}
	expectedImgPath := filepath.Join(os.Getenv("TEMP_DIR"), "img", imgFileName) // 使用 os.Getenv("TEMP_DIR")
	if imgFilePath != expectedImgPath {
		t.Errorf("expected filePath %s, got %s", expectedImgPath, imgFilePath)
	}

	// 保存测试文件 - 非图片
	fileFileName := "testfile.txt"
	fileFileData := []byte("test file data")
	_, err = manager.Save(fileFileName, false, fileFileData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 测试获取非图片文件路径
	filePath, err := manager.GetFilePathByFileName(fileFileName)
	if err != nil {
		t.Fatalf("GetFilePathByFileName failed: %v", err)
	}
	expectedFilePath := filepath.Join(os.Getenv("TEMP_DIR"), "file", fileFileName) // 使用 os.Getenv("TEMP_DIR")
	if filePath != expectedFilePath {
		t.Errorf("expected filePath %s, got %s", expectedFilePath, filePath)
	}
}

func TestTempDir(t *testing.T) {
	// 测试环境变量 TEMP_DIR 未设置的情况
	os.Unsetenv("TEMP_DIR")
	tempDir := utils.TempDir()
	if tempDir != os.TempDir() {
		t.Errorf("expected %s, got %s", os.TempDir(), tempDir)
	}

	// 测试环境变量 TEMP_DIR 已设置的情况
	expectedTempDir := filepath.Join(os.TempDir(), "testtempdir")
	os.Setenv("TEMP_DIR", expectedTempDir)
	defer os.Unsetenv("TEMP_DIR")

	tempDir = utils.TempDir()
	if tempDir != expectedTempDir {
		t.Errorf("expected %s, got %s", expectedTempDir, tempDir)
	}
}

// TestLRU 缓存测试
func TestLRUCache(t *testing.T) {
	// 设置缓存大小为 3
	manager := newCacheManager(3)
	_ = setupTempDir(t)

	// 保存 4 个文件，预期最早保存的文件会被淘汰
	fileNames := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt"}
	for _, fileName := range fileNames {
		_, err := manager.Save(fileName, false, []byte(fileName))
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	// 验证 file1.txt 是否被淘汰
	_, err := manager.GetFilePathByFileName("file1.txt")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}

	// 验证 file2.txt, file3.txt, file4.txt 是否存在
	for _, fileName := range fileNames[1:] {
		_, err := manager.GetFilePathByFileName(fileName)
		if err != nil {
			t.Errorf("GetFilePathByFileName failed: %v", err)
		}
	}

	// 访问 file2.txt，使其成为最近使用的文件
	_, err = manager.GetFileInfoBase64("file2.txt")
	if err != nil {
		t.Fatalf("GetFileInfoBase64 failed: %v", err)
	}

	// 保存 file5.txt，预期 file3.txt 会被淘汰
	_, err = manager.Save("file5.txt", false, []byte("file5.txt"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 验证 file3.txt 是否被淘汰
	_, err = manager.GetFilePathByFileName("file3.txt")
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}

	// 验证 file2.txt, file4.txt, file5.txt 是否存在
	for _, fileName := range []string{"file2.txt", "file4.txt", "file5.txt"} {
		_, err := manager.GetFilePathByFileName(fileName)
		if err != nil {
			t.Errorf("GetFilePathByFileName failed: %v", err)
		}
	}
}

// TestCacheManagerGetFileInfoBase64 测试 GetFileInfoBase64 方法
func TestCacheManagerGetFileInfoBase64(t *testing.T) {
	manager := GetCacheManager()
	_ = setupTempDir(t)

	// 保存测试文件
	fileName := "testfile3.txt"
	fileData := []byte("Hello, Base64!")
	_, err := manager.Save(fileName, false, fileData)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 测试获取 FileInfo 和 Base64 数据
	fileInfo, err := manager.GetFileInfoBase64(fileName)
	if err != nil {
		t.Fatalf("GetFileInfoBase64 failed: %v", err)
	}

	// 验证 FileInfo
	expectedFilePath, err := utils.ConvertToWindows(fileName, false)
	if err != nil {
		t.Fatalf("ConvertToWindows failed: %v", err)
	}
	if fileInfo.FilePath != expectedFilePath {
		t.Errorf("expected filePath %s, got %s", expectedFilePath, fileInfo.FilePath)
	}

	// 验证 Base64 数据
	expectedBase64Str := base64util.EncodeBase64(fileData)
	if fileInfo.Base64 != expectedBase64Str {
		t.Errorf("expected base64 %s, got %s", expectedBase64Str, fileInfo.Base64)
	}
}
