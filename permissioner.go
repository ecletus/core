package core

import "github.com/ecletus/roles"

type Permissioner interface {
	HasPermissionE(mode roles.PermissionMode, ctx *Context) (ok bool, err error)
}

func HasPermission(permissioner Permissioner, mode roles.PermissionMode, ctx *Context) (ok bool) {
	ok, _ = permissioner.HasPermissionE(mode, ctx)
	return
}

func HasPermissionDefault(defaul bool, permissioner Permissioner, mode roles.PermissionMode, ctx *Context) (ok bool) {
	ok, _ = HasPermissionDefaultE(defaul, permissioner, mode, ctx)
	return
}

func HasPermissionDefaultE(defaul bool, permissioner Permissioner, mode roles.PermissionMode, ctx *Context) (ok bool, err error) {
	ok, err = permissioner.HasPermissionE(mode, ctx)
	if roles.IsDefaultPermission(err) {
		ok = defaul
	}
	return
}
