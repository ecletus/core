package core

import "github.com/ecletus/roles"

type Permissioner interface {
	HasPermission(mode roles.PermissionMode, ctx *Context) roles.Perm
}

func RolePermissioner(permissioner roles.Permissioner) Permissioner {
	return NewPermissioner(func(mode roles.PermissionMode, ctx *Context) (perm roles.Perm) {
		return permissioner.HasPermission(ctx, mode, ctx.Roles.Interfaces()...)
	})
}

type PermissionerFunc func(mode roles.PermissionMode, ctx *Context) roles.Perm

func (f PermissionerFunc) HasPermission(mode roles.PermissionMode, ctx *Context) roles.Perm {
	return f(mode, ctx)
}

func NewPermissioner(f func(mode roles.PermissionMode, ctx *Context) (perm roles.Perm)) Permissioner {
	return PermissionerFunc(f)
}

func Permissioners(p ...Permissioner) Permissioner {
	var result permissioners
	for _, p := range p {
		if p == nil {
			continue
		}
		switch t := p.(type) {
		case permissioners:
			result = append(result, t...)
		default:
			result = append(result, p)
		}
	}
	if len(result) == 1 {
		return result[0]
	}
	return result
}

type permissioners []Permissioner

func (this permissioners) HasPermission(mode roles.PermissionMode, ctx *Context) (perm roles.Perm) {
	for _, p := range this {
		if perm = p.HasPermission(mode, ctx); perm != roles.UNDEF {
			return
		}
	}
	return
}

type allowedPermissioners []Permissioner

func (this allowedPermissioners) HasPermission(mode roles.PermissionMode, ctx *Context) (perm roles.Perm) {
	for _, p := range this {
		if !p.HasPermission(mode, ctx).Allow() {
			return roles.DENY
		}
	}
	return roles.ALLOW
}

func AllowedPermissioners(p ...Permissioner) Permissioner {
	permr := Permissioners(p...)
	if items, ok := permr.(permissioners); ok {
		return allowedPermissioners(items)
	}
	return permr
}

type ContextPermissioner interface {
	HasContextPermission(mode roles.PermissionMode, ctx *Context) (perm roles.Perm)
}

type RecordPermissioner interface {
	HasRecordPermission(mode roles.PermissionMode, ctx *Context, record interface{}) (perm roles.Perm)
}

type RecordPermissionerFunc func(mode roles.PermissionMode, ctx *Context, record interface{}) roles.Perm

func (f RecordPermissionerFunc) HasRecordPermission(mode roles.PermissionMode, ctx *Context, record interface{}) roles.Perm {
	return f(mode, ctx, record)
}

func NewRecordPermissioner(f func(mode roles.PermissionMode, ctx *Context, record interface{}) (perm roles.Perm)) RecordPermissioner {
	return RecordPermissionerFunc(f)
}
