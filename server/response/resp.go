package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Error interface {
	error
	Status() int
}

type StatefulError struct {
	Code int
	Err  error
}

type StatefulOk struct {
	Code int
	Data any
}

type Response struct {
	Result bool `json:"result"`
	Data   any  `json:"data"`
}

func (e StatefulError) Status() int {
	return e.Code
}

func (e StatefulError) Error() string {
	return e.Err.Error()
}

func NewError(code int, err error) StatefulError {
	return StatefulError{
		Code: code,
		Err:  err,
	}
}

func RespError(c *gin.Context, err Error) {
	resp := Response{
		Result: false,
		Data:   err.Error(),
	}
	c.JSON(err.Status(), resp)
}

func RespOk(c *gin.Context, data any) {
	RespOkWithFlag(c, true, data)
}

func RespOkWithFlag(c *gin.Context, flag bool, data any) {
	resp := Response{Result: flag}
	if d, ok := data.(StatefulOk); ok {
		resp.Data = d.Data
		c.JSON(d.Code, resp)
		return
	}
	resp.Data = data
	c.JSON(http.StatusOK, resp)
}
