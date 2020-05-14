package main

import (
	"github.com/gin-gonic/gin"
)

var db = make(map[string]string)

// SetupRouter 建立路由关系
func setupRouter() *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	//r := gin.Default()
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	RemoteHelm := NewRemoteHelm()
	router.GET("/1/helm/values", RemoteHelm.Values)
	router.GET("/1/helm/upload", RemoteHelm.Mani)

	return router
}
