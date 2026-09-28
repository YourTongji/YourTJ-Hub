package messages

import (
	"strings"
	"testing"
)

func TestForwardSnapshotBoundsAndFallback(t *testing.T) {
	bundle := &ForwardedBundle{Version: 1, Messages: []ForwardedEntry{{SenderName: "Alice", Content: "hello", CreatedAt: "2026-09-28T10:00:00Z", MsgType: 1}}}
	raw, err := bundle.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if got := DisplayContent(raw, ForwardType); got != "[Chat history]\nAlice: hello" {
		t.Fatalf("fallback = %q", got)
	}
	if ParseForward(`{"version":2,"messages":[]}`) != nil {
		t.Fatal("unknown version accepted")
	}
	for _, content := range []string{"bad JSON", strings.Repeat("x", MaxForwardBytes+1), `{"version":1,"messages":[{"msgType":4}]}`} {
		if ParseForward(content) != nil {
			t.Fatal("invalid content accepted")
		}
	}
	bundle.Messages = make([]ForwardedEntry, 51)
	if _, err := bundle.Encode(); err == nil {
		t.Fatal("oversized entry count accepted")
	}
	if got := DisplayContent("literal text", 1); got != "literal text" {
		t.Fatal("ordinary text changed")
	}
}

func TestForwardSnapshotPreservesNestedHistory(t *testing.T) {
	raw := `{"version":1,"messages":[{"senderName":"Forwarder","avatarUrl":"/forwarder.webp","content":"[Chat history]\nAlice: original","createdAt":"2026-09-28T10:00:00Z","msgType":4,"forwarded":{"version":1,"messages":[{"senderName":"Alice","avatarUrl":"/alice.webp","content":"original","createdAt":"2026-09-28T09:00:00Z","msgType":1}]}}]}`
	bundle := ParseForward(raw)
	if bundle == nil {
		t.Fatal("valid nested history was rejected")
	}
	encoded, err := bundle.Encode()
	if err != nil || !strings.Contains(encoded, `"forwarded":`) || !strings.Contains(encoded, "/alice.webp") {
		t.Fatalf("nested content/avatar lost: %s (%v)", encoded, err)
	}
}

func TestForwardSnapshotTreeBounds(t *testing.T) {
	leaf := func(count int) *ForwardedBundle {
		bundle := &ForwardedBundle{Version: 1}
		for range count {
			bundle.Messages = append(bundle.Messages, ForwardedEntry{SenderName: "Alice", Content: "private body", MsgType: 1})
		}
		return bundle
	}
	wrap := func(child *ForwardedBundle) *ForwardedBundle {
		return &ForwardedBundle{Version: 1, Messages: []ForwardedEntry{{SenderName: "Forwarder", Content: "fallback", MsgType: ForwardType, Forwarded: child}}}
	}
	depth := leaf(1)
	for i := 1; i < MaxForwardDepth; i++ {
		depth = wrap(depth)
	}
	raw, err := depth.Encode()
	if err != nil || ParseForward(raw) == nil || !strings.Contains(DisplayContent(raw, ForwardType), "private body") {
		t.Fatalf("maximum depth must remain readable: %v", err)
	}
	if _, err := wrap(depth).Encode(); err == nil {
		t.Fatal("excessive depth accepted")
	}
	wide := wrap(leaf(24))
	wide.Messages = append(wide.Messages, wrap(leaf(24)).Messages[0])
	if _, err := wide.Encode(); err != nil {
		t.Fatalf("50 total entries should fit: %v", err)
	}
	wide.Messages[1].Forwarded.Messages = append(wide.Messages[1].Forwarded.Messages, leaf(1).Messages[0])
	if _, err := wide.Encode(); err == nil {
		t.Fatal("51 total entries accepted")
	}
	tooLarge := wrap(leaf(1))
	tooLarge.Messages[0].Forwarded.Messages[0].Content = strings.Repeat("x", MaxForwardBytes)
	if _, err := tooLarge.Encode(); err == nil {
		t.Fatal("nested byte limit exceeded")
	}
	for _, invalid := range []*ForwardedBundle{
		wrap(nil), wrap(&ForwardedBundle{Version: 1}), wrap(&ForwardedBundle{Version: 2, Messages: leaf(1).Messages}),
		{Version: 1, Messages: []ForwardedEntry{{MsgType: 1, Forwarded: leaf(1)}}},
	} {
		if _, err := invalid.Encode(); err == nil {
			t.Fatalf("invalid nested snapshot accepted: %#v", invalid)
		}
	}
}
