package resource

import (
	"github.com/ecletus/core"
	"github.com/moisespsena-go/aorm"
	"github.com/moisespsena/go-edis"
	"github.com/moisespsena-go/path-helpers"
)

var pkg = path_helpers.GetCalledDir()

type DBActionEvent uint

func (n DBActionEvent) Before() DBActionEvent {
	v := DBActionEvent(BEFORE | (n &^ AFTER))
	return v
}

func (n DBActionEvent) After() DBActionEvent {
	return DBActionEvent(AFTER | (n &^ BEFORE))
}

func (n DBActionEvent) Error() DBActionEvent {
	return DBActionEvent(ERROR | (n &^ AFTER &^ BEFORE))
}

func (n DBActionEvent) Names() []string {
	var s []string

	if (n & E_DB_ACTION_COUNT) != 0 {
		s = append(s, "count")
	}
	if (n & E_DB_ACTION_CREATE) != 0 {
		s = append(s, "create")
	}
	if (n & E_DB_ACTION_DELETE) != 0 {
		s = append(s, "delete")
	}
	if (n & E_DB_ACTION_FIND_MANY) != 0 {
		s = append(s, "findMany")
	}
	if (n & E_DB_ACTION_FIND_ONE) != 0 {
		s = append(s, "findOne")
	}
	if (n & E_DB_ACTION_SAVE) != 0 {
		s = append(s, "save")
	}
	if (n & BEFORE) != 0 {
		for i := range s {
			s[i] = "before." + s[i]
		}
	} else if (n & AFTER) != 0 {
		for i := range s {
			s[i] = "after." + s[i]
		}
	} else if (n & ERROR) != 0 {
		for i := range s {
			s[i] += ".error"
		}
	}
	return s
}

func (n DBActionEvent) Name() string {
	return n.Names()[0]
}

func (n DBActionEvent) FullNames() []string {
	names := n.Names()
	for i := range names {
		names[i] = pkg + ".db." + names[i]
	}
	return names
}

func (n DBActionEvent) FullName() string {
	return n.FullNames()[0]
}

const (
	BEFORE DBActionEvent = 1 << iota
	AFTER
	ERROR
	E_DB_ACTION_COUNT
	E_DB_ACTION_CREATE
	E_DB_ACTION_DELETE
	E_DB_ACTION_FIND_MANY
	E_DB_ACTION_FIND_ONE
	E_DB_ACTION_SAVE
)

type DBEvent struct {
	edis.EventInterface
	Crud    *CRUD
	Action  DBActionEvent
	Context *core.Context
	DBError error
}

func NewDBEvent(action DBActionEvent, ctx *core.Context) *DBEvent {
	return &DBEvent{EventInterface: edis.NewEvent(action.FullName()), Action: action, Context: ctx}
}

func (e *DBEvent) DB() *aorm.DB {
	return e.Context.GetDB()
}

func (e *DBEvent) SetDB(DB *aorm.DB) *DBEvent {
	e.Context.SetDB(DB)
	return e
}

func (e *DBEvent) Resource() Resourcer {
	return e.Crud.res
}

func (e *DBEvent) OriginalContext() *core.Context {
	return e.Crud.context
}

func (e *DBEvent) updateName() {
	result := e.Result()
	e.EventInterface = edis.NewEvent(e.Action.FullName())
	e.SetResult(result)
}

func (e DBEvent) count() *DBEvent {
	e.Action = E_DB_ACTION_COUNT
	e.updateName()
	return &e
}

func (e DBEvent) after() *DBEvent {
	e.Action = e.Action.After()
	e.updateName()
	return &e
}

func (e DBEvent) before() *DBEvent {
	e.Action = e.Action.Before()
	e.updateName()
	return &e
}

func (e DBEvent) save() *DBEvent {
	e.Action = E_DB_ACTION_SAVE
	e.updateName()
	return &e
}

func (e DBEvent) error(err error) *DBEvent {
	e.Action = e.Action.Error()
	e.DBError = err
	return &e
}

func (res *Resource) OnDBActionE(cb func(e *DBEvent) error, action ...DBActionEvent) (err error) {
	return OnDBActionE(res, cb, action...)
}

func (res *Resource) OnDBAction(cb func(e *DBEvent), action ...DBActionEvent) (err error) {
	return OnDBAction(res, cb, action...)
}

func OnDBActionE(dis edis.EventDispatcherInterface, cb func(e *DBEvent) error, action ...DBActionEvent) (err error) {
	cb2 := func(e edis.EventInterface) error {
		return cb(e.(*DBEvent))
	}
	for _, action := range action {
		for _, actionName := range action.FullNames() {
			err = dis.OnE(actionName, cb2)
			if err != nil {
				return err
			}
		}
	}
	return
}

func OnDBAction(dis edis.EventDispatcherInterface, cb func(e *DBEvent), action ...DBActionEvent) (err error) {
	cb2 := func(e edis.EventInterface) {
		cb(e.(*DBEvent))
	}
	for _, action := range action {
		for _, actionName := range action.FullNames() {
			err = dis.OnE(actionName, cb2)
			if err != nil {
				return err
			}
		}
	}
	return
}
