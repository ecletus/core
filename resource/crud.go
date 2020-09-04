package resource

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/ecletus/core"
	"github.com/ecletus/roles"
	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena-go/edis"
	errwrap "github.com/moisespsena-go/error-wrap"
)

type CRUD struct {
	DefaultDenyMode bool
	edis.EventDispatcher
	dispatchers []edis.EventDispatcherInterface
	res         Resourcer
	context     *core.Context
	metaValues  *MetaValues
	parent      *CRUD
	layout      LayoutInterface
	Chan        interface{}
	Recorde     interface{}
}

func NewCrud(res Resourcer, ctx *core.Context) *CRUD {
	ctx = res.ContextSetup(ctx)
	return &CRUD{res: res, context: ctx, dispatchers: []edis.EventDispatcherInterface{res}}
}

func (this *CRUD) SetDB(DB *aorm.DB) *CRUD {
	this.context.SetDB(DB)
	return this
}

func (this *CRUD) DB() *aorm.DB {
	return this.context.DB()
}

func (this *CRUD) Resource() Resourcer {
	return this.res
}

func (this *CRUD) Context() *core.Context {
	return this.context
}

func (this *CRUD) Layout() LayoutInterface {
	return this.layout
}

func (this *CRUD) Parent() *CRUD {
	return this.parent
}

func (this *CRUD) MetaValues() *MetaValues {
	return this.metaValues
}

func (this *CRUD) Dispatchers() []edis.EventDispatcherInterface {
	return this.dispatchers
}

func (this *CRUD) Dispatcher(dis ...edis.EventDispatcherInterface) *CRUD {
	this = this.sub()
	for _, dis := range dis {
		if dis != nil {
			this.dispatchers = append(this.dispatchers, dis)
		}
	}
	return this
}

func (this *CRUD) sub() *CRUD {
	sub := &(*this)
	sub.dispatchers = this.dispatchers[:]
	sub.parent = this
	return sub
}

func (this *CRUD) SetLayout(layout interface{}) *CRUD {
	var l LayoutInterface
	if layoutName, ok := layout.(string); ok {
		l = this.res.GetLayout(layoutName)
	} else {
		l = layout.(LayoutInterface)
	}
	this = this.sub()
	this.layout = l
	return this
}

func (this *CRUD) SetLayoutOrDefault(layout interface{}, defaul ...interface{}) *CRUD {
	var l LayoutInterface
	if len(defaul) == 0 {
		defaul = append(defaul, DEFAULT_LAYOUT)
	}
	for _, lt := range append([]interface{}{layout}, defaul...) {
		if layoutName, ok := lt.(string); ok {
			l = this.res.GetLayout(layoutName)
			if l != nil {
				break
			}
		} else {
			l = layout.(LayoutInterface)
			break
		}
	}
	this = this.sub()
	this.layout = l
	return this
}

func (this *CRUD) SetContext(ctx *core.Context) *CRUD {
	this = this.sub()
	this.context = ctx
	return this
}

func (this *CRUD) SetMetaValues(metaValues *MetaValues) *CRUD {
	this = this.sub()
	this.metaValues = metaValues
	return this
}

func (this *CRUD) FindOneLayout(key aorm.ID, layout ...interface{}) (result interface{}, err error) {
	if len(layout) > 0 {
		this = this.SetLayout(layout[0])
	}
	this = this.layout.Prepare(this)
	slice, recorde := this.res.NewSliceRecord()
	if err = this.FindOne(recorde, key); err != nil {
		return nil, err
	}
	result = this.layout.FormatResult(this, slice)
	result = reflect.ValueOf(result).Index(0).Interface()
	return
}

func (this *CRUD) FindManyLayout(layout ...interface{}) (result interface{}, err error) {
	if len(layout) > 0 {
		this = this.SetLayout(layout[0])
	}
	this = this.layout.Prepare(this)
	if this.Chan == nil {
		slice := this.res.NewSlicePtr()
		if err = this.FindMany(slice); err != nil {
			return nil, err
		}
		result = this.layout.FormatResult(this, slice)
		return result, nil
	} else {
		err = this.FindMany(this.Chan)
		return
	}
}

func (this *CRUD) FindManyLayoutOrDefault(layout interface{}, defaul ...interface{}) (interface{}, error) {
	return this.SetLayoutOrDefault(layout, defaul...).FindManyLayout()
}

