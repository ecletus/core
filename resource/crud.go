package resource

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ecletus/core"
	"github.com/ecletus/roles"
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
	Recorde     interface{}
}

func NewCrud(res Resourcer, ctx *core.Context) *CRUD {
	return &CRUD{res: res, context: ctx, dispatchers: []edis.EventDispatcherInterface{res}}
}

func (crud *CRUD) SetDB(DB *aorm.DB) *CRUD {
	crud.context.SetDB(DB)
	return crud
}

func (crud *CRUD) DB() *aorm.DB {
	return crud.context.DB
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
	for _, dis := range dis {
		if dis != nil {
			crud.dispatchers = append(crud.dispatchers, dis)
		}
	}
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

func (crud *CRUD) FindOneLayout(key string, layout ...interface{}) (result interface{}, err error) {
	if len(layout) > 0 {
		crud = crud.SetLayout(layout[0])
	}
	crud = crud.layout.Prepare(crud)
	slice, recorde := crud.res.NewSliceRecord()
	crud.context.ResourceID = key
	if err = crud.FindOne(recorde); err != nil {
		return nil, err
	}
	result = crud.layout.FormatResult(crud, slice)
	result = reflect.ValueOf(result).Index(0).Interface()
	return
}

func (crud *CRUD) FindManyLayout(layout ...interface{}) (result interface{}, err error) {
	if len(layout) > 0 {
		crud = crud.SetLayout(layout[0])
	}
	crud = crud.layout.Prepare(crud)
	slice := crud.res.NewSlicePtr()
	if err = crud.FindMany(slice); err != nil {
		return nil, err
	}
	result = crud.layout.FormatResult(crud, slice)
	return result, nil
}

func (crud *CRUD) FindManyLayoutOrDefault(layout interface{}, defaul ...interface{}) (interface{}, error) {
	return crud.SetLayoutOrDefault(layout, defaul...).FindManyLayout()
}

func (crud *CRUD) FindManyBasic() (result []BasicValue, err error) {
	var resultInterface interface{}
	if resultInterface, err = crud.FindManyLayout(BASIC_LAYOUT); err != nil {
		return nil, err
	}
	return resultInterface.([]BasicValue), nil
}

func (crud *CRUD) FindOneBasic(key string) (result BasicValue, err error) {
	resultInterface, err := crud.FindOneLayout(key, BASIC_LAYOUT)
	if err != nil {
		return nil, err
	}
	return resultInterface.(BasicValue), nil
}

func (crud *CRUD) FindOne(result interface{}, key ...string) (err error) {
	context := crud.context.Clone()
	var (
		primaryQuerySQL string
		primaryParams   []interface{}
	)

	if len(key) > 0 && key[0] != "" {
		primaryQuerySQL, primaryParams = StringToPrimaryQuery(crud.res, key[0])
	} else if crud.metaValues == nil {
		primaryQuerySQL, primaryParams = StringToPrimaryQuery(crud.res, context.ResourceID)
	} else {
		primaryQuerySQL, primaryParams = MetaValuesToPrimaryQuery(crud.res, crud.metaValues)

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
				if fmt.Sprint(destroy.Value) != "0" && core.HasPermission(crud.res, roles.Delete, context) {
					context.DB.Delete(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...)
					return ErrProcessorSkipLeft
				}
			}
		}

		e := NewDBEvent(E_DB_ACTION_FIND_ONE, context)
		e.SetResult(result)

		if err = crud.triggerDBAction(e.before()); err != nil {
			return
		}

		if err = context.DB.First(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...).Error; err != nil {
			crud.triggerDBAction(e.error(err))
			return err
		}

		err = crud.triggerDBAction(e.after())
		return err
	}

	return errors.New("failed to find")
}

