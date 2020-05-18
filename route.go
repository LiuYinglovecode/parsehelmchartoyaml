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
	r1 := gin.New()
	r1.Use(gin.Recovery())
	r1.Use(gin.Logger())

	v1 := r1.Group("/1/helm");
	{
		v1.GET("/values", GetValues)
		v1.POST("/manifest", Manifest)
	}
	//RemoteHelm := NewRemoteHelm()
	//router.GET("/1/helm/values", RemoteHelm.Values)
	//router.PUT("/1/helm/upload", RemoteHelm.Mani)

	return r1
}