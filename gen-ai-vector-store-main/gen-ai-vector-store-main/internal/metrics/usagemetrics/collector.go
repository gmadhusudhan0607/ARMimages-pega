// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

import (
	"sync"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"go.uber.org/zap"
)

var (
	collectorInstance *Collector
	collectorOnce     sync.Once
)

// Collector manages the collection and queuing of usage metrics
type Collector struct {
	queue  map[string][]SemanticSearchMetric // keyed by isolationID
	mutex  sync.Mutex
	config Config
	logger *zap.Logger
}

// NewCollector creates a new usage metrics collector with the given configuration
func NewCollector(config Config) *Collector {
	return &Collector{
		queue:  make(map[string][]SemanticSearchMetric),
		config: config,
		logger: log.GetNamedLogger("usagemetrics.collector"),
	}
}

// GetCollector returns the singleton instance of the usage metrics collector
func GetCollector() *Collector {
	collectorOnce.Do(func() {
		collectorInstance = NewCollector(DefaultConfig())
	})
	return collectorInstance
}

// SetCollectorConfig updates the collector configuration
func SetCollectorConfig(config Config) {
	collector := GetCollector()
	collector.mutex.Lock()
	defer collector.mutex.Unlock()
	collector.config = config
}

// AddMetric adds a metric to the queue for the specified isolation ID
func (c *Collector) AddMetric(metric SemanticSearchMetric) {
	if metric.IsolationID == "" {
		c.logger.Debug("Discarding metric: isolation ID is empty",
			zap.String("collectionID", metric.CollectionID),
			zap.String("endpoint", metric.Endpoint))
		return
	}

	if !c.IsEnabled() {
		c.logger.Debug("Usage metrics collection disabled, discarding metric")
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.queue[metric.IsolationID] = append(c.queue[metric.IsolationID], metric)

	c.logger.Debug("Added metric to usage data queue",
		zap.String("isolationID", metric.IsolationID),
		zap.String("collectionID", metric.CollectionID),
		zap.String("endpoint", metric.Endpoint),
		zap.Int("queueSize", len(c.queue[metric.IsolationID])))
}

// GetAndClearQueue atomically returns the current queue and clears it
func (c *Collector) GetAndClearQueue() map[string][]SemanticSearchMetric {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Copy the current queue
	queueCopy := make(map[string][]SemanticSearchMetric, len(c.queue))
	for url, metrics := range c.queue {
		queueCopy[url] = make([]SemanticSearchMetric, len(metrics))
		copy(queueCopy[url], metrics)
	}

	// Clear the original queue
	c.queue = make(map[string][]SemanticSearchMetric)

	totalMetrics := 0
	for _, metrics := range queueCopy {
		totalMetrics += len(metrics)
	}

	if totalMetrics > 0 {
		c.logger.Debug("Retrieved and cleared usage metrics queue",
			zap.Int("totalMetrics", totalMetrics),
			zap.Int("uniqueIsolations", len(queueCopy)))
	}

	return queueCopy
}

// IsEnabled returns whether usage metrics collection is enabled
func (c *Collector) IsEnabled() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.config.Enabled
}

// GetConfig returns the current configuration
func (c *Collector) GetConfig() Config {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.config
}
