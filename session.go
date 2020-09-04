package core

import (
	"context"

	"github.com/ecletus/session"
)

const sessionManagerKey = "qor:session_manager"

func (this *Context) SessionManager() session.RequestSessionManager {
	return GetSessionManager(this)
}

func (this *Context) SetSessionManager(manager session.RequestSessionManager) *Context {
	return this.SetValue(sessionManagerKey, manager)
}

func GetSessionManager(ctx context.Context) session.RequestSessionManager {
	if sm := ctx.Value(sessionManagerKey); sm != nil {
		return sm.(session.RequestSessionManager)
	}
	return nil
}

func SetSessionManager(parent context.Context, sm session.RequestSessionManager) context.Context {
	return context.WithValue(parent, sessionManagerKey, sm)
}
