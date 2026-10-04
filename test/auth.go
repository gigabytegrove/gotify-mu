package test

import (
	"github.com/gigabytegrove/monita/auth"
	"github.com/gigabytegrove/monita/model"
	"github.com/gin-gonic/gin"
)

// WithUser fake an authentication for testing.
func WithUser(ctx *gin.Context, userID uint) {
	auth.RegisterUser(ctx, &model.User{ID: userID})
}
