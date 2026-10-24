package pagination

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"gorm.io/gorm"
)

func NewPagination(maxSize int) *Pagination {
	if maxSize <= 0 {
		maxSize = 50
	}
	return &Pagination{
		maxSize: maxSize,
	}
}

func NewStandardPagination() *Pagination {
	return NewPagination(50)
}

type Pagination struct {
	maxSize       int
	Query         *gorm.DB
	RequestParams *RequestParams
	withCount     bool
}

type Page struct {
	Items interface{}
	Total int64
}

type RequestParams struct {
	Size int
	Page int
}

func (p *Pagination) With(query *gorm.DB) *Pagination {
	p.Query = query
	return p
}

func (p *Pagination) SetCount(count bool) *Pagination {
	p.withCount = count
	return p
}

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
