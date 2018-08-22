package resource

import (
	"strings"

	"github.com/moisespsena/go-edis"
	"github.com/aghape/core"
)

type DBActionEventName string

func (n DBActionEventName) Before() DBActionEventName {
	return DBActionEventName("before" + strings.ToUpper(string(n)[0:1]) + string(n)[1:])
}

func (n DBActionEventName) After() DBActionEventName {
	return DBActionEventName("after" + strings.ToUpper(string(n)[0:1]) + string(n)[1:])
}

func (n DBActionEventName) Error() DBActionEventName {
	return DBActionEventName(string(n) + "Error")
}

func (n DBActionEventName) String() string {
	return string(n)
}

const (
	E_DB_ACTION_COUNT     DBActionEventName = "count"
	E_DB_ACTION_CREATE    DBActionEventName = "create"
	E_DB_ACTION_DELETE    DBActionEventName = "delete"
	E_DB_ACTION_FIND_MANY DBActionEventName = "findMany"
	E_DB_ACTION_FIND_ONE  DBActionEventName = "findOne"
	E_DB_ACTION_SAVE      DBActionEventName = "save"
)

type DBEvent struct {
	edis.EventInterface
	Resource Resourcer
	Action   DBActionEventName
	Recorde  interface{}
	Context  *core.Context
	OriginalContext  *core.Context
	Parent   *Parent
	DBError  error
}

func (e DBEvent) count() *DBEvent {
	e.Action = E_DB_ACTION_COUNT
	return &e
}

func (e DBEvent) after() *DBEvent {
	e.Action = e.Action.After()
	return &e
}

func (e DBEvent) before() *DBEvent {
	e.Action = e.Action.Before()
	return &e
}

func (e DBEvent) save() *DBEvent {
	e.Action = E_DB_ACTION_SAVE
	return &e
}

func (e DBEvent) error(err error) *DBEvent {
	e.Action = e.Action.Error()
	e.DBError = err
	return &e
}

func (res *Resource) OnDBActionE(cb func(e *DBEvent) error, action ...DBActionEventName) (err error) {
	cb2 := func(e edis.EventInterface) error {
		return cb(e.(*DBEvent))
	}
	for _, actionName := range action {
		err = res.OnE("db:"+actionName.String(), cb2)
		if err != nil {
			return err
		}
	}
	return
}

func (res *Resource) OnDBAction(cb func(e *DBEvent), action ...DBActionEventName) (err error) {
	cb2 := func(e edis.EventInterface) {
		cb(e.(*DBEvent))
	}
	for _, actionName := range action {
		err = res.OnE("db:"+actionName.String(), cb2)
		if err != nil {
			return err
		}
	}
	return
}

func (res *Resource) triggerDBAction(e *DBEvent) error {
	e.EventInterface = edis.NewEvent("db:" + e.Action.String())
	return res.Trigger(e)
}
