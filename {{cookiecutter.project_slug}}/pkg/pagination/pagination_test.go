package pagination

import (
	"net/http"
	"net/url"
	"testing"
)

func TestPaginationRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
	}{
		{name: "defaults", query: "", wantPage: 1, wantSize: 50},
		{name: "valid values", query: "page=2&size=20", wantPage: 2, wantSize: 20},
		{name: "invalid page", query: "page=abc&size=20", wantPage: 1, wantSize: 20},
		{name: "invalid size", query: "page=2&size=abc", wantPage: 2, wantSize: 50},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := &http.Request{URL: &url.URL{RawQuery: tt.query}}
			p := NewStandardPagination().Request(req)

			if p.RequestParams.Page != tt.wantPage {
				t.Fatalf("page = %d, want %d", p.RequestParams.Page, tt.wantPage)
			}
			if p.RequestParams.Size != tt.wantSize {
				t.Fatalf("size = %d, want %d", p.RequestParams.Size, tt.wantSize)
			}
		})
	}
}

func TestNewStandardPaginationMaxSize(t *testing.T) {
	t.Parallel()

	p := NewStandardPagination()
	if p.maxSize != 50 {
		t.Fatalf("maxSize = %d, want 50", p.maxSize)
	}
}
