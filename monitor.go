package main

import (
	"log"
	"sort"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"gorm.io/gorm"
)

type Monitor struct {
	db     *gorm.DB
	config *Config
}

func NewMonitor(db *gorm.DB, config *Config) *Monitor {
	return &Monitor{
		db:     db,
		config: config,
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
			m.saveStats(vps.ID, results)
		}
		<-ticker.C
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

func (m *Monitor) saveStats(vpsID string, latencies []time.Duration) {
	if len(latencies) == 0 {
		return
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
}
