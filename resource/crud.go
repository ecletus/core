package resource

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/moisespsena-go/aorm"
	"github.com/aghape/core"
	"github.com/aghape/core/utils"
	"github.com/aghape/roles"
)

// ToPrimaryQueryParams to primary query params
func (res *Resource) ToPrimaryQueryParams(primaryValue string) (string, []interface{}) {
	if primaryValue != "" {
		// multiple primary fields
		if len(res.PrimaryFields) > 1 {
			if primaryValueStrs := strings.Split(primaryValue, ","); len(primaryValueStrs) == len(res.PrimaryFields) {
				sqls := []string{}
				primaryValues := []interface{}{}
				for idx, field := range res.PrimaryFields {
					sqls = append(sqls, fmt.Sprintf("%v.%v = ?", res.FakeScope.QuotedTableName(),
						res.FakeScope.Quote(field.DBName)))
					primaryValues = append(primaryValues, primaryValueStrs[idx])
				}

				return strings.Join(sqls, " AND "), primaryValues
			}
		}

		// fallback to first configured primary field
		if len(res.PrimaryFields) > 0 {
			return fmt.Sprintf("%v.%v = ?", res.FakeScope.QuotedTableName(),
				res.FakeScope.Quote(res.PrimaryFields[0].DBName)), []interface{}{primaryValue}
		}

		// if no configured primary fields found
		if primaryField := res.FakeScope.PrimaryField(); primaryField != nil {
			return fmt.Sprintf("%v.%v = ?", res.FakeScope.QuotedTableName(),
				res.FakeScope.Quote(primaryField.DBName)), []interface{}{primaryValue}
		}
	}

	return "", []interface{}{}
}

// ToPrimaryQueryParamsFromMetaValue to primary query params from meta values
func (res *Resource) ToPrimaryQueryParamsFromMetaValue(metaValues *MetaValues) (string, []interface{}) {
	var (
		sqls          []string
		primaryValues []interface{}
	)

	if metaValues != nil {
		for _, field := range res.PrimaryFields {
			if metaField := metaValues.Get(field.Name); metaField != nil {
				sqls = append(sqls, fmt.Sprintf("%v.%v = ?", res.FakeScope.QuotedTableName(), res.FakeScope.Quote(field.DBName)))
				primaryValues = append(primaryValues, utils.ToString(metaField.Value))
			}
		}
	}

	return strings.Join(sqls, " AND "), primaryValues
}

func (res *Resource) CallFindOneHandler(resourcer Resourcer, result interface{}, metaValues *MetaValues, context *core.Context) (err error) {
	originalContext := context
	context = context.Clone()
	var (
		primaryQuerySQL string
		primaryParams   []interface{}
	)

	if metaValues == nil {
		primaryQuerySQL, primaryParams = res.ToPrimaryQueryParams(context.ResourceID)
	} else {
		primaryQuerySQL, primaryParams = res.ToPrimaryQueryParamsFromMetaValue(metaValues)

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
		if metaValues != nil {
			if destroy := metaValues.Get("_destroy"); destroy != nil {
				if fmt.Sprint(destroy.Value) != "0" && res.HasPermission(roles.Delete, context) {
					context.DB.Delete(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...)
					return ErrProcessorSkipLeft
				}
			}
		}

		e := &DBEvent{Resource: res, Action: E_DB_ACTION_FIND_ONE, Recorde: result, Context: context, OriginalContext:originalContext}
		if err = res.triggerDBAction(e.before()); err != nil {
			return
		}

		if err = context.DB.First(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...).Error; err != nil {
			res.triggerDBAction(e.error(err))
			return err
		}

		err = res.triggerDBAction(e.after())
		return err
	}

	return errors.New("failed to find")
}

func (res *Resource) CallFindManyLayout(r Resourcer, result interface{}, context *core.Context, layout LayoutInterface) error {
	if res.HasPermission(roles.Read, context) {
		if layout.GetPrepare() != nil {
			layout.GetPrepare()(r, context)
		}
		return layout.GetMany()(r, result, context)
	}
	return roles.ErrPermissionDenied
}

func (res *Resource) CallFindOneLayout(r Resourcer, result interface{}, metaValues *MetaValues, context *core.Context, layout LayoutInterface) (err error) {
	if res.HasPermission(roles.Read, context) {
		if layout.GetPrepare() != nil {
			layout.GetPrepare()(r, context)
		}
		return layout.GetOne()(r, result, metaValues, context)
	}
	return roles.ErrPermissionDenied
}

