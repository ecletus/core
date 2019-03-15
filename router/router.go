package router

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/aghape/core"
)

type Container struct {
	mountPath string
	mux       interface{}
}

func (c *Container) Mux() interface{} {
	return c.mux
}

func (c *Container) ServeMux() *ServeMux {
	return c.mux.(*ServeMux)
}

func (c *Container) HttpServeMux() *http.ServeMux {
	return c.mux.(*http.ServeMux)
}

type ServerHandler func(next http.HandlerFunc, w http.ResponseWriter, r *http.Request)

type ServeMux struct {
	*http.ServeMux
	container       *Container
	notFoundHandler http.HandlerFunc
	handler         ServerHandler
	ContextFactory  *core.ContextFactory
}

func NewServerMux(contextFactory *core.ContextFactory) (mux *ServeMux) {
	mux = &ServeMux{http.NewServeMux(), nil, http.NotFound, nil, contextFactory}
	mux.ServeMux.Handle("/", mux)
	return mux
}

func (mux *ServeMux) SetHandler(handler ServerHandler) *ServeMux {
	mux.handler = handler
	return mux
}

func (mux *ServeMux) GetHandler() ServerHandler {
	return mux.handler
}

func (mux *ServeMux) SetNotFoundHandler(f http.HandlerFunc) *ServeMux {
	mux.notFoundHandler = f
	return mux
}

func (mux *ServeMux) GetNotFoundHandler() http.HandlerFunc {
	return mux.notFoundHandler
}

func (mux *ServeMux) GetCurrentNotFoundHandler() http.HandlerFunc {
	notFoundHander := mux.notFoundHandler
	if reflect.ValueOf(notFoundHander).Pointer() == reflect.ValueOf(http.NotFound).Pointer() && mux.container != nil {
		if p, ok := mux.container.mux.(*ServeMux); ok {
			return p.GetCurrentNotFoundHandler()
		}
	}
	return notFoundHander
}

func (mux *ServeMux) next(w http.ResponseWriter, r *http.Request) {
	mux.ServeMux.ServeHTTP(w, r)
}

func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if mux.handler == nil {
		mux.next(w, r)
	} else {
		mux.handler(mux.next, w, r)
	}
}

func (mux *ServeMux) Mux() *http.ServeMux {
	return mux.ServeMux
}

func (mux *ServeMux) MountTo(path string, container *http.ServeMux) *http.ServeMux {
	path = strings.Trim(path, "/")

	if path == "" {
		panic(errors.New("mount path is empty"))
	}
	if container == nil {
		container = http.NewServeMux()
	}
	if mux.container != nil {
		panic(fmt.Errorf("Server has be mounted to %q", mux.container.mountPath))
	}

	path = "/" + path
	mux.container = &Container{path, container}
	container.HandleFunc(path, mux.GetCurrentNotFoundHandler())
	container.HandleFunc(path+"/", func(w http.ResponseWriter, r *http.Request) {
		r, _ = mux.ContextFactory.NewContextFromRequestPair(w, r, path)
		mux.ServeHTTP(w, r)
	})
	return container
}

type PathHandler struct {
	path            string
	handler         http.Handler
	notFoundHandler http.Handler
	contextFactory  *core.ContextFactory
}

func (s *PathHandler) Path() string {
	return s.path
}

func (s *PathHandler) NotFound(handler http.Handler) *PathHandler {
	s.notFoundHandler = handler
	return s
}

func (s *PathHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.path == "/" {
		r, _ := s.contextFactory.NewContextFromRequestPair(w, r, s.path)
		s.handler.ServeHTTP(w, r)
	} else {
		if strings.HasPrefix(r.URL.Path, s.path) {
			r, _ = s.contextFactory.NewContextFromRequestPair(w, r, s.path)
			s.handler.ServeHTTP(w, r)
		} else {
			s.notFoundHandler.ServeHTTP(w, r)
		}
	}
}

func NewPathHandler(contextFactory *core.ContextFactory, path string, handler http.Handler) *PathHandler {
	return &PathHandler{
		"/" + strings.Trim(path, "/"),
		handler,
		http.NotFoundHandler(),
		contextFactory,
	}
}