func (crud *CRUD) FindMany(result interface{}) (err error) {
	var (
		context = crud.context.Clone()
		e       *DBEvent
	)

	if _, ok := context.DB.Get("qor:getting_total_count"); ok {
		e = NewDBEvent(E_DB_ACTION_COUNT, context.Clone())
		e.SetResult(result)
		if err = crud.triggerDBAction(e.before()); err != nil {
			return err
		}
		if err = e.Context.DB.Count(result).Error; err != nil {
			crud.triggerDBAction(e.error(err))
			return err
		}

		err = crud.triggerDBAction(e.after())
		return err
	}

	e = NewDBEvent(E_DB_ACTION_FIND_MANY, context)
	e.SetResult(result)

	if err = crud.triggerDBAction(e.before()); err != nil {
		return err
	}

	if !context.DB.HasOrder() {
		context.DB = context.DB.Set("gorm:order_by_primary_key", "DESC")
	}

	if err = context.DB.Find(result).Error; err != nil {
		crud.triggerDBAction(e.error(err))
		return err
	}

	err = crud.triggerDBAction(e.after())
	return err
}

func (crud *CRUD) Create(record interface{}) error {
	if crud.context.GetDB().NewScope(record).PrimaryKeyZero() &&
		core.HasPermission(crud.res, roles.Create, crud.context) {
		return crud.CallCreate(record)
	}
	return roles.ErrPermissionDenied
}

func (crud *CRUD) CallCreate(record interface{}) (err error) {
	return crud.callSave(record, E_DB_ACTION_CREATE)
}

func (crud *CRUD) Update(record interface{}) error {
	if !crud.context.GetDB().NewScope(record).PrimaryKeyZero() &&
		core.HasPermission(crud.res, roles.Update, crud.context) {
		return crud.CallUpdate(record)
	}
	return roles.ErrPermissionDenied
}

func (crud *CRUD) CallUpdate(recorde interface{}) (err error) {
	return crud.callSave(recorde, E_DB_ACTION_SAVE)
}

func (crud *CRUD) SaveOrCreate(recorde interface{}) error {
	if crud.context.GetDB().NewScope(recorde).PrimaryKeyZero() {
		if core.HasPermission(crud.res, roles.Create, crud.context) {
			return crud.CallCreate(recorde)
		}
	} else if core.HasPermission(crud.res, roles.Update, crud.context) {
		return crud.CallUpdate(recorde)
	}
	return roles.ErrPermissionDenied
}

func (crud *CRUD) callSave(recorde interface{}, eventName DBActionEvent) (err error) {
	defer crud.recorde(recorde)()
	var context = crud.context.Clone()

	e := NewDBEvent(eventName, context)
	e.SetResult(recorde)

	if err = crud.triggerDBAction(e.before()); err != nil {
		return
	}

	if err = crud.triggerDBAction(e); err != nil {
		return
	}

	if err = context.DB.Save(recorde).Error; err != nil {
		crud.triggerDBAction(e.error(err))
	}

	err = crud.triggerDBAction(e.after())
	return
}

func (crud *CRUD) CallDelete(recorde interface{}) (err error) {
	defer crud.recorde(recorde)()
	if primaryQuerySQL, primaryParams := StringToPrimaryQuery(crud.res, crud.context.ResourceID); primaryQuerySQL != "" {
		db := crud.context.GetDB()
		db = db.First(recorde, append([]interface{}{primaryQuerySQL}, primaryParams...)...)
		if !db.RecordNotFound() {
			e := NewDBEvent(E_DB_ACTION_DELETE, crud.context)
			e.SetResult(recorde)

			if err = crud.triggerDBAction(e.before()); err != nil {
				return
			}

			if err = crud.triggerDBAction(e); err != nil {
				return
			}

			if err = db.Delete(recorde).Error; err != nil {
				crud.triggerDBAction(e.error(err))
			}

			err = crud.triggerDBAction(e.after())
		}
		return
	}
	return aorm.ErrRecordNotFound
}

func (crud *CRUD) Delete(record interface{}) (err error) {
	if core.HasPermission(crud.res, roles.Delete, crud.context) {
		return crud.CallDelete(record)
	}
	return roles.ErrPermissionDenied
}

func (crud *CRUD) triggerDBAction(e *DBEvent) (err error) {
	e.Crud = crud
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

func (crud *CRUD) recorde(r interface{}) func() {
	old := crud.Recorde
	crud.Recorde = r
	return func() {
		crud.Recorde = old
	}
}
