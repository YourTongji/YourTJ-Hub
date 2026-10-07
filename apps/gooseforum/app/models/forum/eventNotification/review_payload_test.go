package eventNotification

import "testing"

func TestRejectedPayloadHasOnlyFixedLengthMaskedSubject(t *testing.T) {
	for _, text := range []string{"校园生活中的讨论", "A very long private title Z", "🙂私密内容🙂"} {
		input := NotificationPayload{TopicTitle: text, Title: text, Content: text, TemplateParams: NotificationTemplateParams{Preview: text}, TopicId: 42, PostNo: 3}
		got := RedactReviewRejectedPayload(input)
		chars := []rune(text)
		want := string(chars[0]) + "******" + string(chars[len(chars)-1])
		if got.TopicTitle != want || got.Title != "" || got.Content != "" || got.TemplateParams.Preview != "" || got.TopicId != 42 || got.PostNo != 3 {
			t.Fatalf("unsafe payload: %+v", got)
		}
		if twice := RedactReviewRejectedPayload(got); twice != got {
			t.Fatalf("mask not idempotent: %+v", twice)
		}
	}
}
