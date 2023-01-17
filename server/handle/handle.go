package handle

import (
	resp "cess-cacher/server/response"
	"cess-cacher/server/service"
	"cess-cacher/utils"
	"fmt"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func DownloadHandler(c *gin.Context) {
	tk, ok := c.Get("ticket")
	if !ok {
		resp.RespError(c, resp.NewError(400, errors.New("bad token")))
		return
	}
	res, se := service.DownloadService(tk.(service.Ticket))
	if se != nil {
		if se.Status() == 0 {
			resp.RespOk(c, res)
			return
		}
		resp.RespError(c, se)
		return
	}
	_, fname := path.Split(res)
	if fname == "" {
		fname = utils.GetRandomcode(64)
	}
	c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%v", fname))
	c.Writer.Header().Add("Content-Type", "application/octet-stream")
	c.File(res)
}

func QueryHandler(c *gin.Context) {
	opt := strings.Split(c.Request.URL.Path, "/")[2]
	switch opt {
	case "stats":
		if stat, err := service.QueryMinerStats(); err != nil {
			resp.RespError(c, err)
		} else {
			resp.RespOk(c, stat)
		}
	case "cached":
		res := service.QueryCachedFiles()
		resp.RespOkWithFlag(c, res != nil, res)
	case "file":
		hash := c.Param("hash")
		if hash == "" {
			resp.RespError(c, resp.NewError(400, errors.New("bad params")))
			return
		}
		res := service.QueryFileInfo(hash)
		resp.RespOkWithFlag(c, res.Size > 0, res)
	}
}

func AuthHandler(c *gin.Context) {
	var req service.AuthReq
	if err := c.BindJSON(&req); err != nil {
		resp.RespError(c, resp.NewError(400, errors.Wrap(err, "bad params")))
		return
	}
	if token, err := service.GenerateToken(req.BID, req.Sign); err != nil {
		resp.RespError(c, err)
	} else {
		resp.RespOk(c, token)
	}
}
