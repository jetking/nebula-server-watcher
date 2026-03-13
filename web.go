package main

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

//go:embed web/*
var staticFS embed.FS

type WebServer struct {
	db     *gorm.DB
	config *Config
}

func NewWebServer(db *gorm.DB, config *Config) *WebServer {
	return &WebServer{
		db:     db,
		config: config,
	}
}

func (ws *WebServer) Start(addr string) error {
	// 禁用 Gin 的默认重定向行为，以防万一
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// 获取子文件系统
	subFS, _ := fs.Sub(staticFS, "web")

	// Auth Middleware
	authMiddleware := func(c *gin.Context) {
		password := c.GetHeader("X-Password")
		if password != ws.config.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}

	// API Routes
	api := r.Group("/api", authMiddleware)
	{
		api.GET("/vps", ws.handleGetVPS)
		api.GET("/stats", ws.handleGetStats)
	}

	// 显式路由：不使用 StaticFS 避免任何可能的路径冲突
	r.GET("/", func(c *gin.Context) {
		content, _ := fs.ReadFile(subFS, "index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	})

	r.GET("/vps", func(c *gin.Context) {
		content, _ := fs.ReadFile(subFS, "vps.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	})

	// 处理 favicon 等 404
	r.NoRoute(func(c *gin.Context) {
		c.Status(http.StatusNotFound)
	})

	return r.Run(addr)
}

func (ws *WebServer) handleGetVPS(c *gin.Context) {
	c.JSON(http.StatusOK, ws.config.VPSList)
}

func (ws *WebServer) handleGetStats(c *gin.Context) {
	vpsID := c.Query("vps_id")
	since := c.Query("since")

	query := ws.db.Model(&LatencyRecord{})
	if vpsID != "" {
		query = query.Where("vps_id = ?", vpsID)
	}

	if since != "" {
		d, err := time.ParseDuration(since)
		if err == nil {
			query = query.Where("timestamp > ?", time.Now().Add(-d))
		}
	} else {
		query = query.Where("timestamp > ?", time.Now().Add(-24*time.Hour))
	}

	var stats []LatencyRecord
	if err := query.Order("timestamp desc").Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
