package resource

import (
	"errors"
	"fmt"

	"github.com/aghape/core"
	"github.com/aghape/roles"
	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena/go-edis"
	"github.com/moisespsena/go-error-wrap"
)

type CRUD struct {
	edis.EventDispatcher
	dispatchers []edis.EventDispatcherInterface
	res         Resourcer
	context     *core.Context
	metaValues  *MetaValues
	parent      *CRUD
	layout      LayoutInterface
}

func NewCrud(res Resourcer, ctx *core.Context) *CRUD {
	return &CRUD{res: res, context: ctx}
}

func (crud *CRUD) Resource() Resourcer {
	return crud.res
}

func (crud *CRUD) Context() *core.Context {
	return crud.context
}

func (crud *CRUD) Layout() LayoutInterface {
	return crud.layout
}

func (crud *CRUD) Parent() *CRUD {
	return crud.parent
}

func (crud *CRUD) MetaValues() *MetaValues {
	return crud.metaValues
}

func (crud *CRUD) Dispatchers() []edis.EventDispatcherInterface {
	return crud.dispatchers
}

func (crud *CRUD) Dispatcher(dis ...edis.EventDispatcherInterface) *CRUD {
	crud = crud.sub()
	crud.dispatchers = append(crud.dispatchers, dis...)
	return crud
}

func (crud *CRUD) sub() *CRUD {
	sub := &(*crud)
	sub.dispatchers = crud.dispatchers[:]
	sub.parent = crud
	return sub
}

func (crud *CRUD) SetLayout(layout interface{}) *CRUD {
	var l LayoutInterface
	if layoutName, ok := layout.(string); ok {
		l = crud.res.GetLayout(layoutName)
	} else {
		l = layout.(LayoutInterface)
	}
	crud = crud.sub()
	crud.layout = l
	return crud
}

func (crud *CRUD) SetLayoutOrDefault(layout interface{}, defaul ...interface{}) *CRUD {
	var l LayoutInterface
	if len(defaul) == 0 {
		defaul = append(defaul, DEFAULT_LAYOUT)
	}
	for _, lt := range append([]interface{}{layout}, defaul...) {
		if layoutName, ok := lt.(string); ok {
			l = crud.res.GetLayout(layoutName)
			if l != nil {
				break
			}
		} else {
			l = layout.(LayoutInterface)
			break
		}
	}
	crud = crud.sub()
	crud.layout = l
	return crud
}

func (crud *CRUD) SetContext(ctx *core.Context) *CRUD {
	crud = crud.sub()
	crud.context = ctx
	return crud
}

func (crud *CRUD) SetMetaValues(metaValues *MetaValues) *CRUD {
	crud = crud.sub()
	crud.metaValues = metaValues
	return crud
}

