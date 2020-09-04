package core

import "github.com/ecletus/roles"

type PermissionModeProvider interface {
	Provides() roles.Roles
}

type SitePermissionModeProvider struct {
	Site    *Site
	Std     func() roles.Roles
	Default func() roles.Roles
}

func (this SitePermissionModeProvider) Provides() (modes roles.Roles) {
	modes = roles.NewRoles()
	if this.Std != nil {
		modes = this.Std()
	}
	if cfg, ok := this.Site.GetConfig("roles"); ok {
		for _, v := range cfg.([]string) {
			modes.Append(v)
		}
	} else if this.Default != nil {
		modes.Merge(this.Default())
	} else if modes.Len() == 0 {
		modes = roles.Global.Roles()
	}
	return
}
