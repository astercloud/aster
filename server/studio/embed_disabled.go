//go:build !studio

package studio

import (
	"github.com/gin-gonic/gin"
)

// Enabled indicates whether the studio is embedded
const Enabled = false

// RegisterRoutes is a no-op when studio is not embedded
func RegisterRoutes(router *gin.Engine) {
	// Studio not enabled
}
