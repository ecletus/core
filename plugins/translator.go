package plugins

import (
	"github.com/moisespsena/go-error-wrap"
	"github.com/moisespsena/go-i18n-modular/i18nmod"
	"github.com/aghape/plug"
)

type TranslatorPlugin struct {
	TranslatorKey string
	BackendsKey   []string
}

func (p *TranslatorPlugin) ProvideOptions() []string {
	return []string{p.TranslatorKey}
}

func (p *TranslatorPlugin) RequireOptions() []string {
	return p.BackendsKey
}

func (p *TranslatorPlugin) Init(options *plug.Options) error {
	translator := i18nmod.NewTranslator()
	for _, key := range p.BackendsKey {
		be := options.GetInterface(key).(i18nmod.Backend)
		translator.AddBackend(be)
	}
	err := translator.PreloadAll()
	if err != nil {
		return errwrap.Wrap(err, "Preload Translations")
	}
	options.Set(p.TranslatorKey, translator)
	return nil
}
