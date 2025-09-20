package filter

import (
	"fmt"

	"github.com/unkmonster/go-kit/db/gorm/scope"
	"gorm.io/gorm"
)

type options struct {
	equal map[string]any
}

type Option func(o *options)

func WithEqualValues(m map[string]any) Option {
	return func(o *options) {
		o.equal = m
	}
}

func New(opts ...Option) scope.ScopeFunc {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	return func(db *gorm.DB) *gorm.DB {
		for k, v := range o.equal {
			db = db.Where(fmt.Sprintf("%s=?", k), v)
		}
		return db
	}
}
