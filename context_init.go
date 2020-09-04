package core

import "time"

func (this *Context) Init() {
	if this.Request == nil {
		return
	}
	if this.TimeLocation == nil {
		if user := this.CurrentUser(); user != nil {
			this.TimeLocation = user.GetTimeLocation()
		}
		if this.TimeLocation == nil {
			if htl := this.Request.Header.Get("X-Time-Location"); htl != "" {
				this.TimeLocation, _ = time.LoadLocation(htl)
			}
		}
		if this.TimeLocation == nil {
			if this.Site != nil {
				this.TimeLocation = this.Site.TimeLocation()
			} else {
				this.TimeLocation = time.Local
			}
		}
	}
}
