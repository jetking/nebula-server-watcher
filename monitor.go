package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"gorm.io/gorm"
)

type Monitor struct {
	db     *gorm.DB
	config *Config

	// Alert tracking
	mu              sync.Mutex
	consecutiveHigh map[string]int       // VPS_ID -> consecutive counts
	lastAlertTime   map[string]time.Time // VPS_ID -> last notification time
}

func NewMonitor(db *gorm.DB, config *Config) *Monitor {
	return &Monitor{
		db:              db,
		config:          config,
		consecutiveHigh: make(map[string]int),
		lastAlertTime:   make(map[string]time.Time),
	}
}

func (m *Monitor) Start() {
	for _, vps := range m.config.VPSList {
		go m.watchVPS(vps)
	}
}

func (m *Monitor) watchVPS(vps VPSConfig) {
	log.Printf("Starting monitoring for VPS: %s (%s)", vps.Name, vps.IP)

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		// Run a ping session for 60 seconds
		results := m.pingSession(vps.IP, 55*time.Second)
		if len(results) > 0 {
			median := m.saveStats(vps.ID, results)
			m.checkAlert(vps, median)
		}
		<-ticker.C
	}
}

func (m *Monitor) checkAlert(vps VPSConfig, median float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	alertCfg := m.config.Alert
	threshold := alertCfg.Threshold
	if threshold <= 0 {
		return // Not configured
	}

	// 默认值处理
	consecutiveLimit := alertCfg.ConsecutiveCount
	if consecutiveLimit <= 0 {
		consecutiveLimit = 3 // 默认 3 次
	}
	cooldown := time.Duration(alertCfg.CooldownMinutes) * time.Minute
	if cooldown <= 0 {
		cooldown = 30 * time.Minute // 默认 30 分钟
	}

	if median > threshold {
		m.consecutiveHigh[vps.ID]++
	} else {
		m.consecutiveHigh[vps.ID] = 0
	}

	if m.consecutiveHigh[vps.ID] >= consecutiveLimit {
		lastAlert, ok := m.lastAlertTime[vps.ID]
		if !ok || time.Since(lastAlert) >= cooldown {
			m.sendTelegramAlert(vps, median, consecutiveLimit)
			m.lastAlertTime[vps.ID] = time.Now()
		}
	}
}

func (m *Monitor) sendTelegramAlert(vps VPSConfig, median float64, consecutive int) {
	token := m.config.Telegram.Token
	chatID := m.config.Telegram.ChatID

	if token == "" || chatID == "" {
		log.Printf("Telegram not configured, skipping alert for %s", vps.Name)
		return
	}

	text := fmt.Sprintf("⚠️ *Server Latency Alert*\n\nVPS: %s (%s)\nMedian: %.2fms\nThreshold: %.2fms\nStatus: High latency for %d consecutive minutes.\n\n_This alert is sent by Nebula Server Watcher._",
		vps.Name, vps.IP, median, m.config.Alert.Threshold, consecutive)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := http.PostForm(apiURL, url.Values{
		"chat_id":    {chatID},
		"text":       {text},
		"parse_mode": {"Markdown"},
	})

	if err != nil {
		log.Printf("Error sending Telegram alert: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Telegram alert failed with status: %d", resp.StatusCode)
	} else {
		log.Printf("Telegram alert sent for %s", vps.Name)
	}
}

func (m *Monitor) pingSession(ip string, duration time.Duration) []time.Duration {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		log.Printf("Error creating pinger for %s: %v", ip, err)
		return nil
	}

	// On some systems (like macOS or Linux without root), SetPrivileged(true) might be needed
	// or false to use UDP. Let's try privileged=false first for better compatibility.
	pinger.SetPrivileged(false)

	var latencies []time.Duration
	var mu sync.Mutex

	pinger.OnRecv = func(pkt *probing.Packet) {
		mu.Lock()
		latencies = append(latencies, pkt.Rtt)
		mu.Unlock()
	}

	// Run for the duration
	go func() {
		time.Sleep(duration)
		pinger.Stop()
	}()

	err = pinger.Run()
	if err != nil {
		log.Printf("Error running pinger for %s: %v", ip, err)
	}

	return latencies
}

func (m *Monitor) saveStats(vpsID string, latencies []time.Duration) float64 {
	if len(latencies) == 0 {
		return 0
	}

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}

	count := float64(len(latencies))
	avg := float64(sum.Milliseconds()) / count
	min := float64(latencies[0].Milliseconds())
	max := float64(latencies[len(latencies)-1].Milliseconds())

	var median float64
	mid := len(latencies) / 2
	if len(latencies)%2 == 0 {
		median = float64((latencies[mid-1] + latencies[mid]).Milliseconds()) / 2
	} else {
		median = float64(latencies[mid].Milliseconds())
	}

	record := LatencyRecord{
		VPSID:          vpsID,
		Timestamp:      time.Now(),
		MedianLatency:  median,
		AverageLatency: avg,
		MaxLatency:     max,
		MinLatency:     min,
	}

	if err := m.db.Create(&record).Error; err != nil {
		log.Printf("Error saving record to DB: %v", err)
	}

	return median
}
