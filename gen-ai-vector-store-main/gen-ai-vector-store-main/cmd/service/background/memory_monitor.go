/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

package background

import (
	"runtime"
	"time"

	"go.uber.org/zap"
)

func MonitorMemory(logger *zap.Logger) {
	const softLimitMB = 400

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		usedMB := m.Alloc / 1024 / 1024

		logger.Info("Current memory usage", zap.Uint64("usedMB", usedMB))

		if int(usedMB) > softLimitMB {
			logger.Warn("Memory usage exceeded limit", zap.Int("limitMB", softLimitMB))

			// Trigger garbage collection manually
			runtime.GC()
		}
	}
}
