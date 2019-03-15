package helpers

import (
	"strings"

	"github.com/ecletus/oss"

	"github.com/ecletus/core"
	"github.com/ecletus/core/utils"
)

func GetStorageEndpointSchemeHostFromContext(ctx *core.Context, storageName string) (scheme, host string) {
	siteName := strings.Replace(utils.HumanizeString(ctx.Site.Name()), " ", "-", -1)
	storageName = strings.Replace(utils.HumanizeString(storageName), " ", "-", -1)
	key := "X-Ecletus-Oss-Storage-Endpoint-" + siteName + "-" + storageName + "-Host"
	host = ctx.Request.Header.Get(key)
	if host != "" {
		scheme = ctx.Request.Header.Get("X-Forwarded-Proto")
		if scheme == "" {
			if ctx.Request.TLS != nil {
				scheme = "https"
			} else {
				scheme = "http"
			}
		} else if scheme[len(scheme)-1] == 's' {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	return
}

func GetStorageEndpointFromContext(ctx *core.Context, storage oss.NamedStorageInterface, p ...string) (url string) {
	if scheme, host := GetStorageEndpointSchemeHostFromContext(ctx, storage.Name()); host != "" {
		url = storage.GetDynamicURL(scheme, host, p...)
	} else {
		url = storage.GetURL(p...)
	}
	if url != "" && url[0] == '!' {
		url = ctx.Top().GenURL(core.PATH_MEDIA, url[1:])
	}
	return
}