func (this *CRUD) FindManyBasic() (result []BasicValuer, err error) {
	var resultInterface interface{}
	if resultInterface, err = this.FindManyLayout(BASIC_LAYOUT); err != nil {
		return nil, err
	}
	return resultInterface.([]BasicValuer), nil
}

func (this *CRUD) FindOneBasic(key aorm.ID) (result BasicValuer, err error) {
	resultInterface, err := this.FindOneLayout(key, BASIC_LAYOUT)
	if err != nil {
		return nil, err
	}
	return resultInterface.(BasicValuer), nil
}

func (this *CRUD) FindOne(result interface{}, key ...aorm.ID) (err error) {
	var (
		context         = this.context.Clone()
		primaryQuerySQL string
		primaryParams   []interface{}
		DB              = context.DB()
	)

	var hasKey bool

	if this.res.HasKey() {
		if len(key) > 0 && key[0] != nil {
			if primaryQuerySQL, primaryParams, err = IdToPrimaryQuery(this.context, this.res, false, key[0]); err != nil {
				return
			}
		} else if this.metaValues == nil {
			if !context.ResourceID.IsZero() {
				if primaryQuerySQL, primaryParams, err = IdToPrimaryQuery(this.context, this.res, false, context.ResourceID); err != nil {
					return
				}
			}
		} else if primaryQuerySQL, primaryParams, err = MetaValuesToPrimaryQuery(this.context, this.res, this.metaValues, false); err != nil {
			return
		}
		if primaryQuerySQL != "" {
			if this.metaValues != nil {
				if destroy := this.metaValues.Get("_destroy"); destroy != nil {
					if fmt.Sprint(destroy.Value) != "0" && this.res.HasPermission(roles.Delete, context).Ok(!this.DefaultDenyMode) {
						DB.Delete(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...)
						return ErrProcessorSkipLeft
					}
				}
			}
			hasKey = true
		}
	}

	if hasKey {
		e := NewDBEvent(E_DB_ACTION_FIND_ONE, context)
		e.SetResult(result)

		if err = this.triggerDBAction(e.before()); err != nil {
			return
		}

		DB = context.DB()
		if err = DB.First(result, append([]interface{}{primaryQuerySQL}, primaryParams...)...).Error; err != nil {
			this.triggerDBAction(e.error(err))
			return
		}
		err = this.triggerDBAction(e.after())
	} else if !this.res.HasKey() {
		e := NewDBEvent(E_DB_ACTION_FIND_ONE, context)
		e.SetResult(result)

		if err = this.triggerDBAction(e.before()); err != nil {
			return
		}

		DB = context.DB()
		if err = DB.First(result).Error; err != nil {
			this.triggerDBAction(e.error(err))
			return
		}
		err = this.triggerDBAction(e.after())
	} else {
		return errors.New("failed to find: no key found")
	}

	return
}

func (this *CRUD) Count(result interface{}) (err error) {
	var (
		context = this.context.Clone()
		e       *DBEvent
	)
	e = NewDBEvent(E_DB_ACTION_COUNT, context.Clone())
	e.SetResult(result)
	if err = this.triggerDBAction(e.before()); err != nil {
		return err
	}
	if err = e.Context.DB().Count(result).Error; err != nil {
		this.triggerDBAction(e.error(err))
		return err
	}

	err = this.triggerDBAction(e.after())
	return err
}

func (this *CRUD) FindMany(result interface{}) (err error) {
	var (
		context = this.context.Clone()
		e       *DBEvent
	)

	if len(context.ExcludeResourceID) > 0 {
		context.SetRawDB(context.DB().Where(aorm.InID(context.ExcludeResourceID...).Exclude()))
	}

	if _, ok := context.DB().Get("qor:getting_total_count"); ok {
		e = NewDBEvent(E_DB_ACTION_COUNT, context.Clone())
		e.SetResult(result)
		if err = this.triggerDBAction(e.before()); err != nil {
			return err
		}
		if err = context.DB().Count(result).Error; err != nil {
			this.triggerDBAction(e.error(err))
			return err
		}

		err = this.triggerDBAction(e.after())
		return err
	}

	e = NewDBEvent(E_DB_ACTION_FIND_MANY, context)
	e.SetResult(result)

	if err = this.triggerDBAction(e.before()); err != nil {
		return err
	}
	DB := e.Context.DB()
	if !DB.HasOrder() {
		DB = DB.Set("gorm:order_by_primary_key", "DESC")
	}

	if err = DB.Find(result).Error; err != nil {
		this.triggerDBAction(e.error(err))
		return err
	}

	err = this.triggerDBAction(e.after())
	return err
}

