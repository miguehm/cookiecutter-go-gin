// Package pagination turns the "page" and "size" query parameters into
// GORM limit/offset clauses and wraps the result in a [Page].
package pagination

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"gorm.io/gorm"
)

// NewPagination returns a Pagination that accepts pages of at most maxSize
// rows; maxSize values below one fall back to the default of 50.
func NewPagination(maxSize int) *Pagination {
	if maxSize <= 0 {
		maxSize = 50
	}
	return &Pagination{
		maxSize: maxSize,
	}
}

// NewStandardPagination returns a Pagination with the default page size of 50.
func NewStandardPagination() *Pagination {
	return NewPagination(50)
}

// Pagination applies limit/offset to a GORM query using the parameters parsed
// from the request.
type Pagination struct {
	maxSize       int
	Query         *gorm.DB
	RequestParams *RequestParams
	withCount     bool
}

// Page is the paginated result returned to the client.
type Page struct {
	Items interface{}
	Total int64
}

// RequestParams holds the validated "page" and "size" query parameters.
type RequestParams struct {
	Size int
	Page int
}

// With stores the query that will be paginated.
func (p *Pagination) With(query *gorm.DB) *Pagination {
	p.Query = query
	return p
}

// SetCount enables counting the total number of rows for the current filters.
func (p *Pagination) SetCount(count bool) *Pagination {
	p.withCount = count
	return p
}

// Request parses and validates the "page" and "size" query parameters of the
// given request, storing them on the pagination.
func (p *Pagination) Request(request *http.Request) (*Pagination, error) {
	if request == nil || request.URL == nil {
		return nil, errors.New("invalid request")
	}
	query := request.URL.Query()
	requestParams := &RequestParams{}
	sizeStr := query.Get("size")
	if sizeStr == "" {
		requestParams.Size = 50
	} else {
		i, e := strconv.Atoi(sizeStr)
		if e != nil {
			return nil, errors.New("invalid size")
		}
		if i <= 0 || i > p.maxSize {
			return nil, fmt.Errorf("invalid size")
		}
		requestParams.Size = i
	}

	pageStr := query.Get("page")
	if pageStr == "" {
		requestParams.Page = 1
	} else {
		i, e := strconv.Atoi(pageStr)
		if e != nil {
			return nil, errors.New("invalid page")
		}
		if i < 1 {
			return nil, errors.New("invalid page")
		}
		requestParams.Page = i
	}
	p.RequestParams = requestParams
	return p, nil
}

// Response runs the stored query with limit/offset applied and fills results
// with the current page, returning it together with the total row count when
// counting is enabled.
func (p *Pagination) Response(results interface{}) (*Page, error) {
	if p == nil || p.Query == nil {
		return nil, errors.New("query not set")
	}
	if p.RequestParams == nil {
		p.RequestParams = &RequestParams{Size: 50, Page: 1}
	}
	if p.RequestParams.Size <= 0 || p.RequestParams.Size > p.maxSize || p.RequestParams.Page < 1 {
		return nil, errors.New("invalid pagination parameters")
	}

	var total int64
	dbs := p.Query.Statement.Session(&gorm.Session{NewDB: true})
	q := dbs.Unscoped().Table("(?) AS s", p.Query)
	if p.withCount {
		if err := q.Count(&total).Error; err != nil {
			return nil, err
		}
		q = q.Limit(p.RequestParams.Size).Offset((p.RequestParams.Page - 1) * p.RequestParams.Size)
		if err := q.Find(results).Error; err != nil {
			return nil, err
		}
	} else {
		q = q.Limit(p.RequestParams.Size).Offset((p.RequestParams.Page - 1) * p.RequestParams.Size)
		if err := q.Find(results).Error; err != nil {
			return nil, err
		}
	}

	return &Page{
		Items: results,
		Total: total,
	}, nil
}
