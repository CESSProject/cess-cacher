package middleware

import (
	resp "cess-cacher/server/response"
	"cess-cacher/server/service"

	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const BEARER_PREFIX = "Bearer "

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		bearer := c.Request.Header.Get("Authorization")
		if bearer == "" {
			resp.RespError(c, resp.NewError(
				http.StatusUnauthorized,
				errors.New("authorization field in header cannot be found"),
			))
			c.Abort()
		}
		jwtStr := strings.TrimPrefix(bearer, BEARER_PREFIX)
		claims, err := service.PraseToken(jwtStr)
		if err != nil {
			resp.RespError(c, resp.NewError(http.StatusUnauthorized, err))
			c.Abort()
		}
		if !claims.VerifyExpiresAt(time.Now(), true) {
			c.Abort()
		}
		c.Set("ticket", claims.Ticket)
		c.Next()
	}
}
