package badgeservice

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const badgeAssetsDir = "../../../resource/static/badges"

// TestBadgeAssetsHashMatches 是 badgeAssetVersion 的机械守卫：改动
// static/badges 下任何 SVG 都会改变目录哈希，使本测试失败。确认要改图形时，
// 先递增 definitions.go 里的 badgeAssetVersion（让老用户的 /static 缓存失效），
// 再把错误信息里打印的新哈希抄进 badgeAssetHash。
func TestBadgeAssetsHashMatches(t *testing.T) {
	entries, err := os.ReadDir(badgeAssetsDir)
	if err != nil {
		t.Fatalf("读取徽章图形目录 %s: %v", badgeAssetsDir, err)
	}
	h := sha256.New()
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".svg") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(badgeAssetsDir, entry.Name()))
		if err != nil {
			t.Fatalf("读取 %s: %v", entry.Name(), err)
		}
		// 归一化掉 \r，避免 Windows 检出 CRLF 时哈希与 Linux 不一致
		h.Write([]byte(entry.Name()))
		h.Write([]byte{0})
		h.Write([]byte(strings.ReplaceAll(string(content), "\r", "")))
		h.Write([]byte{0})
		count++
	}
	if count == 0 {
		t.Fatalf("%s 下没有 SVG 文件", badgeAssetsDir)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != badgeAssetHash {
		t.Errorf("徽章图形有改动但 badgeAssetVersion 未递增？目录哈希 = %s，记录值 = %s。\n若确认要改图形：递增 badgeAssetVersion，并把 badgeAssetHash 更新为 %s", got, badgeAssetHash, got)
	}
}
