package core

import (
	"context"
	"time"
)

type Contexts []context.Context

func (this Contexts) Deadline() (deadline time.Time, ok bool) {
	if l := len(this); l > 0 {
		for i := l - 1; i >= 0; i-- {
			if deadline, ok = this[i].Deadline(); ok {
				return
			}
		}
	}
	return
}

func (this Contexts) Done() (done <-chan struct{}) {
	if l := len(this); l > 0 {
		for i := l - 1; i >= 0; i-- {
			if done = this[i].Done(); done != nil {
				return
			}
		}
	}
	return
}

func (this Contexts) Err() (err error) {
	if l := len(this); l > 0 {
		for i := l - 1; i >= 0; i-- {
			if err = this[i].Err(); err != nil {
				return
			}
		}
	}
	return
}

func (this Contexts) Value(key interface{}) (value interface{}) {
	if l := len(this); l > 0 {
		for i := l - 1; i >= 0; i-- {
			if value = this[i].Value(key); value != nil {
				return
			}
		}
	}
	return nil
}

func (this Contexts) Prepend(ctx ...context.Context) Contexts {
	return append(append(Contexts{}, ctx...), this...)
}

func (this Contexts) Append(ctx ...context.Context) Contexts {
	return append(this, ctx...)
}
