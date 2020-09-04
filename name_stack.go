package core

import "strings"

type NameFrame struct {
	Key   string
	Name  string
	Namer func() string
	Index int
}

type NameStack []NameFrame

func (this *NameStack) withFrame(frame NameFrame, key ...string) func() {
	for _, frame.Key = range key {
	}
	*this = append(*this, frame)
	return func() {
		*this = (*this)[0 : len(*this)-1]
	}
}

func (this *NameStack) WithName(name string, key ...string) func() {
	return this.withFrame(NameFrame{Name: name}, key...)
}
func (this *NameStack) WithNamer(f func() string, key ...string) func() {
	return this.withFrame(NameFrame{Namer: f}, key...)
}
func (this *NameStack) WithIndex(index int, key ...string) func() {
	return this.withFrame(NameFrame{Index: index}, key...)
}

func (this NameStack) JoinOptions(sep string, indexFormat func(index int) string) string {
	var result = make([]string, len(this))
	for i, el := range this {
		if el.Name != "" {
			result[i] = el.Name
		} else if el.Namer != nil {
			result[i] = el.Namer()
		} else {
			result[i] = indexFormat(el.Index)
		}
	}
	return strings.Join(result, sep)
}

type NameStacker struct {
	Stack       NameStack
	Sep         string
	IndexFormat func(index int) string
}

func (this NameStacker) String() string {
	return this.Stack.JoinOptions(this.Sep, this.IndexFormat)
}

func (this *NameStacker) WithName(name string, key ...string) func() {
	return (&this.Stack).WithName(name, key...)
}
func (this *NameStacker) WithNamer(f func() string, key ...string) func() {
	return (&this.Stack).WithNamer(f, key...)
}
func (this *NameStacker) WithIndex(index int, key ...string) func() {
	return (&this.Stack).WithIndex(index, key...)
}