func (this *CRUD) Create(record interface{}) error {
	if this.HasPermission(roles.Create) {
		return this.CallCreate(record)
	}
	return roles.ErrPermissionDenied
}

func (this *CRUD) CallCreate(record interface{}) (err error) {
	return this.callSave(record, E_DB_ACTION_CREATE)
}

func (this *CRUD) Update(record interface{}, old ...interface{}) error {
	if this.HasPermission(roles.Update) {
		if this.Resource().IsSingleton() || !aorm.ZeroIdOf(record) {
			return this.CallUpdate(record, old...)
		}
		return errors.New("BID not set")
	}
	return roles.ErrPermissionDenied
}

func (this *CRUD) CallUpdate(recorde interface{}, old ...interface{}) (err error) {
	return this.callSave(recorde, E_DB_ACTION_UPDATE, old...)
}

func (this *CRUD) SaveOrCreate(recorde interface{}) error {
	if aorm.ZeroIdOf(recorde) {
		if this.HasPermission(roles.Create) {
			return this.CallCreate(recorde)
		}
	} else if this.HasPermission(roles.Update) {
		return this.CallUpdate(recorde)
	}
	return roles.ErrPermissionDenied
}

func (this *CRUD) callSave(recorde interface{}, eventName DBActionEvent, old ...interface{}) (err error) {
	defer this.recorde(recorde)()
	var (
		context = this.context.Clone()
		e       = NewDBEvent(eventName, context)
		DB      = context.DB()
	)

	e.SetResult(recorde)

	for _, e.old = range old {
	}

	if err = this.triggerDBAction(e.before()); err != nil {
		return
	}

	if err = this.triggerDBAction(e); err != nil {
		return
	}

	DB = e.Context.DB()
	if eventName == E_DB_ACTION_CREATE {
		err = DB.Create(recorde).Error
	} else if len(old) > 0 && old[0] != nil {
		//.Where(old[0])
		db := DB.Model(recorde).Opt(aorm.OptStoreBlankField())
		err = db.Update(recorde).Error
	} else {
		err = DB.Save(recorde).Error
	}

	if err != nil {
		if d := aorm.GetDuplicateUniqueIndexError(err); d != nil {
			return DuplicateUniqueIndexError{d, recorde, this.res}
		}
		this.triggerDBAction(e.error(err))
		return
	}

	err = this.triggerDBAction(e.after())
	return
}

func (this *CRUD) CallDelete(recorde interface{}) (err error) {
	var (
		primaryQuerySQL string
		primaryParams   []interface{}
	)
	defer this.recorde(recorde)()
	if primaryQuerySQL, primaryParams, err = IdToPrimaryQuery(this.context, this.res, false, this.context.ResourceID); err != nil {
		return
	} else {
		db := this.context.DB()
		db = db.First(recorde, append([]interface{}{primaryQuerySQL}, primaryParams...)...)
		if !db.RecordNotFound() {
			e := NewDBEvent(E_DB_ACTION_DELETE, this.context)
			e.SetResult(recorde)

			if err = this.triggerDBAction(e.before()); err != nil {
				return
			}

			if err = this.triggerDBAction(e); err != nil {
				return
			}

			if err = db.Delete(recorde).Error; err != nil {
				this.triggerDBAction(e.error(err))
			}

			err = this.triggerDBAction(e.after())
		}
		return
	}
	return aorm.ErrRecordNotFound
}

func (this *CRUD) Delete(record interface{}) (err error) {
	if this.HasPermission(roles.Delete) {
		return this.CallDelete(record)
	}
	return roles.ErrPermissionDenied
}

func (this *CRUD) triggerDBAction(e *DBEvent) (err error) {
	e.Crud = this
	if err = this.Trigger(e); err != nil {
		return err
	}

	for i, d := range this.dispatchers {
		if err = d.Trigger(e); err != nil {
			return errwrap.Wrap(err, "Dispatcher %d", i)
		}
	}
	return nil
}

func (this *CRUD) recorde(r interface{}) func() {
	old := this.Recorde
	this.Recorde = r
	return func() {
		this.Recorde = old
	}
}

func (this *CRUD) HasPermission(mode roles.PermissionMode) bool {
	return this.res.HasPermission(mode, this.context).Ok(!this.DefaultDenyMode)
}