func (res *Resource) FindManyLayout(result interface{}, context *core.Context, layout LayoutInterface) error {
	return res.CallFindManyLayout(res, result, context, layout)
}

func (res *Resource) FindOneLayout(result interface{}, metaValues *MetaValues, context *core.Context, layout LayoutInterface) (err error) {
	return res.CallFindOneLayout(res, result, metaValues, context, layout)
}

func (res *Resource) FindMany(result interface{}, context *core.Context) error {
	return res.FindManyLayout(result, context, res.Layouts[DEFAULT_LAYOUT])
}

func (res *Resource) FindOne(result interface{}, metaValues *MetaValues, context *core.Context) error {
	return res.FindOneLayout(result, metaValues, context, res.Layouts[DEFAULT_LAYOUT])
}

func (res *Resource) CallFindManyHandler(resourcer Resourcer, result interface{}, context *core.Context) error {
	return CallFindManyHandler(resourcer, result, context)
}

func CallFindManyHandler(resourcer Resourcer, result interface{}, context *core.Context) (err error) {
	originalContext := context
	context = context.Clone()
	callbacks := resourcer.GetBeforeFindCallbacks()
	for _, cb := range callbacks {
		err = cb.Handler(resourcer, result, context, nil)
		if err != nil {
			return err
		}
	}

	var e *DBEvent
	res := resourcer.GetResource()

	if _, ok := context.DB.Get("qor:getting_total_count"); ok {
		e = &DBEvent{Resource: resourcer, Action: E_DB_ACTION_COUNT, Recorde: result, Context: context, OriginalContext:originalContext}
		if err = res.triggerDBAction(e.before()); err != nil {
			return err
		}
		if err = context.DB.Count(result).Error; err != nil {
			res.triggerDBAction(e.error(err))
			return err
		}

		err = res.triggerDBAction(e.after())
		return err
	}

	e = &DBEvent{Resource: resourcer, Action: E_DB_ACTION_FIND_MANY, Recorde: result, Context: context}
	if err = res.triggerDBAction(e.before()); err != nil {
		return err
	}

	if err = context.DB.Set("gorm:order_by_primary_key", "DESC").Find(result).Error; err != nil {
		res.triggerDBAction(e.error(err))
		return err
	}

	err = res.triggerDBAction(e.after())
	return err
}

func (res *Resource) FindOneBasic(db *aorm.DB, id string) (BasicValue, error) {
	return res.CallFindOneBasic(res, db, id)
}

func (res *Resource) CallFindOneBasic(r Resourcer, db *aorm.DB, id string) (BasicValue, error) {
	context := &core.Context{DB: db, ResourceID: id}
	context.Data().Set("skip.fragments", true)
	l := res.Layouts[BASIC_LAYOUT]
	v := l.NewStruct()
	err := res.CallFindOneLayout(r, v, nil, context, l)
	if err != nil {
		return nil, err
	}
	if b, ok := v.(BasicValue); ok {
		return b, nil
	}
	return res.TransformToBasicValueFunc(v), nil
}

func (res *Resource) FindManyBasic(db *aorm.DB, id string) (r []BasicValue, err error) {
	return res.CallFindManyBasic(res, db, id)
}

func (res *Resource) CallFindManyBasic(r Resourcer, db *aorm.DB, id string) (rbv []BasicValue, err error) {
	context := &core.Context{DB: db}
	context.Data().Set("skip.fragments", true)
	l := res.Layouts[BASIC_LAYOUT]
	v := l.NewSlice()
	err = res.CallFindManyLayout(r, v, context, l)
	if err != nil {
		return
	}
	return v.([]BasicValue), nil
}

func (res *Resource) DBDelete(result interface{}, context *core.Context, parent *Parent) error {
	return nil
}

var DB_PARENT_KEY = "qor:resource.parent"

