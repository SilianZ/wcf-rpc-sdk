package wcf_rpc_sdk

import (
	"testing"
)

func TestFileInfo_ExtractRelativePath(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected string
	}{
		{
			name:     "Path with MsgAttach",
			filePath: "C:/Users/Administrator/Documents/WeChat Files/wxid_p5z4fuhnbdgs22/FileStorage/MsgAttach/84d8449549662bc200b18aabcf977f3a/Image/2025-02/010ff5751d7a461e3a98fe27fd5df1ab.dat",
			expected: "/84d8449549662bc200b18aabcf977f3a/Image/2025-02/010ff5751d7a461e3a98fe27fd5df1ab.dat",
		},
		{
			name:     "Path without MsgAttach",
			filePath: "/path/without/MsgAttach/file.doc",
			expected: "/file.doc",
		},
		{
			name:     "Path with MsgAttach at the end",
			filePath: "/path/to/MsgAttach",
			expected: "/",
		},
		{
			name:     "Path with MsgAttach but empty after",
			filePath: "/path/to/MsgAttach/",
			expected: "/",
		},
		{
			name:     "Empty path",
			filePath: "",
			expected: "",
		},
		{
			name:     "Path with MsgAttach and double slashes",
			filePath: "/path//MsgAttach//double//slashes/file.txt",
			expected: "/double/slashes/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fi := &FileInfo{FilePath: tt.filePath}
			fi.ExtractRelativePath()
			if fi.RelativePathAfterMsgAttach != tt.expected {
				t.Errorf("ExtractRelativePath() got = %v, want %v", fi.RelativePathAfterMsgAttach, tt.expected)
			}
		})
	}
}
