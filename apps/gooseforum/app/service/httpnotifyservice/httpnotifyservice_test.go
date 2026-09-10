package httpnotifyservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func TestEndpointAcceptsSelectedEvent(t *testing.T) {
	endpoint := pageConfig.HttpNotifyEndpoint{Enabled: true, URL: "http://example.com/hook", Events: []string{"topic.published"}}

	if !endpointAccepts(endpoint, "topic.published") {
		t.Fatal("expected endpoint to accept selected event")
	}
	if endpointAccepts(endpoint, "comment.created") {
		t.Fatal("expected endpoint to reject unselected event")
	}
}

func TestShouldNotifyConfig(t *testing.T) {
	config := pageConfig.HttpNotifyConfig{Enabled: true, Endpoints: []pageConfig.HttpNotifyEndpoint{{
		Enabled: true,
		URL:     "http://example.com/hook",
		Events:  []string{"topic.published"},
	}}}

	if !shouldNotify(config, "topic.published") {
		t.Fatal("expected enabled matching endpoint to notify")
	}
	if shouldNotify(config, "comment.created") {
		t.Fatal("expected unmatched event to skip notification")
	}
	config.Enabled = false
	if shouldNotify(config, "topic.published") {
		t.Fatal("expected disabled config to skip notification")
	}
}

func TestBuildSignedRequest(t *testing.T) {
	body := []byte(`{"event":"topic.published"}`)
	req, err := buildRequest(pageConfig.HttpNotifyEndpoint{
		URL:    "http://example.com/hook",
		Secret: "secret",
	}, "topic.published", "delivery-1", 1710000000, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte("1710000000."))
	mac.Write(body)
	wantSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if req.Header.Get("X-Goose-Event") != "topic.published" {
		t.Fatalf("event header = %q", req.Header.Get("X-Goose-Event"))
	}
	if req.Header.Get("X-Goose-Delivery") != "delivery-1" {
		t.Fatalf("delivery header = %q", req.Header.Get("X-Goose-Delivery"))
	}
	if req.Header.Get("X-Goose-Signature") != wantSignature {
		t.Fatalf("signature = %q, want %q", req.Header.Get("X-Goose-Signature"), wantSignature)
	}
}

func TestApplyDeliveryFailureDisablesEndpointAfterThreeFailures(t *testing.T) {
	config := pageConfig.HttpNotifyConfig{Enabled: true, Endpoints: []pageConfig.HttpNotifyEndpoint{{
		Id:      "endpoint-1",
		Enabled: true,
		URL:     "http://example.com/hook",
	}}}

	config, changed := applyDeliveryResult(config, "endpoint-1", "http://example.com/hook", false, "timeout")
	if !changed || config.Endpoints[0].FailureCount != 1 || !config.Endpoints[0].Enabled {
		t.Fatalf("first failure not recorded correctly: %+v", config.Endpoints[0])
	}
	config, _ = applyDeliveryResult(config, "endpoint-1", "http://example.com/hook", false, "timeout")
	config, _ = applyDeliveryResult(config, "endpoint-1", "http://example.com/hook", false, "timeout")

	endpoint := config.Endpoints[0]
	if endpoint.Enabled {
		t.Fatal("expected endpoint disabled after three failures")
	}
	if !endpoint.AbnormalTerminated {
		t.Fatal("expected endpoint marked abnormal terminated")
	}
	if endpoint.LastError != "timeout" {
		t.Fatalf("last error = %q", endpoint.LastError)
	}
}

func TestApplyDeliverySuccessResetsFailureCount(t *testing.T) {
	config := pageConfig.HttpNotifyConfig{Endpoints: []pageConfig.HttpNotifyEndpoint{{
		Id:           "endpoint-1",
		Enabled:      true,
		URL:          "http://example.com/hook",
		FailureCount: 2,
		LastError:    "timeout",
	}}}

	config, changed := applyDeliveryResult(config, "endpoint-1", "http://example.com/hook", true, "")
	if !changed {
		t.Fatal("expected successful delivery to update endpoint")
	}
	endpoint := config.Endpoints[0]
	if endpoint.FailureCount != 0 || endpoint.LastError != "" || endpoint.AbnormalTerminated {
		t.Fatalf("success did not reset endpoint failure state: %+v", endpoint)
	}
}

// TestMergeDeliveryStatePreservesSecrets 验证投递结果写回只更新状态字段，
// 端点密文/存量明文 secret 原样保留（issue #324 S1：领域形状 json:"-" 不含密钥，
// 若用领域形状覆盖落库会丢失密文）。
func TestMergeDeliveryStatePreservesSecrets(t *testing.T) {
	storage := pageConfig.HttpNotifyStorageConfig{
		Enabled: true,
		Endpoints: []pageConfig.HttpNotifyStorageEndpoint{
			{Id: "ep1", Name: "webhook", Enabled: true, URL: "https://hook.example.com",
				SecretEncrypted: "sealed-secret", Events: []string{"topic.created"}, TimeoutSeconds: 5},
			{Id: "ep2", Name: "legacy", Enabled: true, URL: "https://legacy.example.com",
				Secret: "legacy-plain", Events: []string{"topic.created"}, TimeoutSeconds: 5},
		},
	}
	applied := storage.ToConfig()
	applied, changed := applyDeliveryResult(applied, "ep1", "https://hook.example.com", false, "boom")
	if !changed {
		t.Fatal("expected delivery failure to change state")
	}
	merged := mergeDeliveryState(storage, applied)

	if merged.Endpoints[0].SecretEncrypted != "sealed-secret" {
		t.Fatalf("secretEncrypted lost after merge: %+v", merged.Endpoints[0])
	}
	if merged.Endpoints[0].FailureCount != 1 || merged.Endpoints[0].LastError != "boom" {
		t.Fatalf("delivery state not applied: %+v", merged.Endpoints[0])
	}
	if merged.Endpoints[1].Secret != "legacy-plain" {
		t.Fatalf("legacy secret lost after merge: %+v", merged.Endpoints[1])
	}
	if merged.Endpoints[1].FailureCount != 0 {
		t.Fatalf("unrelated endpoint mutated: %+v", merged.Endpoints[1])
	}
}
