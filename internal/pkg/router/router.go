package router

import (
	"context"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net"
	"net/http"
	"reflect"
)

type API interface {
	Version() uint
}

type Handler interface {
	Serve(ctx *gin.Context)
}

type Router struct {
	r   *gin.Engine
	api struct {
		r        *gin.RouterGroup
		versions map[uint]*gin.RouterGroup
	}
}

func New() *Router {
	r := &Router{
		r: gin.Default(),
	}
	r.r.Use(cors.Default())
	r.api.r = r.r.Group("/api")
	r.api.versions = map[uint]*gin.RouterGroup{}
	return r
}

func (r *Router) AddAPI(api API) {
	rb, ok := failer(reflect.TypeOf(api).Name())
	defer rb()
	r.addStruct(api, r.getVersionGroup(api.Version()))
	ok()
}

func (r *Router) ServeHTTP(ctx context.Context, network, addr string) error {
	slog.Info("Serving http and listening", "network", network, "addr", addr)

	ln, err := new(net.ListenConfig).Listen(ctx, network, addr)
	if err != nil {
		return err
	}

	return (&http.Server{
		Handler:     r.r,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}).Serve(ln)
}
