// Package imgutil
// @Author Clover
// @Data 2024/7/22 下午1:53:00
// @Desc 图片处理工具类
package imgutil

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 创建一个全局的 http.Client，并配置为跳过 TLS 验证
var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func ImgFetch(path string) ([]byte, error) {
	if IsURL(path) {
		return fetchFromURL(path)
	}
	return fetchFromFile(path)
}

func IsURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// fetchFromURL fetches the content from the URL
func fetchFromURL(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetchFromURL: creating request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")

	resp, err := httpClient.Do(req) // 使用全局的 http.Client
	if err != nil {
		return nil, fmt.Errorf("fetchFromURL: http.Get(%q): %w", url, err)
	}
	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetchFromURL: resp.Body.ReadAll(): %w", err)
	}
	return bytes, nil
}

// fetchFromFile fetches the content from the file
func fetchFromFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("fetchFromFile: os.Open(%q): %w", filePath, err)
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("fetchFromFile: io.ReadAll(file): %w", err)
	}
	return bytes, nil
}

// FileType 表示文件类型的枚举
type FileType string

const (
	JPEG FileType = "jpg"
	PNG  FileType = "png"
	GIF  FileType = "gif"
	BMP  FileType = "bmp"
	TIFF FileType = "tiff"
	// 可以根据需要添加更多类型
)

// 图片文件头的签名信息
var imagePrefixBtsMap = map[FileType][]byte{
	JPEG: {0xFF, 0xD8, 0xFF},       // JPEG (jpg)，文件头：FFD8FF
	PNG:  {0x89, 0x50, 0x4E, 0x47}, // PNG (png)，文件头：89504E47
	GIF:  {0x47, 0x49, 0x46, 0x38}, // GIF (gif)，文件头：47494638
	TIFF: {0x49, 0x49, 0x2A, 0x00}, // TIFF (tif)，文件头：49492A00 (Little-endian TIFF)
	BMP:  {0x42, 0x4D},             // Windows Bitmap (bmp)，文件头：424D
}

var (
	ErrUnknowFileType = errors.New("unknown file type")
	ErrDecodeFail     = errors.New("decode fail") // 新增 decode fail 错误
)

// DetectFileType 检测文件的字节前缀以确定其类型
func DetectFileType(data []byte) (FileType, error) {
	for fileType, signatures := range imagePrefixBtsMap {
		if len(data) >= len(signatures) && bytes.Equal(data[:len(signatures)], signatures) {
			return fileType, nil
		}
	}
	return "", fmt.Errorf("detectFileType: %w", ErrUnknowFileType)
}

// GetMimeTypeByFileType 根据 FileType 返回 MIME 类型
func GetMimeTypeByFileType(fileType FileType) string {
	switch fileType {
	case JPEG:
		return "image/jpeg"
	case PNG:
		return "image/png"
	case GIF:
		return "image/gif"
	case BMP:
		return "image/bmp"
	case TIFF:
		return "image/tiff"
	default:
		return "application/octet-stream"
	}
}

// GetEtxByFileType 根据 FileType 返回 Ext 文件后缀
func GetEtxByFileType(fileType FileType) string {
	switch fileType {
	case JPEG:
		return ".jpg"
	case PNG:
		return ".png"
	case GIF:
		return ".gif"
	case BMP:
		return ".bmp"
	case TIFF:
		return ".tiff"
	default:
		return ""
	}
}