func (res *Resource) DBSave(resourcer Resourcer, result interface{}, context *core.Context, parent *Parent) (err error) {
	originalContext := context
	context = context.Clone()
	var insides []interface{}
	if parent != nil {
		if parent.Index != -1 {
			insides = append(insides, fmt.Sprint("%v[%v]", parent.Inline.FieldName, parent.Index))
		} else {
			insides = append(insides, parent.Inline.FieldName)
		}
	}
	insides = append(insides, res)
	context.SetDB(context.DB.Inside(insides...).Set(DB_PARENT_KEY, parent))
	if res.beforeSaveCallbacks != nil {
		for _, cb := range res.beforeSaveCallbacks {
			if err := cb.Handler(resourcer, result, context, parent); err != nil {
				return err
			}
		}
	}

	if parent != nil {
		for _, cb := range parent.Inline.BeforeSaveCallbacks {
			err = cb(resourcer, result, context, parent)
			if err != nil {
				return err
			}
		}
	}

	e := &DBEvent{Resource: res, Action: E_DB_ACTION_SAVE, Recorde: result, Context: context, OriginalContext: originalContext, Parent: parent}

	if err = res.triggerDBAction(e.before()); err != nil {
		return
	}

	if res.Inlines.Len > 0 {
		value := reflect.ValueOf(result).Elem()
		inlineValues := make([]interface{}, res.Inlines.Len)
		inlineFields := make([]reflect.Value, res.Inlines.Len)

		res.Inlines.Each(func(inline *InlineResourcer) bool {
			inlineValue := value.FieldByIndex(inline.fieldIndex)
			inlineFields[inline.Index] = inlineValue
			inlineValueInterface := inlineValue.Interface()
			inlineValues[inline.Index] = inlineValueInterface
			inlineValue.Set(reflect.Zero(inlineValue.Type()))
			p := &Parent{Parent: parent, Resource: resourcer, Index: -1, Record: result, Inline: inline}
			if inline.Slice {
				// TODO: iterate of slice
			} else {
				err = inline.Resource.DBSave(inline.Resource, inlineValueInterface, context, p)
				if err != nil {
					inlineValue.Set(reflect.ValueOf(inlineValueInterface))
					return false
				}

				reflectedInlineValue := reflect.Indirect(reflect.ValueOf(inlineValueInterface))

				for i, inlineValuePKField := range inline.Resource.GetResource().PrimaryFields {
					// the ID field from inline
					inlineValuePKFieldValue := reflectedInlineValue.FieldByIndex(inlineValuePKField.StructIndex)
					resultPKFieldValue := value.FieldByIndex(inline.keyFieldsIndex[i])
					resultPKFieldValue.Set(inlineValuePKFieldValue)
				}
			}
			return true
		})

		if err != nil {
			return err
		}

		if err = res.triggerDBAction(e); err != nil {
			return
		}

		if err = context.DB.Save(result).Error; err != nil {
			res.triggerDBAction(e.error(err))
		}

		res.Inlines.Each(func(inline *InlineResourcer) bool {
			reflectedInlineValue := reflect.Indirect(reflect.ValueOf(inlineValues[inline.Index]))
			value.FieldByIndex(inline.fieldIndex).Set(reflectedInlineValue.Addr())
			return true
		})
	} else {
		if err = res.triggerDBAction(e); err != nil {
			return
		}
		if err = context.DB.Save(result).Error; err != nil {
			res.triggerDBAction(e.error(err))
		}
	}

	if err == nil && parent != nil {
		for _, cb := range parent.Inline.AfterSaveCallbacks {
			err = cb(resourcer, result, context, parent)
			if err != nil {
				return err
			}
		}
	}

	err = res.triggerDBAction(e.after())
	return
}

func (res *Resource) saveHandler(resourcer Resourcer, result interface{}, context *core.Context) error {
	if (context.GetDB().NewScope(result).PrimaryKeyZero() &&
		res.HasPermission(roles.Create, context)) || // has create permission
		res.HasPermission(roles.Update, context) { // has update permission
		return res.DBSave(resourcer, result, context, nil)
	}
	return roles.ErrPermissionDenied
}

func (res *Resource) deleteHandler(resourcer Resourcer, result interface{}, context *core.Context) error {
	if res.HasPermission(roles.Delete, context) {
		if primaryQuerySQL, primaryParams := res.ToPrimaryQueryParams(context.ResourceID); primaryQuerySQL != "" {
			if !context.DB.First(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...).RecordNotFound() {
				return context.DB.Delete(result).Error
			}
		}
		return aorm.ErrRecordNotFound
	}
	return roles.ErrPermissionDenied
}

// CallSave call save method
func (res *Resource) Save(result interface{}, context *core.Context) error {
	return res.CallSave(res, result, context)
}

// CallSave call save method
func (res *Resource) CallSave(r Resourcer, result interface{}, context *core.Context) error {
	return res.SaveHandler(r, result, context)
}

// CallDelete call delete method
func (res *Resource) Delete(result interface{}, context *core.Context) error {
	return res.CallDelete(res, result, context)
}

// CallDelete call delete method
func (res *Resource) CallDelete(r Resourcer, result interface{}, context *core.Context) error {
	return res.DeleteHandler(r, result, context)
}
