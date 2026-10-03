package test

import (
	"github.com/gin-gonic/gin"
	"github.com/gigabytegrove/monita/auth"
	"github.com/gigabytegrove/monita/model"
)

// WithUser fake an authentication for testing.
func WithUser(ctx *gin.Context, userID uint) {
	auth.RegisterUser(ctx, &model.User{ID: userID})
}
