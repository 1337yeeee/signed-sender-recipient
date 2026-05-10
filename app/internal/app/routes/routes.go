package routes

import (
	"net/http"

	"electronic-digital-signature/internal/app/container"
	"electronic-digital-signature/internal/app/ui"

	"github.com/gin-gonic/gin"
)

func SetupRouter(appContainer *container.AppContainer) *gin.Engine {
	r := gin.Default()
	if appContainer != nil {
		r.Use(corsMiddleware(appContainer.CORSAllowedOrigins))
	}
	ui.RegisterRoutes(r)

	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"status": "ok",
			},
		})
	})

	api := r.Group("/api/v1")
	if appContainer == nil {
		api.GET("/identity", handlerNotConfigured)
		api.POST("/documents/send", handlerNotConfigured)
		api.POST("/packages/verify-decrypt", handlerNotConfigured)
		return r
	}

	if appContainer.IdentityHandler == nil {
		api.GET("/identity", handlerNotConfigured)
	} else {
		api.GET("/identity", appContainer.IdentityHandler.GetIdentity)
	}

	if appContainer.DocumentHandler == nil {
		api.POST("/documents/send", handlerNotConfigured)
		api.POST("/packages/verify-decrypt", handlerNotConfigured)
	} else {
		api.POST("/documents/send", appContainer.DocumentHandler.SendDocument)
		api.POST("/packages/verify-decrypt", appContainer.DocumentHandler.VerifyDecryptPackage)
	}

	return r
}

func handlerNotConfigured(ctx *gin.Context) {
	ctx.JSON(http.StatusInternalServerError, gin.H{
		"success": false,
		"error": gin.H{
			"code":    "internal_error",
			"message": "Requested handler is not configured.",
		},
	})
}
