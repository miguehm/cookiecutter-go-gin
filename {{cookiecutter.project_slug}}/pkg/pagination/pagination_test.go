package pagination

import (
	"net/http"
	"net/url"
	"testing"
)

func TestPaginationRequest(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantPage int
		wantSize int
		wantErr  bool
	}{
		{name: "defaults", query: "", wantPage: 1, wantSize: 50},
		{name: "valid values", query: "page=2&size=20", wantPage: 2, wantSize: 20},
		{name: "invalid page", query: "page=abc&size=20", wantErr: true},
		{name: "invalid size", query: "page=2&size=abc", wantErr: true},
		{name: "negative page", query: "page=0&size=20", wantErr: true},
		{name: "zero size", query: "page=1&size=0", wantErr: true},
		{name: "size too large", query: "page=1&size=100", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{URL: &url.URL{RawQuery: tt.query}}
			p, err := NewStandardPagination().Request(req)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
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
	p := NewStandardPagination()
	if p.maxSize != 50 {
		t.Fatalf("maxSize = %d, want 50", p.maxSize)
	}
}
