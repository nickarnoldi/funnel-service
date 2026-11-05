package handler

import (
	"log"
	"runtime"
)

var m runtime.MemStats

func LogMemoryUsage(msg ...string) {
	bToKb := func(b uint64) uint64 {
		return b / 1024
	}
	bToMb := func(b uint64) uint64 {
		return b / 1024 / 1024
	}

	runtime.ReadMemStats(&m)

	if len(msg) > 0 {
		log.Printf("Memory Usage: %s: allocMB %d (%d Kb) | totalMB %d (%d Kb) | sysMB %d (%d Kb) | numGC %d", msg[0], bToMb(m.Alloc), bToKb(m.Alloc), bToMb(m.TotalAlloc), bToKb(m.TotalAlloc), bToMb(m.Sys), bToKb(m.Sys), m.NumGC)
		return
	}
	log.Printf("Memory Usage: allocMB %d (%d Kb) | totalMB %d (%d Kb) | sysMB %d (%d Kb) | numGC %d", bToMb(m.Alloc), bToKb(m.Alloc), bToMb(m.TotalAlloc), bToKb(m.TotalAlloc), bToMb(m.Sys), bToKb(m.Sys), m.NumGC)
	// WithField("allocMB", bToMb(m.Alloc)).WithField("totalMB", bToMb(m.TotalAlloc)).WithField("sysMB", bToMb(m.Sys)).WithField("numGC", m.NumGC).Info("Memory Usage")
}
