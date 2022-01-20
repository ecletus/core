package resource

import (
	"fmt"
	"strings"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
)

func MustParseID(res Resourcer, s string) aorm.ID {
	if id, err := res.ParseID(s); err != nil {
		panic(err)
	} else {
		return id
	}
}

func StringToPrimaryQuery(ctx *core.Context, res Resourcer, s string, exclude bool) (query string, primaryValues []interface{}, err error) {
	var id aorm.ID
	if id, err = res.ParseID(s); err != nil {
		return
	}
	return IdToPrimaryQuery(ctx, res, exclude, id)
}

// IdToPrimaryQuery to returns primary query params
func IdToPrimaryQuery(ctx *core.Context, res Resourcer, exclude bool, id ...aorm.ID) (query string, args []interface{}, err error) {
	if id == nil {
		return "", nil, nil
	}
	var (
		sqls   []string
		scope  = ctx.DB().NewScope(res.GetValue())
		slice  = len(id) > 1
		fields = res.GetPrimaryFields()
	)

	for _, id := range id {
		args = append(args, res.PrimaryValues(id)...)
	}

	if slice {
		if len(fields) == 1 {
			op := " IN"
			if exclude {
				op = "NOT IN"
			}

			sqls = append(sqls, fmt.Sprintf("%v.%v"+op+" IN %v", scope.FromName(), scope.Quote(fields[0].DBName),
				aorm.TupleQueryArgs(len(args))))
		} else {
			var fieldsSql []string
			for _, field := range fields {
				fieldsSql = append(fieldsSql, fmt.Sprintf("%v.%v = ?", scope.FromName(), scope.Quote(field.DBName)))
			}
			for range id {
				sqls = append(sqls, "("+strings.Join(fieldsSql, " AND ")+")")
			}
			query = strings.Join(sqls, " OR ")
			if exclude {
				query = "NOT (" + query + ")"
			}
		}
	} else {
		op := "="
		if exclude {
			op = "<>"
		}
		for _, field := range fields {
			sqls = append(sqls, fmt.Sprintf("%v.%v "+op+" ?", scope.FromName(), scope.Quote(field.DBName)))
		}
		query = strings.Join(sqls, " AND ")
	}
	return
}
