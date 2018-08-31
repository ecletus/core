package resource

import "github.com/aghape/core"

type IconGetter interface {
	GetIcon() string
}

type IconContextGetter interface {
	GetIcon(ctx *core.Context) string
}
