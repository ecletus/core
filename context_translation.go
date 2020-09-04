package core

import "github.com/moisespsena/template/html/template"

type I18nLabelPair interface {
	GetLabelPair() (keys []string, defaul string)
}

func (this *Context) Tt(o I18nLabelPair) (r template.HTML) {
	return template.HTML(this.TtS(o))
}

func (this *Context) TtS(o I18nLabelPair) (r string) {
	keys, defaul := o.GetLabelPair()
	for _, key := range keys[0 : len(keys)-1] {
		if r = this.Ts(key, "<nil>"); r != "<nil>" {
			return
		}
	}
	return this.Ts(keys[len(keys)-1], defaul)
}
