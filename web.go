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
		api.GET("/uptime", ws.handleGetUptime)
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
	start := c.Query("start")
	end := c.Query("end")

	query := ws.db.Model(&LatencyRecord{})
	if vpsID != "" {
		query = query.Where("vps_id = ?", vpsID)
	}

	if start != "" && end != "" {
		query = query.Where("timestamp BETWEEN ? AND ?", start, end)
	} else if since != "" {
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

func (ws *WebServer) handleGetUptime(c *gin.Context) {
	vpsID := c.Query("vps_id")
	if vpsID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vps_id is required"})
		return
	}

	days := 90
	since := time.Now().AddDate(0, 0, -days)

	type DailyStatus struct {
		Date        string  `json:"date"`
		AvgMedian   float64 `json:"avg_median"`
		RedCount    int     `json:"red_count"`    // > 200ms
		YellowCount int     `json:"yellow_count"` // 50-200ms
	}

	var results []DailyStatus
	// 针对 SQLite 使用 strftime 函数进行按日期分组
	err := ws.db.Raw(`
		SELECT 
			strftime('%Y-%m-%d', timestamp) as date,
			AVG(median_latency) as avg_median,
			SUM(CASE WHEN median_latency > 200 THEN 1 ELSE 0 END) as red_count,
			SUM(CASE WHEN median_latency >= 50 AND median_latency <= 200 THEN 1 ELSE 0 END) as yellow_count
		FROM latency_records
		WHERE vps_id = ? AND timestamp > ?
		GROUP BY date
		ORDER BY date ASC
	`, vpsID, since).Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, results)
}
