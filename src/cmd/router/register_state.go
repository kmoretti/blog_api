package cmd

import (
	"blog_api/src/handler"
	"blog_api/src/middleware"

	"github.com/gin-gonic/gin"
)

func registerStateRoutes(apiGroup *gin.RouterGroup, masterPassword string) {
	stateHandler := handler.NewStateHandler()
	stateGroup := apiGroup.Group("/internal/states")
	stateGroup.Use(middleware.StateMasterAuth(masterPassword))
	stateGroup.PUT("/:key", stateHandler.PutState)
	stateGroup.GET("/:key", stateHandler.GetState)
	stateGroup.DELETE("/:key", stateHandler.DeleteState)
}
