package ui

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed static/index.html static/styles.css static/app.js
var staticFiles embed.FS

func RegisterRoutes(router *gin.Engine) {
	router.GET("/", func(ctx *gin.Context) {
		content, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			ctx.String(http.StatusInternalServerError, "failed to load index")
			return
		}

		ctx.Data(http.StatusOK, "text/html; charset=utf-8", content)
	})

	router.GET("/assets/styles.css", func(ctx *gin.Context) {
		content, err := staticFiles.ReadFile("static/styles.css")
		if err != nil {
			ctx.String(http.StatusInternalServerError, "failed to load stylesheet")
			return
		}

		ctx.Data(http.StatusOK, "text/css; charset=utf-8", content)
	})

	router.GET("/assets/app.js", func(ctx *gin.Context) {
		content, err := staticFiles.ReadFile("static/app.js")
		if err != nil {
			ctx.String(http.StatusInternalServerError, "failed to load script")
			return
		}

		ctx.Data(http.StatusOK, "application/javascript; charset=utf-8", content)
	})
}
