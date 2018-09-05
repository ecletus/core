package plugins

import (
	"github.com/moisespsena/go-i18n-modular/i18nmod"
	"github.com/aghape/plug"
	"github.com/aghape/core"
)

type ContextFactoryPlugin struct {
	TranslatorKey, ContextFactoryKey string
}

func (p *ContextFactoryPlugin) RequireOptions() []string {
	return []string{p.TranslatorKey}
}

func (p *ContextFactoryPlugin) ProvideOptions() []string {
	return []string{p.ContextFactoryKey}
}

func (p *ContextFactoryPlugin) Init(options *plug.Options) {
	translator := options.GetInterface(p.TranslatorKey).(*i18nmod.Translator)
	cf := core.NewContextFactory(translator)
	options.Set(p.ContextFactoryKey, cf)
}