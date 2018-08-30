package core

type BreadcrumberFunc func(ctx *Context) []Breadcrumb

func (f BreadcrumberFunc) Breadcrumbs(ctx *Context) []Breadcrumb {
	return f(ctx)
}

type Breadcrumber interface {
	Breadcrumbs(ctx *Context) (crumbs []Breadcrumb)
}

type Breadcrumb interface {
	URI(context *Context) string
	Icon() string
	Label() string
}

type BreadcrumbProxy struct {
	uri   string
	label string
	icon  string
}

func (b BreadcrumbProxy) URI(context *Context) string {
	return b.uri
}
func (b BreadcrumbProxy) Label() string {
	return b.label
}
func (b BreadcrumbProxy) Icon() string {
	return b.icon
}

func NewBreadcrumb(uri, label string, icon ...string) *BreadcrumbProxy {
	if len(icon) == 0 {
		icon = append(icon, "")
	}
	return &BreadcrumbProxy{uri, label, icon[0]}
}

type Breadcrumbs struct {
	Items     []Breadcrumb
	afterNext []Breadcrumb
}

func (b *Breadcrumbs) Append(breadcrumbs ...Breadcrumb) {
	b.Items = append(b.Items, append(breadcrumbs, b.afterNext...)...)
	b.afterNext = []Breadcrumb{}
}

func (b *Breadcrumbs) AfterNext(breadcrumbs ...Breadcrumb) {
	b.afterNext = append(b.afterNext, breadcrumbs...)
}

func (b *Breadcrumbs) IsEmpty() bool {
	return len(b.Items) == 0
}

func (b *Breadcrumbs) ItemsWithoutLast() (items []Breadcrumb) {
	if len(b.Items) == 0 {
		return
	}
	return b.Items[0 : len(b.Items)-1]
}

func (b *Breadcrumbs) Last() Breadcrumb {
	if len(b.Items) == 0 {
		return nil
	}
	return b.Items[len(b.Items)-1]
}
