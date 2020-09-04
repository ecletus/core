package helpers

import (
	"github.com/moisespsena-go/httpu"
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
		scheme = httpu.HttpScheme(ctx.Request)
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
		url = ctx.Top().Path(core.PATH_MEDIA, url[1:])
	}
	return
}
