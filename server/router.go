package server

import (
	"cess-cacher/server/handle"
	"cess-cacher/server/middleware"
	resp "cess-cacher/server/response"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(cors.Default())
	router.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		resp.RespError(c, resp.NewError(http.StatusInternalServerError, err.(error)))
	}))
	//download file group
	dowmloadGroup := router.Group("/download").Use(middleware.Auth())
	dowmloadGroup.GET("/file", handle.DownloadHandler)

	//query group
	query := router.Group("/query")
	query.GET("/stats", handle.QueryHandler)
	query.GET("/cached", handle.QueryHandler)
	query.GET("/file/:hash", handle.QueryHandler)

	//auth group
	auth := router.Group("/auth")
	auth.POST("/gen", handle.AuthHandler)
	return router
}