// decodeDatFileInternal 内部函数，解码微信 .dat 文件内容并写入 io.Writer
func decodeDatFileInternal(datFilePath string, writer io.Writer) error {
	sourceFile, err := os.Open(datFilePath)
	if err != nil {
		return fmt.Errorf("decodeDatFileInternal: open dat file: %w", err)
	}
	defer func() { _ = sourceFile.Close() }()

	preTenBts := make([]byte, 10)
	_, err = sourceFile.Read(preTenBts)
	if err != nil && err != io.EOF { // 忽略 EOF 错误，文件可能小于 10 字节
		return fmt.Errorf("decodeDatFileInternal: read prefix bytes: %w", err)
	}

	decodeByte, _, er := findDecodeByte(preTenBts) // 只需要 decodeByte，ext 在外部函数处理
	if er != nil {
		return fmt.Errorf("decodeDatFileInternal: %w", er) // 返回 findDecodeByte 的错误
	}

	_, err = sourceFile.Seek(0, io.SeekStart) // 移动到文件开头
	if err != nil {
		return fmt.Errorf("decodeDatFileInternal: seek file start: %w", err)
	}

	rBts := make([]byte, 1024)
	bufWriter := bufio.NewWriter(writer) // 使用 bufio.Writer 提高效率
	defer func() { _ = bufWriter.Flush() }()

	for {
		n, err := sourceFile.Read(rBts)
		if err != nil {
			if err == io.EOF {
				break // 文件结束，正常退出循环
			}
			return fmt.Errorf("decodeDatFileInternal: read file content: %w", err)
		}
		for i := 0; i < n; i++ {
			if err := bufWriter.WriteByte(rBts[i] ^ decodeByte); err != nil {
				return fmt.Errorf("decodeDatFileInternal: write decoded byte: %w", err)
			}
		}
	}
	return nil
}

// DecodeDatFile 解码微信 .dat 文件为图片, 并保存到指定目录
// datFilePath: .dat 文件路径
// outputDir:  输出目录
func DecodeDatFile(datFilePath, outputDir string) error {
	// 检查输出目录是否存在，不存在则创建
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("DecodeDatFile: create output dir: %w", err)
		}
	}

	// 使用新的检查函数，设置重试次数为 5
	if err := checkDatFileExists(datFilePath, 5); err != nil {
		return fmt.Errorf("DecodeDatFile: %w", err)
	}

	// 获取文件名用于输出路径，因为 checkDatFileExists 已经确认文件存在
	fileName := filepath.Base(datFilePath)

	preTenBts := make([]byte, 10) //  提前声明 preTenBts
	sourceFile, err := os.Open(datFilePath)
	if err != nil {
		return fmt.Errorf("DecodeDatFile: open dat file: %w", err)
	}
	defer sourceFile.Close()
	_, err = sourceFile.Read(preTenBts) //  读取 preTenBts
	if err != nil && err != io.EOF {
		return fmt.Errorf("DecodeDatFile: read prefix bytes: %w", err)
	}
	_, ext, er := findDecodeByte(preTenBts) //  只需要 ext
	if er != nil {
		return fmt.Errorf("DecodeDatFile: %w", er)
	}
	if ext == "" {
		return errors.New("DecodeDatFile: file extension not found")
	}

	// 使用从 datFilePath 获取的文件名构建输出路径
	outputFilePath := filepath.Join(outputDir, strings.TrimSuffix(fileName, filepath.Ext(fileName))+ext)
	distFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("DecodeDatFile: create output file: %w", err)
	}
	defer distFile.Close()

	if err := decodeDatFileInternal(datFilePath, distFile); err != nil { // 调用内部函数
		// 如果解码失败，尝试删除已创建的空输出文件
		_ = distFile.Close()          // 确保文件已关闭
		_ = os.Remove(outputFilePath) // 尝试删除，忽略错误
		return fmt.Errorf("DecodeDatFile: decodeDatFileInternal: %w", err)
	}

	fmt.Println("DecodeDatFile: output file：", distFile.Name()) // 保留输出信息
	return nil
}

