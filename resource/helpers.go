package resource

import (
	"fmt"
	"strings"

	"github.com/aghape/core/utils"
)

// ToPrimaryQueryParams to primary query params
func ToPrimaryQueryParams(res Resourcer, primaryValue string) (string, []interface{}) {
	var (
		primaryFields = res.GetPrimaryFields()
		scope         = res.GetFakeScope()
	)
	if primaryValue != "" {
		// multiple primary fields
		if len(primaryFields) > 1 {
			if primaryValueStrs := strings.Split(primaryValue, ","); len(primaryValueStrs) == len(primaryFields) {
				sqls := []string{}
				primaryValues := []interface{}{}
				for idx, field := range primaryFields {
					sqls = append(sqls, fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(),
						scope.Quote(field.DBName)))
					primaryValues = append(primaryValues, primaryValueStrs[idx])
				}

				return strings.Join(sqls, " AND "), primaryValues
			}
		}

		// fallback to first configured primary field
		if len(primaryFields) > 0 {
			return fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(),
				scope.Quote(primaryFields[0].DBName)), []interface{}{primaryValue}
		}

		// if no configured primary fields found
		if primaryField := scope.PrimaryField(); primaryField != nil {
			return fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(),
				scope.Quote(primaryField.DBName)), []interface{}{primaryValue}
		}
	}

	return "", []interface{}{}
}

// ToPrimaryQueryParamsFromMetaValue to primary query params from meta values
func ToPrimaryQueryParamsFromMetaValue(res Resourcer, metaValues *MetaValues) (string, []interface{}) {
	var (
		sqls          []string
		primaryValues []interface{}
		scope         = res.GetFakeScope()
	)

	if metaValues != nil {
		for _, field := range res.GetPrimaryFields() {
			if metaField := metaValues.Get(field.Name); metaField != nil {
				sqls = append(sqls, fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(), scope.Quote(field.DBName)))
				primaryValues = append(primaryValues, utils.ToString(metaField.Value))
			}
		}
	}

	return strings.Join(sqls, " AND "), primaryValues
}
