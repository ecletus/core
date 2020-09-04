package core

import (
	"strings"

	"github.com/apex/log"
	"github.com/moisespsena-go/getters"
	"github.com/moisespsena-go/logging"
	logging_helpers "github.com/moisespsena-go/logging-helpers"
	"github.com/moisespsena-go/maps"
	"github.com/moisespsena-go/middleware"
	"github.com/moisespsena-go/stringvar"

	iocommon "github.com/moisespsena-go/io-common"
)

func (this *Site) initLogger() {
	var mapSI maps.MapSI
	if !mapSI.ReadKey(this.Config(), "log") {
		return
	}

	var cfg logging_helpers.ModuleLoggingConfig
	if err := mapSI.CopyTo(&cfg); err != nil {
		log.Errorf("[%s] logging failed", this.Name())
	}
	siteLog := logging.GetOrCreateLogger("site:" + this.Name())
	logging.SetLogLevel(siteLog, cfg.GetLevel(logging.INFO), "site:"+this.Name())

	if backends := cfg.BackendPrinter(); len(backends) > 0 {
		func(backends ...logging.BackendPrintCloser) logging.Printer {
			var bce = make([]logging.Backend, len(backends))
			for i, bc := range backends {
				bce[i] = bc
			}
			for _, bce := range backends {
				this.OnDestroy(iocommon.MustCloser(bce))
			}
			return logging.MultiLogger(bce...).(logging.Printer)
		}(backends...)
		this.Log = siteLog
	}
	return
}

func (this *Site) RequestLogger(key string) (fmtr middleware.LogAndPanicFormatter) {
	var mapSI maps.MapSI
	if !mapSI.ReadKey(this.Config(), strings.Split(key, "/")) {
		return
	}

	var cfg logging_helpers.ModuleLoggingConfig
	if err := mapSI.CopyTo(&cfg); err != nil {
		log.Errorf("[%s] %q failed", this.Name(), key)
	}

	svar := stringvar.New(
		"SITE_NAME",
		this.name,
		"SITE_ROOT",
		this.systemStorage.Base,
	)

	for i, v := range cfg.Backends {
		svar.FormatPtr(&v.Dst)
		cfg.Backends[i] = v
	}

	for i, v := range cfg.ErrBackends {
		svar.FormatPtr(&v.Dst)
		cfg.ErrBackends[i] = v
	}

	g := mapSI.Getter()
	ignoreExts, _ := getters.TrueStrings(g, "ignore_ext")
	truncateUri, _ := getters.Int(g, "truncate_uri")
	noColor, _ := getters.Bool(g, "no_color")
	colorTtyCheck, _ := getters.Bool(g, "color_tty_check")
	backends := cfg.BackendPrinter()
	errBackends := cfg.ErrBackendPrinter()

	if len(backends) > 0 || len(errBackends) > 0 {
		var logger, errLogger middleware.LoggerInterface
		if len(backends) > 0 {
			backend := func(backends ...logging.BackendPrintCloser) logging.Printer {
				var bce = make([]logging.Backend, len(backends))
				for i, bc := range backends {
					bce[i] = bc
				}
				for _, bce := range backends {
					this.OnDestroy(iocommon.MustCloser(bce))
				}
				return logging.MultiLogger(bce...).(logging.Printer)
			}(backends...)

			logger = logging.MustPrint(backend.Print)
		} else {
			logger = middleware.DefaultRequestLogFormatter.Logger
		}
		if len(errBackends) > 0 {
			errBackend := func(backends ...logging.BackendPrintCloser) logging.Printer {
				var bce = make([]logging.Backend, len(backends))
				for i, bc := range backends {
					bce[i] = bc
				}
				for _, bce := range backends {
					this.OnDestroy(iocommon.MustCloser(bce))
				}
				return logging.MultiLogger(bce...).(logging.Printer)
			}(errBackends...)

			errLogger = logging.MustPrint(errBackend.Print)
		} else {
			errLogger = middleware.DefaultRequestLogFormatter.PanicLogger
		}
		return &middleware.DefaultLogAndPanicFormatter{
			Logger:           logger,
			PanicLogger:      errLogger,
			NoColor:          noColor,
			IgnoreExtensions: middleware.DefaultLoggerExtensionsIgnore.Update(ignoreExts),
			TruncateUri:      truncateUri,
			NoColorTtyCheck:  !colorTtyCheck,
		}
	}
	return
}
