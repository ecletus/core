package core

import (
	"path"

	"github.com/moisespsena-go/httpu"
)

func (this *Context) DefaultStorageEndpoint() (url string) {
	return this.StorageEndpoint("default")
}

func (this *Context) StorageEndpoint(storageName string) (url string) {
	var (
		storage           = this.Site.GetMediaStorageOrDefault(storageName)
		siteName          = this.Site.Name()
		key, host, scheme string
	)
	key = "X-Ecletus-Oss-Storage-Endpoint-" + siteName + "-" + storage.Name() + "-Host"
	host = this.Request.Header.Get(key)
	if host != "" {
		scheme = httpu.HttpScheme(this.Request)
	}

	if host != "" {
		url = storage.GetDynamicURL(scheme, host)
	} else {
		url = storage.GetURL()
	}
	if url != "" && url[0] == '!' {
		url = this.Top().Path(PATH_MEDIA, url[1:])
	}
	return
}

func (this *Context) MediaURL(storageName, name string, pth ...string) string {
	return this.StorageEndpoint(storageName) + "/" + path.Join(append([]string{name}, pth...)...)
}

func (this *Context) DefaultMediaURL(name string, pth ...string) string {
	return this.MediaURL("default", name, pth...)
}
