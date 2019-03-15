package resource

import "fmt"

type BasicLabel interface {
	BasicLabel() string
}
type BasicIcon interface {
	BasicIcon() string
}

type BasicValue interface {
	fmt.Stringer
	BasicLabel
	BasicIcon
	GetID() string
}

type Basic struct {
	ID    string
	Label string
	Icon  string
}

func (b *Basic) GetID() string {
	return b.ID
}

func (b *Basic) BasicLabel() string {
	return b.Label
}

func (b *Basic) BasicIcon() string {
	return b.Icon
}

func (b *Basic) String() string {
	return b.BasicLabel()
}
