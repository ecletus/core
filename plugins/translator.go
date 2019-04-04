package plugins

import (
	"github.com/ecletus-pkg/locale"
	"github.com/ecletus/plug"
	"github.com/moisespsena-go/i18n-modular/i18nmod"
	"github.com/op/go-logging"
)

type TranslatorPlugin struct {
	TranslatorKey string
	LocaleKey     string
	BackendsKey   []string
	loaded        bool
	log           *logging.Logger
	PreLoad       []func()
	translator    *i18nmod.Translator
	locale        *locale.Locale
}

func (p *TranslatorPlugin) SetLog(log *logging.Logger) {
	p.log = log
}

func (p *TranslatorPlugin) ProvideOptions() []string {
	return []string{p.TranslatorKey}
}

func (p *TranslatorPlugin) RequireOptions() []string {
	return append([]string{p.LocaleKey}, p.BackendsKey...)
}

func (p *TranslatorPlugin) Init(options *plug.Options) error {
	p.locale = options.GetInterface(p.LocaleKey).(*locale.Locale)
	p.translator = i18nmod.NewTranslator()
	p.translator.DefaultLocale = p.locale.Default

	for _, key := range p.BackendsKey {
		be := options.GetInterface(key).(i18nmod.Backend)
		p.translator.AddBackend(be)
	}
	options.Set(p.TranslatorKey, p.translator)
	return nil
}

func (p *TranslatorPlugin) Translator() *i18nmod.Translator {
	return p.translator
}

func (p *TranslatorPlugin) Load() {
	if p.loaded {
		return
	}

	for _, f := range p.PreLoad {
		f()
	}

	if err := p.translator.PreloadAll(); err != nil {
		p.log.Error("Load translations failed: %v", err)
	} else {
		p.loaded = true
	}

}