// DecodeDatFileToBytes 解码微信 .dat 文件为图片, 并返回字节数组
// datFilePath: .dat 文件路径
func DecodeDatFileToBytes(datFilePath string) ([]byte, error) {
	// 使用新的检查函数，设置重试次数为 5
	if err := checkDatFileExists(datFilePath, 5); err != nil {
		//logging.ErrorWithErr(err, "DecodeDatFile .dat was not exist") // 可以保留日志记录
		return nil, fmt.Errorf("DecodeDatFileToBytes: %w", err)
	}

	preTenBts := make([]byte, 10) // 提前声明 preTenBts
	sourceFile, err := os.Open(datFilePath)
	if err != nil {
		return nil, fmt.Errorf("DecodeDatFileToBytes: open dat file: %w", err)
	}
	defer func() { _ = sourceFile.Close() }()
	_, err = sourceFile.Read(preTenBts) // 读取 preTenBts
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("DecodeDatFileToBytes: read prefix bytes: %w", err)
	}
	_, ext, er := findDecodeByte(preTenBts) // 只需要 ext
	if er != nil {
		return nil, fmt.Errorf("DecodeDatFileToBytes: %w", er)
	}
	if ext == "" {
		return nil, errors.New("DecodeDatFileToBytes: file extension not found")
	}

	var decodedData bytes.Buffer                                             // 使用 bytes.Buffer 存储解码后的数据
	if err := decodeDatFileInternal(datFilePath, &decodedData); err != nil { // 调用内部函数，传入 bytes.Buffer
		return nil, fmt.Errorf("DecodeDatFileToBytes: decodeDatFileInternal: %w", err)
	}

	fmt.Println("DecodeDatFileToBytes: decode file success") // 保留输出信息
	return decodedData.Bytes(), nil                          // 返回解码后的字节数组
}

func findDecodeByte(bts []byte) (byte, string, error) {
	for fileType, prefixBytes := range imagePrefixBtsMap {
		deCodeByte, err := testPrefix(prefixBytes, bts)
		if err == nil {
			etx := GetEtxByFileType(fileType) // 使用 GetEtxByFileType 获取扩展名
			if etx == "" {
				return 0, "", fmt.Errorf("findDecodeByte: no extension for file type: %s", fileType)
			}
			return deCodeByte, etx, nil
		}
	}
	return 0, "", ErrDecodeFail // 使用预定义的 ErrDecodeFail
}

func testPrefix(prefixBytes []byte, bts []byte) (deCodeByte byte, error error) {
	if len(bts) < len(prefixBytes) {
		return 0, errors.New("testPrefix: data too short to match prefix") // 数据太短，无法匹配前缀
	}
	if len(bts) == 0 || len(prefixBytes) == 0 {
		return 0, errors.New("testPrefix: empty input data or prefix") //  数据或前缀为空
	}

	initDecodeByte := prefixBytes[0] ^ bts[0]
	for i, prefixByte := range prefixBytes {
		if b := prefixByte ^ bts[i]; b != initDecodeByte {
			return 0, errors.New("testPrefix: byte mismatch") // 字节不匹配
		}
	}
	return initDecodeByte, nil
}

// CreateTempFile 创建临时文件
func CreateTempFile(ext string) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", "tempimage-*"+ext)
	if err != nil {
		return nil, fmt.Errorf("CreateTempFile: os.CreateTemp: %w", err)
	}
	return tmpFile, nil
}

// RemoveTempFile 移除临时文件
func RemoveTempFile(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("RemoveTempFile: os.Remove: %w", err)
	}
	return nil
}

// checkDatFileExists 检查 .dat 文件是否存在，并包含重试逻辑
func checkDatFileExists(datFilePath string, retries int) error {
	var err error
	for i := 0; i < retries; i++ {
		info, err := os.Stat(datFilePath)
		if err == nil {
			// 检查是否是目录或扩展名不是 .dat
			if info.IsDir() || filepath.Ext(info.Name()) != ".dat" {
				return fmt.Errorf("invalid dat file path: %s", datFilePath) // 返回更具体的错误信息
			}
			return nil // 文件有效，返回 nil
		}
		if !os.IsNotExist(err) {
			// 如果错误不是 "文件不存在"，则直接返回错误
			return fmt.Errorf("stat dat file: %w", err)
		}
		// 文件不存在，等待后重试
		if i < retries-1 { // 避免最后一次重试后还等待
			time.Sleep(2 * time.Second)
		}
	}
	// 重试次数用尽后仍然失败
	return fmt.Errorf(".dat file was not exist after %d retries: %w", retries, err)
}
