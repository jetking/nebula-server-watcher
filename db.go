package main

import (
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type LatencyRecord struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	VPSID          string    `gorm:"index" json:"vps_id"`
	Timestamp      time.Time `gorm:"index" json:"timestamp"`
	MedianLatency  float64   `json:"median_latency"`
	AverageLatency float64   `json:"average_latency"`
	MaxLatency     float64   `json:"max_latency"`
	MinLatency     float64   `json:"min_latency"`
}

func InitDB(path string) (*gorm.DB, error) {
	// 使用纯 Go 驱动 glebarez/sqlite
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&LatencyRecord{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
