package handler

import (
	"log"

	"electronic-digital-signature/internal/app/dto"

	"github.com/gin-gonic/gin"
)

func respondSuccess(ctx *gin.Context, status int, data any) {
	ctx.JSON(status, dto.SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func respondError(ctx *gin.Context, status int, code, message string) {
	ctx.JSON(status, dto.ErrorResponse{
		Success: false,
		Error: dto.ErrorDetails{
			Code:    code,
			Message: message,
		},
	})
}

func logRequestError(ctx *gin.Context, scope string, err error) {
	if err == nil {
		return
	}

	log.Printf("%s %s %s: %v", scope, ctx.Request.Method, ctx.FullPath(), err)
}

func logRequestInfo(ctx *gin.Context, scope, message string) {
	log.Printf("%s %s %s: %s", scope, ctx.Request.Method, ctx.FullPath(), message)
}
