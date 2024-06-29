package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io/fs"
	"log/slog"
	"net/http"
	"reflect"
	"runtime/debug"
	"sync"
)

func failer(name string) (rollback, ok func()) {
	fail := true
	return func() {
		if fail {
			slog.Error("failed to register field", "name", name)
			debug.PrintStack()
		}
	}, func() { fail = false }
}

func (r *Router) addStruct(v any, gr *gin.RouterGroup) {
	IterStruct(reflect.ValueOf(v), r.registerStructField(gr))
}

func (r *Router) registerStructField(gr *gin.RouterGroup) func(sv reflect.Value, st reflect.Type, sf reflect.StructField) {
	return func(sv reflect.Value, st reflect.Type, sf reflect.StructField) {
		rb, ok := failer(fmt.Sprintf("%s(%s)", st.Name(), sf.Name))
		defer rb()
		MustBeExported(sf)
		if handler, ok := GetValue[Handler](sv, 5); ok {
			registerHandler(handler, sf.Tag, gr)
		} else if static, ok := GetValue[fs.FS](sv, 5); ok {
			fsys := http.FS(static)
			r.r.NoRoute(func(ctx *gin.Context) {
				file := ctx.Request.URL.Path
				f, err := fsys.Open(file)
				if err != nil {
					ctx.FileFromFS("main.html", fsys)
					return
				}
				closeFiles := sync.OnceFunc(func() { _ = f.Close() })
				defer closeFiles()

				if stat, err := f.Stat(); err != nil || stat.IsDir() {
					ctx.FileFromFS("main.html", fsys)
					return
				}
				closeFiles()
				ctx.FileFromFS(file, fsys)
			})
		} else {
			r.registerGroup(sv, sf, gr)
		}
		ok()
	}
}

func registerHandler(handler Handler, tag reflect.StructTag, gr *gin.RouterGroup) {
	for _, method := range GetTag[[]string](tag, "method", false, true) {
		for _, route := range GetTag[[]string](tag, "route", true, true) {
			getMethod(method)(gr, formatRoute(route), handler.Serve)
		}
	}
}

func formatRoute(route string) string {
	if route != "" {
		return fmt.Sprintf("/%s", route)
	}
	return ""
}

func (r *Router) registerGroup(sv reflect.Value, sf reflect.StructField, gr *gin.RouterGroup) {
	groupName := GetTag[string](sf.Tag, "group", false, true)
	r.addStruct(sv.Interface(), gr.Group(fmt.Sprintf("/%s", groupName)))
}

func (r *Router) getVersionGroup(v uint) *gin.RouterGroup {
	if v, ok := r.api.versions[v]; ok {
		return v
	}
	return r.createVersionGroup(v)
}

func (r *Router) createVersionGroup(v uint) *gin.RouterGroup {
	r.api.versions[v] = r.api.r.Group(fmt.Sprintf("/v%d", v))
	return r.api.versions[v]
}
