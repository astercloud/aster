//go:build studio

package studio

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:dist
var distFS embed.FS

// Enabled indicates whether the studio is embedded
const Enabled = true

// RegisterRoutes registers the studio routes
func RegisterRoutes(router *gin.Engine) {
	// Get the dist subdirectory
	distDir, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("failed to get dist directory: " + err.Error())
	}

	// Handle /studio/* routes
	router.GET("/studio/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")

		// Remove leading slash
		if strings.HasPrefix(filepath, "/") {
			filepath = filepath[1:]
		}

		// Default to index.html for root path
		if filepath == "" {
			filepath = "index.html"
		}

		// Try to serve the static file
		file, err := distDir.Open(filepath)
		if err == nil {
			defer file.Close()
			// Serve the file
			http.ServeFileFS(c.Writer, c.Request, distDir, filepath)
			return
		}

		// For SPA routing, serve index.html for non-existent paths
		http.ServeFileFS(c.Writer, c.Request, distDir, "index.html")
	})

	// Redirect /studio to /studio/
	router.GET("/studio", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/studio/")
	})
}
