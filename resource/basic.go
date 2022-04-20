package resource

import (
	"github.com/go-aorm/aorm"
)

type ID = aorm.ID

type Basic struct {
	ID    ID
	Label string
	Icon  string
}

func (b Basic) GetID() ID {
	return b.ID
}

func (b Basic) BasicLabel() string {
	return b.Label
}

func (b Basic) BasicIcon() string {
	return b.Icon
}

func (b Basic) String() string {
	return b.BasicLabel()
}

type IDLabeler interface {
	Label() string
	GetID() ID
}

type idLabel struct {
	id    ID
	label string
}

func IDLabel(id ID, label string) IDLabeler {
	return idLabel{id: id, label: label}
}

func (this idLabel) Label() string {
	return this.label
}

func (this idLabel) GetID() ID {
	return this.id
}

type BasicDescriptableValue struct {
	Basic
	Description string
}

func (this BasicDescriptableValue) BasicDescription() string {
	return this.Description
}

func (this BasicDescriptableValue) GetDescription() string {
	return this.Description
}
