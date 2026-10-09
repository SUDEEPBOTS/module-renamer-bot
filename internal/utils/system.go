package utils

import (
	"fmt"
	"runtime"
	"time"
)

var StartTime = time.Now()

// SystemStats aggregates telemetry data for the admin /stats command.
type SystemStats struct {
	Uptime       string
	Goroutines   int
	AllocRAMMB   float64
	TotalRAMMB   float64
	SysRAMMB     float64
	NumCPU       int
	GoVersion    string
}

// GetSystemStats captures runtime memory and process stats.
func GetSystemStats() SystemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(StartTime).Round(time.Second)

	return SystemStats{
		Uptime:     uptime.String(),
		Goroutines: runtime.NumGoroutine(),
		AllocRAMMB: float64(m.Alloc) / 1024 / 1024,
		TotalRAMMB: float64(m.TotalAlloc) / 1024 / 1024,
		SysRAMMB:   float64(m.Sys) / 1024 / 1024,
		NumCPU:     runtime.NumCPU(),
		GoVersion:  runtime.Version(),
	}
}

// FormatStatsMessage formats stats into an attractive monospace card.
func (s SystemStats) FormatStatsMessage(totalUsers, totalRenames int64) string {
	return fmt.Sprintf(
		"📊 <b><u>𝐒𝐘𝐒𝐓𝐄𝐌 𝐓𝐄𝐋𝐄𝐌𝐄𝐓𝐑𝐘 & 𝐒𝐓𝐀𝐓𝐒</u></b>\n\n"+
			"⏱ <b>Uptime:</b> <code>%s</code>\n"+
			"👥 <b>Total Users:</b> <code>%d</code>\n"+
			"🔄 <b>Total Renames:</b> <code>%d</code>\n"+
			"⚙️ <b>Go Version:</b> <code>%s</code>\n"+
			"🧵 <b>Goroutines:</b> <code>%d</code>\n"+
			"💻 <b>CPU Cores:</b> <code>%d</code>\n"+
			"💾 <b>RAM Allocated:</b> <code>%.2f MB</code>\n"+
			"📦 <b>Sys Memory:</b> <code>%.2f MB</code>\n",
		s.Uptime, totalUsers, totalRenames, s.GoVersion, s.Goroutines, s.NumCPU, s.AllocRAMMB, s.SysRAMMB,
	)
}
