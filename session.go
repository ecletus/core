package core

import (
	"github.com/ecletus/session"
)

const sessionManagerKey = "qor:session_manager"

func (context *Context) SessionManager() session.RequestSessionManager {
	v := context.Data().Get(sessionManagerKey)
	if v == nil {
		return nil
	}
	return v.(session.RequestSessionManager)
}

func (context *Context) SetSessionManager(manager session.RequestSessionManager) *Context {
	context.Data().Set(sessionManagerKey, manager)
	return context
}
