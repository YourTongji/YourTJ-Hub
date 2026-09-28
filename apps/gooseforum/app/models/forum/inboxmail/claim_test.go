package inboxmail

import "testing"

// 逐 claim 的状态分类必须互斥且完整：#778 的领取入口用成功集合跳过已完成项，
// 重试扫描用可重试集合捞起未完成项；把 failed 当终态会让失败项永远不被重试。
func TestClaimStatusPredicates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status    string
		success   bool
		retryable bool
	}{
		{status: ClaimStatusPending, success: false, retryable: true},
		{status: ClaimStatusGranted, success: true, retryable: false},
		{status: ClaimStatusAlreadyOwned, success: true, retryable: false},
		{status: ClaimStatusFailed, success: false, retryable: true},
		{status: ClaimStatusExpired, success: false, retryable: true},
		{status: "", success: false, retryable: false},
		{status: "unknown", success: false, retryable: false},
	}
	for _, tc := range cases {
		name := tc.status
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := IsSuccessfulClaimStatus(tc.status); got != tc.success {
				t.Fatalf("IsSuccessfulClaimStatus(%q) = %v, want %v", tc.status, got, tc.success)
			}
			if got := IsRetryableClaimStatus(tc.status); got != tc.retryable {
				t.Fatalf("IsRetryableClaimStatus(%q) = %v, want %v", tc.status, got, tc.retryable)
			}
		})
	}
}

// 每个已声明的状态必须恰好属于「成功」或「可重试」之一；将来新增状态（例如 #778
// 的结算态）若忘了分类，这里会失败，避免它被两个扫描同时漏掉或重复处理。
func TestClaimStatusVocabularyIsClassified(t *testing.T) {
	t.Parallel()
	for _, status := range []string{
		ClaimStatusPending,
		ClaimStatusGranted,
		ClaimStatusAlreadyOwned,
		ClaimStatusFailed,
		ClaimStatusExpired,
	} {
		success := IsSuccessfulClaimStatus(status)
		retryable := IsRetryableClaimStatus(status)
		if success == retryable {
			t.Fatalf("status %q must be exactly one of successful/retryable (success=%v retryable=%v)", status, success, retryable)
		}
	}
}
