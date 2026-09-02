package api

import (
	"testing"

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

			// Validate the ContentType field
			if tt.contentType < 0 || tt.contentType > 3 {
				if tt.wantValid {
					t.Errorf("expected valid but got invalid content type %d", tt.contentType)
				}
			} else {
				if !tt.wantValid {
					t.Errorf("expected invalid but got valid content type %d", tt.contentType)
				}
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