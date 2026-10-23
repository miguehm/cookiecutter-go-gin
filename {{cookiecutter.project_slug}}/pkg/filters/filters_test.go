package gormfilter

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
)

func TestFilterGetQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		filter    Filter
		value     string
		wantWhere string
		wantParam interface{}
		wantErr   bool
	}{
		{
			name:      "string",
			filter:    Filter{Field: "name", Type: FTypeString, Op: "="},
			value:     "alice",
			wantWhere: "name = ?",
			wantParam: "alice",
		},
		{
			name:      "int",
			filter:    Filter{Field: "age", Type: FTypeInt, Op: ">="},
			value:     "18",
			wantWhere: "age >= ?",
			wantParam: int64(18),
		},
		{
			name:      "decimal",
			filter:    Filter{Field: "amount", Type: FTypeDecimal, Op: "<"},
			value:     "10.5",
			wantWhere: "amount < ?",
			wantParam: decimal.RequireFromString("10.5"),
		},
		{
			name:    "invalid int",
			filter:  Filter{Field: "age", Type: FTypeInt, Op: "="},
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "invalid decimal",
			filter:  Filter{Field: "amount", Type: FTypeDecimal, Op: "="},
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "invalid uuid",
			filter:  Filter{Field: "id", Type: FTypeUuid, Op: "="},
			value:   "not-a-uuid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			where, param, err := tt.filter.GetQuery(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if where != tt.wantWhere {
				t.Fatalf("where = %q, want %q", where, tt.wantWhere)
			}

			wantDecimal, isDecimal := tt.wantParam.(decimal.Decimal)
			if isDecimal {
				gotDecimal, ok := param.(decimal.Decimal)
				if !ok || !gotDecimal.Equal(wantDecimal) {
					t.Fatalf("param = %v, want %v", param, tt.wantParam)
				}
				return
			}

			if !reflect.DeepEqual(param, tt.wantParam) {
				t.Fatalf("param = %v, want %v", param, tt.wantParam)
			}
		})
	}
}

func TestFilterBuilderBuildFilter(t *testing.T) {
	t.Parallel()

	query := url.Values{
		"name":       {"alice"},
		"age":        {"18"},
		"order":      {"name"},
		"order_type": {"desc"},
	}
	req := &http.Request{URL: &url.URL{RawQuery: query.Encode()}}

	b := FilterBuilder{req: req}
	b.AddFilter(Filter{Param: "name", Field: "name", Type: FTypeString, Op: "="}).
		AddFilter(Filter{Param: "age", Field: "age", Type: FTypeInt, Op: "="}).
		SetOrdering([]string{"name"})

	wheres, params, ordering := b.BuildFilter()

	if wheres != "name = ? and age = ?" {
		t.Fatalf("wheres = %q", wheres)
	}
	wantParams := []interface{}{"alice", int64(18)}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("params = %v, want %v", params, wantParams)
	}
	if ordering != "name desc" {
		t.Fatalf("ordering = %q, want %q", ordering, "name desc")
	}
}

func TestFilterBuilderRejectsUnknownOrdering(t *testing.T) {
	t.Parallel()

	query := url.Values{
		"order":      {"password"},
		"order_type": {"desc"},
	}
	req := &http.Request{URL: &url.URL{RawQuery: query.Encode()}}

	b := FilterBuilder{req: req}
	b.SetOrdering([]string{"name"})

	_, _, ordering := b.BuildFilter()
	if ordering != "" {
		t.Fatalf("ordering = %q, want empty", ordering)
	}
}
