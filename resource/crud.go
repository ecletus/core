package resource

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/qor/qor"
	"github.com/qor/qor/utils"
	"github.com/qor/roles"
)

// ToPrimaryQueryParams to primary query params
func (res *Resource) ToPrimaryQueryParams(primaryValue string, context *qor.Context) (string, []interface{}) {
	if primaryValue != "" {
		scope := context.DB.NewScope(res.Value)

		// multiple primary fields
		if len(res.PrimaryFields) > 1 {
			if primaryValueStrs := strings.Split(primaryValue, ","); len(primaryValueStrs) == len(res.PrimaryFields) {
				sqls := []string{}
				primaryValues := []interface{}{}
				for idx, field := range res.PrimaryFields {
					sqls = append(sqls, fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(), scope.Quote(field.DBName)))
					primaryValues = append(primaryValues, primaryValueStrs[idx])
				}

				return strings.Join(sqls, " AND "), primaryValues
			}
		}

		// fallback to first configured primary field
		if len(res.PrimaryFields) > 0 {
			return fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(), scope.Quote(res.PrimaryFields[0].DBName)), []interface{}{primaryValue}
		}

		// if no configured primary fields found
		if primaryField := scope.PrimaryField(); primaryField != nil {
			return fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(), scope.Quote(primaryField.DBName)), []interface{}{primaryValue}
		}
	}

	return "", []interface{}{}
}

// ToPrimaryQueryParamsFromMetaValue to primary query params from meta values
func (res *Resource) ToPrimaryQueryParamsFromMetaValue(metaValues *MetaValues, context *qor.Context) (string, []interface{}) {
	var (
		sqls          []string
		primaryValues []interface{}
		scope         = context.DB.NewScope(res.Value)
	)

	if metaValues != nil {
		for _, field := range res.PrimaryFields {
			if metaField := metaValues.Get(field.Name); metaField != nil {
				sqls = append(sqls, fmt.Sprintf("%v.%v = ?", scope.QuotedTableName(), scope.Quote(field.DBName)))
				primaryValues = append(primaryValues, utils.ToString(metaField.Value))
			}
		}
	}

	return strings.Join(sqls, " AND "), primaryValues
}

func (res *Resource) findOneHandler(result interface{}, metaValues *MetaValues, context *qor.Context) (err error) {
	if res.HasPermission(roles.Read, context) {
		var (
			primaryQuerySQL string
			primaryParams   []interface{}
		)

		if metaValues == nil {
			primaryQuerySQL, primaryParams = res.ToPrimaryQueryParams(context.ResourceID, context)
		} else {
			primaryQuerySQL, primaryParams = res.ToPrimaryQueryParamsFromMetaValue(metaValues, context)

			if len(primaryParams) == 1 {
				if s, ok := primaryParams[0].(string); ok {
					if s == "" {
						return nil
					}
				} else if s, ok := primaryParams[0].(int64); ok {
					if s == 0 {
						return nil
					}
				}
			}
		}

		if primaryQuerySQL != "" {
			if metaValues == nil {
			} else {
				if destroy := metaValues.Get("_destroy"); destroy != nil {
					if fmt.Sprint(destroy.Value) != "0" && res.HasPermission(roles.Delete, context) {
						context.DB.Delete(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...)
						return ErrProcessorSkipLeft
					}
				}
			}
			err := context.DB.First(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...).Error
			return err
		}

		return errors.New("failed to find")
	}
	return roles.ErrPermissionDenied
}

func (res *Resource) findManyHandler(result interface{}, context *qor.Context) error {
	if res.HasPermission(roles.Read, context) {
		db := context.DB
		if _, ok := db.Get("qor:getting_total_count"); ok {
			return context.DB.Count(result).Error
		}
		return context.DB.Set("gorm:order_by_primary_key", "DESC").Find(result).Error
	}

	return roles.ErrPermissionDenied
}

func (res *Resource) saveHandler(result interface{}, context *qor.Context) error {
	if (context.DB.NewScope(result).PrimaryKeyZero() &&
		res.HasPermission(roles.Create, context)) || // has create permission
		res.HasPermission(roles.Update, context) { // has update permission
		return context.DB.Save(result).Error
	}
	return roles.ErrPermissionDenied
}

func (res *Resource) deleteHandler(result interface{}, context *qor.Context) error {
	if res.HasPermission(roles.Delete, context) {
		if primaryQuerySQL, primaryParams := res.ToPrimaryQueryParams(context.ResourceID, context); primaryQuerySQL != "" {
			if !context.DB.First(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...).RecordNotFound() {
				return context.DB.Delete(result).Error
			}
		}
		return gorm.ErrRecordNotFound
	}
	return roles.ErrPermissionDenied
}

// CallFindOne call find one method
func (res *Resource) CallFindOne(result interface{}, metaValues *MetaValues, context *qor.Context) error {
	return res.FindOneHandler(result, metaValues, context)
}

// CallFindMany call find many method
func (res *Resource) CallFindMany(result interface{}, context *qor.Context) error {
	return res.FindManyHandler(result, context)
}

// CallFindOne call find one method
func (res *Resource) CallFindOneReadonly(result interface{}, metaValues *MetaValues, context *qor.Context) error {
	return res.FindOneReadonlyHandler(result, metaValues, context)
}

// CallFindMany call find many method
func (res *Resource) CallFindManyReadonly(result interface{}, context *qor.Context) error {
	return res.FindManyReadonlyHandler(result, context)
}

// CallSave call save method
func (res *Resource) CallSave(result interface{}, context *qor.Context) error {
	return res.SaveHandler(result, context)
}

// CallDelete call delete method
func (res *Resource) CallDelete(result interface{}, context *qor.Context) error {
	return res.DeleteHandler(result, context)
}
