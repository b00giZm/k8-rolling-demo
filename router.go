package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"k8-rolling-demo/lib/ws"
)

var router *gin.Engine

func init() {
	router = gin.Default()

	wsGroup := router.Group("/ws")
	{
		wsGroup.GET("/service", func(c *gin.Context) {
			ws.WsHandler(c.Writer, c.Request)
		})
	}

	// main html
	router.LoadHTMLGlob("templates/*")
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{})
	})
}
