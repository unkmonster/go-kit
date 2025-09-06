package query

import (
	"strings"

	"github.com/unkmonster/go-kit/db/gorm/scope"
	"github.com/unkmonster/go-kit/db/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type options struct {
	stableOrderKey string
}

type Option func(o *options)

func New(m *query.Model, opts ...Option) scope.ScopeFunc {
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}

	return func(db *gorm.DB) *gorm.DB {
		if m == nil {
			return db
		}

		db = db.Offset(int(m.Offset))
		if m.Limit != 0 {
			db = db.Limit(int(m.Limit))
		}

		if m.SortBy != "" {
			db = db.Order(clause.OrderByColumn{
				Column: clause.Column{Name: m.SortBy},
				Desc:   strings.EqualFold("desc", m.Order),
			})
		}

		if options.stableOrderKey != "" && m.SortBy != options.stableOrderKey {
			db = db.Order(options.stableOrderKey)
		}
		return db
	}
}

func WithStableOrderKey(key string) Option {
	return func(o *options) {
		o.stableOrderKey = key
	}
}
