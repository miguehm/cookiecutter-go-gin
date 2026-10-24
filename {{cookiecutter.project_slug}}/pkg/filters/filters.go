package gormfilter

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	FTypeString  = "string"
	FTypeUuid    = "uuid"
	FTypeDecimal = "decimal"
	FTypeInt     = "int"
)

var (
	allowedOps     = []string{"=", "<>", "!=", ">", ">=", "<", "<=", "LIKE", "ILIKE"}
	identifierRe   = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)
)

type Filter struct {
	Param string
	Field string
	Type  string
	Op    string
}

func (f *Filter) GetQuery(value string) (string, interface{}, error) {
	if f.Field == "" || !identifierRe.MatchString(f.Field) {
		return "", nil, errors.New("invalid filter field")
	}
	if !slices.Contains(allowedOps, strings.ToUpper(f.Op)) {
		return "", nil, errors.New("invalid filter operator")
	}
	op := strings.ToUpper(f.Op)
	if op == "!=" {
		op = "<>"
	}

	var where string
	var params interface{}
	switch f.Type {
	case FTypeString:
		params = value
	case FTypeUuid:
		v, err := uuid.Parse(value)
		if err != nil {
			return "", nil, err
		}
		params = v
	case FTypeInt:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return "", nil, err
		}
		params = v
	case FTypeDecimal:
		v, err := decimal.NewFromString(value)
		if err != nil {
			return "", nil, err
		}
		params = v
	default:
		return "", nil, errors.New("invalid filter type")
	}

	// #nosec G201 -- field and operator validated against allowlists
	where = fmt.Sprintf("%s %s ?", f.Field, op)
	return where, params, nil
}

type FilterBuilder struct {
	Filters       []Filter
	req           *http.Request
	Ordering      []string
	AllowedFields []string

	Query *gorm.DB
}

func (b *FilterBuilder) SetQuery(query *gorm.DB) *FilterBuilder {
	b.Query = query
	return b
}

func (b *FilterBuilder) SetOrdering(ordering []string) *FilterBuilder {
	b.Ordering = ordering
	return b
}

func (b *FilterBuilder) SetAllowedFields(fields []string) *FilterBuilder {
	b.AllowedFields = fields
	return b
}

func (b *FilterBuilder) SetRequest(req *http.Request) *FilterBuilder {
	b.req = req
	return b
}

func (b *FilterBuilder) AddFilter(filter Filter) *FilterBuilder {
	b.Filters = append(b.Filters, filter)
	return b
}

func (b *FilterBuilder) BuildFilter() (string, []interface{}, string, error) {
	var wheres []string
	var params []interface{}
	var errs []error
	if b.req == nil || b.req.URL == nil {
		return "", nil, "", nil
	}
	query := b.req.URL.Query()
	for _, filter := range b.Filters {
		valueParam := query.Get(filter.Param)
		if valueParam == "" {
			continue
		}
		if len(b.AllowedFields) > 0 && !slices.Contains(b.AllowedFields, filter.Field) && !slices.Contains(b.AllowedFields, filter.Param) {
			errs = append(errs, fmt.Errorf("filter field not allowed: %s", filter.Field))
			continue
		}
		w, p, err := filter.GetQuery(valueParam)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		wheres = append(wheres, w)
		params = append(params, p)
	}

	ordering := ""

	if len(b.Ordering) > 0 {
		orderingParam := query.Get("order")
		orderingType := strings.ToLower(query.Get("order_type"))
		if orderingParam != "" {
			if slices.Contains(b.Ordering, orderingParam) && identifierRe.MatchString(orderingParam) {
				if orderingType != "asc" && orderingType != "desc" {
					orderingType = "asc"
				}
				ordering = fmt.Sprintf("%s %s", orderingParam, orderingType)
			} else {
				errs = append(errs, errors.New("invalid ordering"))
			}
		}
		if orderingType != "" && orderingParam == "" {
			errs = append(errs, errors.New("order_type requires order"))
		}
	} else if query.Get("order") != "" || query.Get("order_type") != "" {
		errs = append(errs, errors.New("invalid ordering"))
	}

	return strings.Join(wheres, " and "), params, ordering, errors.Join(errs...)
}

func (b *FilterBuilder) GetFilterQuery() (*gorm.DB, error) {
	wheres, params, ordering, err := b.BuildFilter()
	if err != nil {
		return nil, err
	}
	q := b.Query
	if q == nil {
		return nil, errors.New("query not set")
	}
	if wheres != "" {
		q = q.Where(wheres, params...)
	}
	if ordering != "" {
		q = q.Order(ordering)
	}
	return q, nil
}
