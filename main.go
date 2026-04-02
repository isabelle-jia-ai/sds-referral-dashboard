package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"

	"go.apps.applied.dev/lib/cloudlogger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

//go:generate sh -c "cd frontend && npm install && npm run build"

//go:embed frontend/dist
var frontendFS embed.FS

func main() {
	logger := cloudlogger.New()
	zap.ReplaceGlobals(logger)
	defer logger.Sync()

	initDatalakeClient(context.Background())
	initGreenhouseClient()

	go ghClient.PreWarmLinkedInCache()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	RegisterAPIRoutes(r)

	if os.Getenv("ENV") != "dev" {
		distFS, err := fs.Sub(frontendFS, "frontend/dist")
		if err != nil {
			zap.L().Fatal("failed to load frontend", zap.Error(err))
		}

		serveIndex := func(c *gin.Context) {
			data, _ := fs.ReadFile(distFS, "index.html")
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		}

		r.GET("/", serveIndex)
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if len(path) > 1 {
				if f, err := distFS.Open(path[1:]); err == nil {
					f.Close()
					c.FileFromFS(path[1:], http.FS(distFS))
					return
				}
			}
			serveIndex(c)
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	zap.L().Info("server starting", zap.String("port", port))
	if err := r.Run(":" + port); err != nil {
		zap.L().Fatal("failed to start server", zap.Error(err))
	}
}
