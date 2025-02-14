// Package imgutil
// @Author Clover
// @Data 2024/7/30 下午11:39:00
// @Desc 图片工具测试
package imgutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectImgType(t *testing.T) {
	data, err := fetchFromURL("https://www.freeimg.cn/i/2024/04/22/66260f2eed1d6.jpg")
	if err != nil {
		t.Error(err)
	}

	fileType, err := DetectFileType(data)
	if err != nil {
		t.Error(err)
	}
	t.Log(fileType)

	data2, err := fetchFromURL("https://www.freeimg.cn/i/2024/04/22/66260f0ae65a3.png")
	if err != nil {
		t.Error(err)
	}

	fileType2, err := DetectFileType(data2)
	if err != nil {
		t.Error(err)
	}
	t.Log(fileType2)
}

func TestDecodeDatFile(t *testing.T) {
	// 1. 创建一个临时的测试目录
	outputDir := filepath.Join(os.TempDir(), "imgutil_test_output")
	defer os.RemoveAll(outputDir) // 测试结束后清理

	testCases := []struct {
		name         string
		sourceURL    string //  这里 sourceURL  实际上是本地文件路径了
		encryptByte  byte
		expectedType FileType
		expectedExt  string
	}{
		{
			name:         "PNGTest",
			sourceURL:    filepath.Join("./source_test", "test.png"), // 本地 png 图片路径
			encryptByte:  byte(0x55),
			expectedType: PNG,
			expectedExt:  ".png",
		},
		//{
		//	name:          "JPEGTest",
		//	sourceURL:     filepath.Join("./source_test", "test.jpg"), // 本地 jpg 图片路径
		//	encryptByte:   byte(0x55),
		//	expectedType:  JPEG,
		//	expectedExt:   ".jpg",
		//},
		//{
		//	name:          "GIFTest",
		//	sourceURL:     filepath.Join("./source_test", "test.gif"), // 本地 gif 图片路径
		//	encryptByte:   byte(0x55),
		//	expectedType:  GIF,
		//	expectedExt:   ".gif",
		//},
		//{
		//	name:          "BMPTest",
		//	sourceURL:     filepath.Join("./source_test", "test.bmp"), // 本地 bmp 图片路径
		//	encryptByte:   byte(0x55),
		//	expectedType:  BMP,
		//	expectedExt:   ".bmp",
		//},
		//{
		//	name:          "TIFFTest",
		//	sourceURL:     filepath.Join("./source_test", "test.tiff"), // 本地 tiff 图片路径
		//	encryptByte:   byte(0x55),
		//	expectedType:  TIFF,
		//	expectedExt:   ".tiff",
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 2. 创建一个模拟的 .dat 文件
			datFileName := "test" + tc.expectedExt + ".dat" // 文件名带上原始扩展名信息，方便查看
			datFilePath := filepath.Join(os.TempDir(), datFileName)
			defer os.Remove(datFilePath)

			originalData, err := os.ReadFile(tc.sourceURL) // 从本地文件读取
			if err != nil {
				t.Fatalf("[%s] ReadFile failed: %v", tc.name, err) // 修改错误信息
			}

			// 假设解密 byte 是 0x55，对图片数据进行加密模拟 .dat 文件内容
			encryptByte := byte(0x55)
			var datContent []byte
			for _, b := range originalData {
				datContent = append(datContent, b^encryptByte)
			}

			err = os.WriteFile(datFilePath, datContent, 0644)
			if err != nil {
				t.Fatalf("[%s] WriteFile failed: %v", tc.name, err)
			}

			// 3. 调用 DecodeDatFile 函数进行解码
			err = DecodeDatFile(datFilePath, outputDir)
			if err != nil {
				t.Fatalf("[%s] DecodeDatFile failed: %v", tc.name, err)
			}

			// 4. 验证解码结果
			outputFilePath := filepath.Join(outputDir, datFileName+tc.expectedExt) // 预期的输出文件名
			decodedData, err := os.ReadFile(outputFilePath)
			if err != nil {
				t.Fatalf("[%s] ReadFile decoded file failed: %v", tc.name, err)
			}

			fileType, err := DetectFileType(decodedData)
			if err != nil {
				t.Fatalf("[%s] DetectFileType failed: %v", tc.name, err)
			}

			if fileType != tc.expectedType {
				t.Errorf("[%s] Expected file type to be %v, but got %v", tc.name, tc.expectedType, fileType)
			}

			// 可以增加更严格的验证，例如比较解码后的数据和原始数据 (如果需要完全一致，可能需要处理文件格式的差异)
			t.Logf("[%s] Dat file decoded successfully to: %s, file type: %s", tc.name, outputFilePath, fileType)
		})
	}
}

