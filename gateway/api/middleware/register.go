package middleware

import "github.com/gin-gonic/gin"

func Register(engine *gin.Engine) {
	engine.Use(Recovery())
	engine.Use(CORS())
}
