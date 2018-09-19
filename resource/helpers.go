package resource

import (
	"fmt"
	"strings"

	"github.com/aghape/core/utils"
	"github.com/moisespsena-go/aorm"
)

// StringToPrimaryQuery to primary query params
func StringToPrimaryQuery(res Resourcer, value string, exclude ...bool) (string, []interface{}) {
	return StringsToPrimaryQuery(res, strings.Split(strings.Trim(value, ","), ","), exclude...)
}

// StringsToPrimaryQuery to primary query params
func StringsToPrimaryQuery(res Resourcer, values []string, exclude ...bool) (string, []interface{}) {
	if len(values) == 0 {
		return "", nil
	}

	var (
		sqls          []string
		primaryValues []interface{}
		scope         = res.GetFakeScope()
	)

	fields := res.GetPrimaryFields()
	if len(values) > len(fields) {
		return "", nil
	}
	op := "="
	if exclude != nil && exclude[0] {
		op = "<>"
	}
	for i, field := range fields {
		sqls = append(sqls, fmt.Sprintf("%v.%v "+op+" ?", scope.QuotedTableName(), scope.Quote(field.DBName)))
		primaryValues = append(primaryValues, values[i])
	}

	return strings.Join(sqls, " AND "), primaryValues
}

// MetaValuesToPrimaryQuery to primary query params from meta values
func MetaValuesToPrimaryQuery(res Resourcer, metaValues *MetaValues, exclude ...bool) (string, []interface{}) {
	var (
		values []string
	)

	if metaValues != nil {
		for _, field := range res.GetPrimaryFields() {
			if metaField := metaValues.Get(field.Name); metaField != nil {
				values = append(values, utils.ToString(metaField.Value))
			}
		}
	}

	return StringsToPrimaryQuery(res, values, exclude...)
}

// ValuesToPrimaryQuery to primary query params from values
func ValuesToPrimaryQuery(res Resourcer, exclude bool, values ...interface{}) (string, []interface{}) {
	var (
		sql, op string
		scope   = res.GetFakeScope()
	)

	if values != nil {
		field := res.GetPrimaryFields()[0]
		if exclude {
			op = " NOT"
		}
		sql = fmt.Sprintf("%v.%v"+op+" IN %v", scope.QuotedTableName(), scope.Quote(field.DBName),
			aorm.TupleQueryArgs(len(values)))
	}

	return sql, values
}