func TestDecodeDatFileToBytes(t *testing.T) {
	// 1. 创建一个临时的测试目录 (这里其实不需要目录，为了代码结构统一保留)
	outputDir := filepath.Join(os.TempDir(), "imgutil_test_output")
	defer os.RemoveAll(outputDir) // 测试结束后清理

	testCases := []struct {
		name         string
		sourceURL    string //  这里 sourceURL  实际上是本地文件路径了
		encryptByte  byte
		expectedType FileType
		expectedExt  string
	}{
		{
			name:         "PNGTest",
			sourceURL:    filepath.Join("./source_test", "test.png"), // 本地 png 图片路径
			encryptByte:  byte(0x55),
			expectedType: PNG,
			expectedExt:  ".png",
		},
		//{
		//	name:          "JPEGTest",
		//	sourceURL:     filepath.Join("./source_test", "test.jpg"), // 本地 jpg 图片路径
		//	encryptByte:   byte(0x55),
		//	expectedType:  JPEG,
		//	expectedExt:   ".jpg",
		//},
		//{
		//	name:          "GIFTest",
		//	sourceURL:     filepath.Join("./source_test", "test.gif"), // 本地 gif 图片路径
		//	encryptByte:   byte(0x55),
		//	expectedType:  GIF,
		//	expectedExt:   ".gif",
		//},
		//{
		//	name:          "BMPTest",
		//	sourceURL:     filepath.Join("./source_test", "test.bmp"), // 本地 bmp 图片路径
		//	encryptByte:   byte(0x55),
		//	expectedType:  BMP,
		//	expectedExt:   ".bmp",
		//},
		//{
		//	name:          "TIFFTest",
		//	sourceURL:     filepath.Join("./source_test", "test.tiff"), // 本地 tiff 图片路径
		//	encryptByte:   byte(0x55),
		//	expectedType:  TIFF,
		//	expectedExt:   ".tiff",
		//},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 2. 创建一个模拟的 .dat 文件
			datFileName := "test" + tc.expectedExt + ".dat" // 文件名带上原始扩展名信息，方便查看
			datFilePath := filepath.Join(os.TempDir(), datFileName)
			defer os.Remove(datFilePath)

			originalData, err := os.ReadFile(tc.sourceURL) // 从本地文件读取
			if err != nil {
				t.Fatalf("[%s] ReadFile failed: %v", tc.name, err) // 修改错误信息
			}

			// 假设解密 byte 是 0x55，对图片数据进行加密模拟 .dat 文件内容
			encryptByte := byte(0x55)
			var datContent []byte
			for _, b := range originalData {
				datContent = append(datContent, b^encryptByte)
			}

			err = os.WriteFile(datFilePath, datContent, 0644)
			if err != nil {
				t.Fatalf("[%s] WriteFile failed: %v", tc.name, err)
			}

			// 3. 调用 DecodeDatFileToBytes 函数进行解码
			decodedDataBytes, err := DecodeDatFileToBytes(datFilePath) // 调用新的 DecodeDatFileToBytes 函数
			if err != nil {
				t.Fatalf("[%s] DecodeDatFileToBytes failed: %v", tc.name, err)
			}

			// 4. 验证解码结果 (直接使用 byte 数组)
			fileType, err := DetectFileType(decodedDataBytes) // 使用解码后的 byte 数组
			if err != nil {
				t.Fatalf("[%s] DetectFileType failed: %v", tc.name, err)
			}

			if fileType != tc.expectedType {
				t.Errorf("[%s] Expected file type to be %v, but got %v", tc.name, tc.expectedType, fileType)
			}

			// 可以增加更严格的验证，例如比较解码后的数据和原始数据 (如果需要完全一致，可能需要处理文件格式的差异)
			t.Logf("[%s] Dat file decoded to bytes successfully, file type: %s", tc.name, fileType) // 修改日志信息
		})
	}
}
