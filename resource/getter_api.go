package resource

import "github.com/ecletus/core"

type IconGetter interface {
	GetIcon() string
}

type IconContextGetter interface {
	GetIcon(ctx *core.Context) string
}

type DescriptionGetter interface {
	GetDescription() string
}
