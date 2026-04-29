// Copyright (c) 2025 Pegasystems Inc.
// All rights reserved.

package dbconfigpuller

import (
	"context"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/db"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/helpers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	workersmetrics "github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/metrics/workers"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/schema/migrations"
	"go.uber.org/zap"
)

const (
	DefaultPullIntervalSec = 300 // 5 minutes
	workerName             = "db-config-puller"
)

var configPullerLogger = log.GetNamedLogger(workerName)

// DBConfigPuller periodically pulls database configuration values and exposes them as Prometheus metrics
type DBConfigPuller struct {
	logger          *zap.Logger
	database        db.Database
	pullIntervalSec int
	configKeys      []string // List of configuration keys to expose
}

// NewDBConfigPuller creates a new configuration puller with a configurable pull interval
func NewDBConfigPuller(database db.Database) *DBConfigPuller {
	pullInterval := int(helpers.GetEnvOrDefaultInt64("DB_CONFIG_PULL_INTERVAL_SEC", DefaultPullIntervalSec))

	return &DBConfigPuller{
		logger:          configPullerLogger,
		database:        database,
		pullIntervalSec: pullInterval,
		configKeys: []string{
			migrations.KeyVsSchemaVersion,
			migrations.KeyVsSchemaVersionPrev,
		},
	}
}

// Run starts the periodic configuration pulling process
func (p *DBConfigPuller) Run(ctx context.Context) error {
	// Initialize worker progress to 0%
	workersmetrics.SetWorkerProgress(p.logger, workerName, 0.0)
	p.logger.Info("Initialized DB config puller progress metric to 0%")

	p.logger.Info("Starting DB configuration puller",
		zap.Int("pullIntervalSec", p.pullIntervalSec),
		zap.Strings("configKeys", p.configKeys))

	// Perform initial pull immediately
	p.pullAndExposeConfiguration(ctx)

	// Set progress to 100% after initial run (no incremental progress for polling workers)
	workersmetrics.SetWorkerProgress(p.logger, workerName, 100.0)
	p.logger.Info("Set DB config puller progress to 100% after initial run")

	ticker := time.NewTicker(time.Duration(p.pullIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("DB configuration puller stopped")
			return ctx.Err()
		case <-ticker.C:
			p.pullAndExposeConfiguration(ctx)
		}
	}
}

// pullAndExposeConfiguration reads configuration from a database and exposes as metrics
func (p *DBConfigPuller) pullAndExposeConfiguration(ctx context.Context) {
	p.logger.Debug("Pulling DB configuration values")

	allConfigs, err := migrations.GetVsConfiguration(ctx, p.database.GetConn())
	if err != nil {
		p.logger.Error("Failed to get database configuration", zap.Error(err))
		return
	}

	// Expose each configured key as a metric
	for _, key := range p.configKeys {
		value, exists := allConfigs[key]
		if !exists {
			p.logger.Debug("Configuration key not found in database",
				zap.String("key", key))
			continue
		}

		workersmetrics.SetConfigurationValue(p.logger, key, value)
		p.logger.Debug("Exposed configuration value",
			zap.String("key", key),
			zap.String("value", value))
	}

	p.logger.Debug("Successfully updated DB configuration metrics",
		zap.Int("keysExposed", len(p.configKeys)))
}