func (crud *CRUD) FindOneLayout(layout ...interface{}) (interface{}, error) {
	if len(layout) > 0 {
		crud = crud.SetLayout(layout)
	}
	result := crud.layout.NewStruct()
	if err := crud.FindOne(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (crud *CRUD) FindManyLayout(layout ...interface{}) (interface{}, error) {
	if len(layout) > 0 {
		crud = crud.SetLayout(layout)
	}
	result := crud.layout.NewSlice()
	if err := crud.FindMany(result); err != nil {
		return nil, err
	}
	return result, nil
}

func (crud *CRUD) FindManyLayoutOrDefault(layout interface{}, defaul ...interface{}) (interface{}, error) {
	return crud.SetLayoutOrDefault(layout, defaul...).FindManyLayout()
}

func (crud *CRUD) FindManyBasic() (result []BasicValue, err error) {
	crud = crud.SetLayout(crud.res.GetLayout(BASIC_LAYOUT))
	result = crud.layout.NewSlice().([]BasicValue)
	if err = crud.FindMany(nil); err != nil {
		return nil, err
	}
	return
}

func (crud *CRUD) FindOneBasic(key string) (result BasicValue, err error) {
	crud = crud.SetLayout(crud.res.GetLayout(BASIC_LAYOUT))
	result = crud.layout.NewStruct().(BasicValue)
	crud.context.ResourceID = key
	if err = crud.FindOne(result); err != nil {
		return nil, err
	}
	return
}

func (crud *CRUD) FindOne(result interface{}) (err error) {
	originalContext := crud.context
	context := originalContext.Clone()
	var (
		primaryQuerySQL string
		primaryParams   []interface{}
	)

	if crud.metaValues == nil {
		primaryQuerySQL, primaryParams = ToPrimaryQueryParams(crud.res, context.ResourceID)
	} else {
		primaryQuerySQL, primaryParams = ToPrimaryQueryParamsFromMetaValue(crud.res, crud.metaValues)

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
		if crud.metaValues != nil {
			if destroy := crud.metaValues.Get("_destroy"); destroy != nil {
				if fmt.Sprint(destroy.Value) != "0" && crud.res.HasPermission(roles.Delete, context) {
					context.DB.Delete(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...)
					return ErrProcessorSkipLeft
				}
			}
		}

		e := &DBEvent{Resource: crud.res, Action: E_DB_ACTION_FIND_ONE, Recorde: result, Context: context, OriginalContext: originalContext}
		if err = crud.triggerDBAction(e.before()); err != nil {
			return
		}

		if err = context.DB.First(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...).Error; err != nil {
			crud.triggerDBAction(e.error(err))
			return err
		}

		if crud.layout != nil {
			crud.layout.FormatResult(crud, []interface{}{result})
		}

		err = crud.triggerDBAction(e.after())
		return err
	}

	return errors.New("failed to find")
}

func (crud *CRUD) FindMany(result interface{}) (err error) {
	var (
		originalContext = crud.context
		context         = originalContext.Clone()
		res             = crud.res
	)

	var e *DBEvent

	if crud.layout != nil {
		crud = crud.layout.Prepare(crud)
	}

	if _, ok := context.DB.Get("qor:getting_total_count"); ok {
		e = &DBEvent{Resource: res, Action: E_DB_ACTION_COUNT, Recorde: result, Context: context, OriginalContext: originalContext}
		if err = crud.triggerDBAction(e.before()); err != nil {
			return err
		}
		if err = context.DB.Count(result).Error; err != nil {
			crud.triggerDBAction(e.error(err))
			return err
		}

		err = crud.triggerDBAction(e.after())
		return err
	}

	e = &DBEvent{Resource: res, Action: E_DB_ACTION_FIND_MANY, Recorde: result, Context: context}
	if err = crud.triggerDBAction(e.before()); err != nil {
		return err
	}

	if err = context.DB.Set("gorm:order_by_primary_key", "DESC").Find(result).Error; err != nil {
		crud.triggerDBAction(e.error(err))
		return err
	}

	if crud.layout != nil {
		crud.layout.FormatResult(crud, result)
	}

	err = crud.triggerDBAction(e.after())
	return err
}

func (crud *CRUD) Create(record interface{}) error {
	if crud.context.GetDB().NewScope(record).PrimaryKeyZero() &&
		crud.res.HasPermission(roles.Create, crud.context) {
		return crud.CallCreate(record)
	}
	return roles.ErrPermissionDenied
}

func (crud *CRUD) CallCreate(record interface{}) (err error) {
	return crud.callSave(record, E_DB_ACTION_CREATE)
}

func (crud *CRUD) Update(record interface{}) error {
	if !crud.context.GetDB().NewScope(record).PrimaryKeyZero() &&
		crud.res.HasPermission(roles.Update, crud.context) {
		return crud.CallUpdate(record)
	}
	return roles.ErrPermissionDenied
}

func (crud *CRUD) CallUpdate(record interface{}) (err error) {
	return crud.callSave(record, E_DB_ACTION_SAVE)
}

func (crud *CRUD) SaveOrCreate(record interface{}) error {
	if crud.context.GetDB().NewScope(record).PrimaryKeyZero() {
		if crud.res.HasPermission(roles.Create, crud.context) {
			return crud.CallCreate(record)
		}
	} else if crud.res.HasPermission(roles.Update, crud.context) {
		return crud.CallUpdate(record)
	}
	return roles.ErrPermissionDenied
}

func (crud *CRUD) callSave(record interface{}, eventName DBActionEventName) (err error) {
	var (
		originalContext = crud.context
		context         = originalContext.Clone()
		res             = crud.res
	)

	var insides []interface{}
	insides = append(insides, res)

	e := &DBEvent{Resource: res, Action: eventName, Recorde: record, Context: context, OriginalContext: originalContext}

	if err = crud.triggerDBAction(e.before()); err != nil {
		return
	}

	if err = crud.triggerDBAction(e); err != nil {
		return
	}

	if err = context.DB.Save(record).Error; err != nil {
		crud.triggerDBAction(e.error(err))
	}

	err = crud.triggerDBAction(e.after())
	return
}

func (crud *CRUD) CallDelete(record interface{}) (err error) {
	if primaryQuerySQL, primaryParams := ToPrimaryQueryParams(crud.res, crud.context.ResourceID); primaryQuerySQL != "" {
		if db := crud.context.GetDB(); !db.First(record, append([]interface{}{primaryQuerySQL}, primaryParams...)...).RecordNotFound() {
			e := &DBEvent{Resource: crud.res, Action: E_DB_ACTION_DELETE, Recorde: record, Context: crud.context, OriginalContext: crud.context}

			if err = crud.triggerDBAction(e.before()); err != nil {
				return
			}

			if err = crud.triggerDBAction(e); err != nil {
				return
			}

			if err = db.Delete(record).Error; err != nil {
				crud.triggerDBAction(e.error(err))
			}

			err = crud.triggerDBAction(e.after())
		}
		return
	}
	return aorm.ErrRecordNotFound
}

func (crud *CRUD) Delete(record interface{}) (err error) {
	if crud.res.HasPermission(roles.Delete, crud.context) {
		return crud.CallDelete(record)
	}
	return roles.ErrPermissionDenied
}

func (crud *CRUD) triggerDBAction(e *DBEvent) (err error) {
	e.EventInterface = edis.NewEvent("db:" + e.Action.String())
	if err = crud.Trigger(e); err != nil {
		return err
	}

	for i, d := range crud.dispatchers {
		if err = d.Trigger(e); err != nil {
			return errwrap.Wrap(err, "Dispatcher %d", i)
		}
	}
	return nil
}
