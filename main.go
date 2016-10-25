package main

import "github.com/gin-gonic/gin"
import "net/http"

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.Static("/assets", "static/assets")
	r.Static("/data", "static/data")
	r.GET("/toon", func(c *gin.Context) {
		c.HTML(http.StatusOK, "toon.tmpl", gin.H{
			"image": "goat_toon.jpg",
		})
	})
	r.Run()
}
