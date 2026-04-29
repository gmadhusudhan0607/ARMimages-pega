// Copyright (c) 2026 Pegasystems Inc.
// All rights reserved.

package usagemetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/log"
	"github.com/Pega-CloudEngineering/gen-ai-vector-store/internal/resources/isolations"
	"go.uber.org/zap"
)

const (
	metricSourceGenAIVectorStore = "GenAIVectorStore"
	maxRetryCount                = 10
	uploaderLoggerName           = "usagemetrics.uploader"
	pdcSenderLoggerName          = "usagemetrics.uploader.pdc"
)

// Uploader handles background uploading of usage metrics
type Uploader struct {
	collector  *Collector
	isoManager IsolationGetter
	pdcSender  *PDCSender[SemanticSearchMetric]
	logger     *zap.Logger
}

type IsolationGetter interface {
	GetIsolation(ctx context.Context, isolationID string) (*isolations.Details, error)
}

// NewUploader creates a new usage metrics uploader
func NewUploader(collector *Collector, isoManager isolations.IsoManager) *Uploader {
	config := collector.GetConfig()

	pdcSender := NewPDCSender[SemanticSearchMetric](PDCSenderConfig{
		RequestTimeoutSeconds: config.RequestTimeoutSeconds,
		MaxPayloadSizeBytes:   config.MaxPayloadSizeBytes,
	}, pdcSenderLoggerName)

	return &Uploader{
		collector:  collector,
		isoManager: isoManager,
		pdcSender:  pdcSender,
		logger:     log.GetNamedLogger(uploaderLoggerName),
	}
}

// StartBackgroundUploader starts the background upload process
// This should be called in a separate goroutine
func (u *Uploader) StartBackgroundUploader(ctx context.Context) {
	config := u.collector.GetConfig()
	if !config.Enabled {
		u.logger.Info("Usage metrics upload disabled")
		return
	}

	u.logger.Info("Starting usage metrics background uploader",
		zap.Int("intervalSeconds", config.UploadIntervalSeconds))

	ticker := time.NewTicker(time.Duration(config.UploadIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			u.logger.Info("Usage metrics uploader stopping due to context cancellation")
			return
		case <-ticker.C:
			u.processQueue(ctx)
		}
	}
}

// processQueue processes all queued metrics and uploads them to their respective usage data endpoints
func (u *Uploader) processQueue(ctx context.Context) {
	queue := u.collector.GetAndClearQueue()
	if len(queue) == 0 {
		u.logger.Debug("No usage metrics to upload")
		return
	}

	u.logger.Info("Processing usage metrics queue", zap.Int("uniqueIsolations", len(queue)))

	for isolationID, metrics := range queue {
		// Get isolation details to retrieve PDC endpoint URL
		iso, err := u.isoManager.GetIsolation(ctx, isolationID)
		if err != nil {
			u.logger.Error("Failed to get isolation details, discarding metrics",
				zap.String("isolationID", isolationID),
				zap.Int("metricsCount", len(metrics)),
				zap.Error(err))
			continue
		}

		if iso.PDCEndpointURL == "" {
			u.logger.Warn("Usage data endpoint URL not configured for isolation, discarding metrics",
				zap.String("isolationID", isolationID),
				zap.Int("metricsCount", len(metrics)))
			continue
		}

		if err, missedMetrics := u.uploadMetrics(ctx, iso.PDCEndpointURL, metrics); err != nil {
			u.logger.Error("Failed to upload metrics for isolation",
				zap.String("isolationID", isolationID),
				zap.String("usageDataURL", iso.PDCEndpointURL),
				zap.Int("metricsCount", len(metrics)),
				zap.Error(err))

			// Re-queue the missed metrics for next attempt
			u.requeueMetrics(missedMetrics)
		}
	}
}

// uploadMetrics uploads metrics to the specified usage data URL with retry logic
// PDCSender handles chunking automatically
func (u *Uploader) uploadMetrics(ctx context.Context, usageDataURL string, metrics []SemanticSearchMetric) (err error, missedMetrics []SemanticSearchMetric) {
	config := u.collector.GetConfig()

	u.logger.Info("Uploading metrics to usage data endpoint",
		zap.String("usageDataURL", usageDataURL),
		zap.Int("metricsCount", len(metrics)))

	// Implement retry logic (PDCSender doesn't retry)
	var lastErr error
	for attempt := 0; attempt < config.RetryCount; attempt++ {
		// PDCSender handles chunking and URL conversion
		if err := u.pdcSender.Send(ctx, usageDataURL, metrics); err != nil {
			lastErr = err

			// Check if this is a partial failure (some chunks succeeded)
			partialErr, _ := err.(*PartialFailureError)

			// Exponential backoff: 10s, 20s, 30s...
			backoffDuration := time.Duration(10*(attempt+1)) * time.Second

			u.logger.Warn("Usage metrics upload attempt failed, retrying",
				zap.String("usageDataURL", usageDataURL),
				zap.Int("attempt", attempt+1),
				zap.Int("maxRetries", config.RetryCount),
				zap.Duration("backoffDuration", backoffDuration),
				zap.Error(err))

			if attempt < config.RetryCount-1 { // Don't sleep on last attempt
				select {
				case <-ctx.Done():
					// On context cancellation, return only unsent metrics
					if partialErr != nil {
						return ctx.Err(), metrics[partialErr.SuccessfullySent:]
					}
					return ctx.Err(), metrics
				case <-time.After(backoffDuration):
					// Continue retry with only unsent metrics
					if partialErr != nil {
						u.logger.Info("Retrying with unsent metrics only",
							zap.Int("successfullySent", partialErr.SuccessfullySent),
							zap.Int("remaining", len(metrics)-partialErr.SuccessfullySent))
						metrics = metrics[partialErr.SuccessfullySent:]
					}
					// Continue to next attempt
				}
			} else {
				// Last attempt failed - return only unsent metrics
				if partialErr != nil {
					return fmt.Errorf("failed to upload after %d attempts: %w", config.RetryCount, lastErr),
						metrics[partialErr.SuccessfullySent:]
				}
			}
		} else {
			u.logger.Info("Successfully uploaded usage metrics",
				zap.String("usageDataURL", usageDataURL),
				zap.Int("metricsCount", len(metrics)),
				zap.Int("attempt", attempt+1))
			return nil, nil
		}
	}

	return fmt.Errorf("failed to upload after %d attempts: %w", config.RetryCount, lastErr), metrics
}

// requeueMetrics adds failed metrics back to the collector queue
func (u *Uploader) requeueMetrics(metrics []SemanticSearchMetric) {
	u.logger.Info("Re-queueing failed metrics for next upload attempt",
		zap.Int("metricsCount", len(metrics)))

	for _, metric := range metrics {
		if metric.retryCount >= maxRetryCount {
			u.logger.Warn("Discarding metric after max retry attempts reached",
				zap.String("isolationID", metric.IsolationID),
				zap.String("collectionID", metric.CollectionID),
				zap.String("endpoint", metric.Endpoint),
				zap.Int("retryCount", metric.retryCount))
			continue
		}

		metric.retryCount++

		u.collector.AddMetric(metric)
	}
}
