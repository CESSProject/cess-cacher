package server

import (
	"cess-cacher/config"
	"cess-cacher/logger"
	"cess-cacher/server/service"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	READ_TIMEOUT  = time.Second * 10
	WRITE_TIMEOUT = time.Second * 10
)

func SetupGinServer() {

	service.InitTickets()
	go service.OrdersCleanServer()

	gin.SetMode(gin.ReleaseMode)
	router := NewRouter()
	httpServer := &http.Server{
		Addr:           ":" + config.GetConfig().ServerPort,
		Handler:        router,
		ReadTimeout:    READ_TIMEOUT,
		WriteTimeout:   WRITE_TIMEOUT,
		MaxHeaderBytes: 1 << 20,
	}
	if err := httpServer.ListenAndServe(); err != nil {
		logger.Uld.Sugar().Errorf("run http server error:%v", err)
		log.Printf("run http server error:%v.\n", err)
	}
}
