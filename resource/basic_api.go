package resource

import "fmt"

type BasicLabel interface {
	BasicLabel() string
}

type BasicIcon interface {
	BasicIcon() string
}

type BasicValuer interface {
	fmt.Stringer
	BasicLabel
	BasicIcon
	GetID() ID
}

type BasicDescriptableValuer interface {
	BasicValuer
	BasicDescription() string
}
