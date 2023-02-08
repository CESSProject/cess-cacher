package middleware

import (
	resp "cess-cacher/server/response"
	"cess-cacher/server/service"
	"errors"
	"strings"

	"github.com/btcsuite/btcutil/base58"

	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		if token == "" {
			resp.RespError(c, resp.NewError(400, errors.New("bad token")))
			c.Abort()
			return
		}
		bytes := base58.Decode(token)
		pbs, err := service.GetAESHandle().SymmetricDecrypt(bytes)
		if err != nil {
			resp.RespError(c, resp.NewError(400, err))
			c.Abort()
			return
		}
		params := strings.Split(string(pbs), "-")
		if len(params) < 2 {
			resp.RespError(c, resp.NewError(400, errors.New("bad token")))
			c.Abort()
			return
		}
		ticket, err := service.PraseTicketByBID(params[0], params[1])
		if err != nil {
			resp.RespError(c, resp.NewError(500, err))
			c.Abort()
			return
		}
		c.Set("ticket", ticket)
		c.Next()
	}
}
