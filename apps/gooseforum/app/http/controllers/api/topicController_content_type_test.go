package api

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/validate"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
)

func TestWriteTopicReqContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType int8
		wantValid   bool
	}{
		{
			name:        "default regular post",
			contentType: posts.ContentTypeRegular,
			wantValid:   true,
		},
		{
			name:        "question type",
			contentType: posts.ContentTypeQuestion,
			wantValid:   true,
		},
		{
			name:        "thought type",
			contentType: posts.ContentTypeThought,
			wantValid:   true,
		},
		{
			name:        "article type",
			contentType: posts.ContentTypeArticle,
			wantValid:   true,
		},
		{
			name:        "invalid negative value",
			contentType: -1,
			wantValid:   false,
		},
		{
			name:        "invalid high value",
			contentType: 4,
			wantValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := WriteTopicReq{
				Content:     "test content",
				Title:       "test title",
				CategoryId:  []uint64{1},
				ContentType: tt.contentType,
			}

			// Validate using the actual validator
			err := validate.Valid(req)
			isValid := err == nil

			if isValid != tt.wantValid {
				t.Errorf("validate.Valid() isValid = %v, want %v (contentType=%d, err=%v)", isValid, tt.wantValid, tt.contentType, err)
			}

			// Verify the ContentType is set correctly
			if req.ContentType != tt.contentType {
				t.Errorf("ContentType = %d, want %d", req.ContentType, tt.contentType)
			}
		})
	}
}

func TestPostContentTypeConstants(t *testing.T) {
	// Verify content type constants match expected values
	tests := []struct {
		name     string
		constant int8
		want     int8
	}{
		{"ContentTypeRegular", posts.ContentTypeRegular, 0},
		{"ContentTypeQuestion", posts.ContentTypeQuestion, 1},
		{"ContentTypeThought", posts.ContentTypeThought, 2},
		{"ContentTypeArticle", posts.ContentTypeArticle, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.constant, tt.want)
			}
		})
	}
}
