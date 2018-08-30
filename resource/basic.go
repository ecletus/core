package resource

import "fmt"

type BasicValue interface {
	fmt.Stringer
	GetID() string
	BasicLabel() string
	BasicIcon() string
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
