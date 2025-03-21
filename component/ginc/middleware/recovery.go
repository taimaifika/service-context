package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	sctx "github.com/taimaifika/service-context"
)

type CanGetStatusCode interface {
	StatusCode() int
}

func Recovery(serviceCtx sctx.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.Header("Content-Type", "application/json")

				if appErr, ok := err.(CanGetStatusCode); ok {
					c.AbortWithStatusJSON(appErr.StatusCode(), appErr)
				} else {
					// General panic cases
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"code":    http.StatusInternalServerError,
						"status":  "internal server error",
						"message": "something went wrong, please try again or contact supporters",
					})
				}

				slog.Error("%+v \n", err)

				// Must go with gin recovery
				if gin.IsDebugging() {
					panic(err)
				}
			}
		}()
		c.Next()
	}
}
