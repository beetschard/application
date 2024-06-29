package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

var handlerMap = map[string]func(gin.IRoutes, string, ...gin.HandlerFunc) gin.IRoutes{
	http.MethodGet:     gin.IRoutes.GET,
	http.MethodHead:    gin.IRoutes.HEAD,
	http.MethodOptions: gin.IRoutes.OPTIONS,
	http.MethodPost:    gin.IRoutes.POST,
	http.MethodPatch:   gin.IRoutes.PATCH,
	http.MethodPut:     gin.IRoutes.PUT,
	http.MethodDelete:  gin.IRoutes.DELETE,
}

func getMethod(method string) func(gin.IRoutes, string, ...gin.HandlerFunc) gin.IRoutes {
	method = strings.ToUpper(method)
	if fn, ok := handlerMap[method]; ok {
		return fn
	}
	panic(fmt.Sprintf("unsupported method %s", method))
}
