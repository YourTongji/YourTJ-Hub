package agentmention

import (
	"reflect"
	"strings"
	"testing"
)

func TestFindBasicMentions(t *testing.T) {
	text := "hello @agent-one and @agent_two!"
	got := Find(text)
	want := []string{"agent-one", "agent_two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Find(%q) = %#v, want %#v", text, got, want)
	}
}

func TestFindCaseSensitive(t *testing.T) {
	text := "@AgentBot and @agentbot are different"
	got := Find(text)
	want := []string{"AgentBot", "agentbot"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Find(%q) = %#v, want %#v", text, got, want)
	}
}

func TestFindRequiresValidBoundaries(t *testing.T) {
	tests := []struct {
		text string
		want []string
	}{
		{"@short", nil},                      // below min length
		{"@" + strings.Repeat("a", 33), nil}, // above max length
		{"x@agent-one", nil},                 // previous byte is a username char
		{"(@agent-one)", []string{"agent-one"}},
		{"@agent-one. @agent-two", []string{"agent-one", "agent-two"}},
		{"@agent-one_2", []string{"agent-one_2"}},
		{"a@b@agent-one", nil},                  // @ preceded by a username char is not a boundary
		{"@-agent-one", []string{"-agent-one"}}, // leading dash allowed by grammar
		{"@_agent_one", []string{"_agent_one"}},
	}
	for _, tc := range tests {
		if got := Find(tc.text); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("Find(%q) = %#v, want %#v", tc.text, got, tc.want)
		}
	}
}

func TestFindDeduplicatesInTextOrder(t *testing.T) {
	text := "@agent-one then @agent-two then @agent-one again"
	got := Find(text)
	want := []string{"agent-one", "agent-two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Find(%q) = %#v, want %#v", text, got, want)
	}
}

func TestFindMaxMentions(t *testing.T) {
	var builder strings.Builder
	for i := 0; i < 15; i++ {
		builder.WriteString("@agent-")
		builder.WriteByte(byte('a' + i))
		builder.WriteString(" ")
	}
	got := Find(builder.String())
	if len(got) != MaxMentions {
		t.Fatalf("Find returned %d mentions, want max %d", len(got), MaxMentions)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Fatalf("mentions out of text order: %#v", got)
		}
	}
}

func TestFindEmptyAndNoMentions(t *testing.T) {
	if got := Find(""); len(got) != 0 {
		t.Fatalf("Find empty = %#v", got)
	}
	if got := Find("no mentions here"); len(got) != 0 {
		t.Fatalf("Find plain text = %#v", got)
	}
	if got := Find("@"); len(got) != 0 {
		t.Fatalf("Find lone @ = %#v", got)
	}
}

func TestFindStopsAfterMaxEvenWithMoreValid(t *testing.T) {
	text := "@agent-aa @agent-bb @agent-cc @agent-dd @agent-ee @agent-ff @agent-gg @agent-hh @agent-ii @agent-jj @agent-kk"
	got := Find(text)
	if len(got) != MaxMentions {
		t.Fatalf("Find returned %d mentions, want %d", len(got), MaxMentions)
	}
	if got[len(got)-1] != "agent-jj" {
		t.Fatalf("last mention = %q, want agent-jj (first ten in order)", got[len(got)-1])
	}
}
