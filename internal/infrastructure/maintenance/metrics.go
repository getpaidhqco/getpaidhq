package maintenance

import (
	"context"
	"time"
)

// MetricsCollector interface for collecting scheduler metrics
type MetricsCollector interface {
	RecordPartitionMaintenance(duration time.Duration, created, dropped int, err error)
	RecordMaterializedViewRefresh(duration time.Duration, err error)
	RecordPartitionCount(count int)
	RecordPartitionSize(partitionName string, sizeBytes int64)
}

// NoOpMetricsCollector is a no-op implementation for when metrics aren't configured
type NoOpMetricsCollector struct{}

func (n *NoOpMetricsCollector) RecordPartitionMaintenance(duration time.Duration, created, dropped int, err error) {}
func (n *NoOpMetricsCollector) RecordMaterializedViewRefresh(duration time.Duration, err error) {}
func (n *NoOpMetricsCollector) RecordPartitionCount(count int) {}
func (n *NoOpMetricsCollector) RecordPartitionSize(partitionName string, sizeBytes int64) {}

// WithMetrics adds metrics collection to the scheduler
func (s *PartitionScheduler) WithMetrics(collector MetricsCollector) *PartitionScheduler {
	s.metrics = collector
	return s
}

// Add metrics field to PartitionScheduler struct
type PartitionSchedulerWithMetrics struct {
	*PartitionScheduler
	metrics MetricsCollector
}

// performPartitionMaintenanceWithMetrics wraps maintenance with metrics
func (s *PartitionScheduler) performPartitionMaintenanceWithMetrics() {
	start := time.Now()
	ctx := context.Background()

	s.logger.Info("Starting partition maintenance with metrics")
	
	var resultJSON []byte
	err := s.usageDB.QueryRowContext(ctx, "SELECT maintain_usage_partitions()").Scan(&resultJSON)
	
	duration := time.Since(start)
	
	if s.metrics != nil {
		// Parse result to get created/dropped counts
		// For now, just record the duration
		s.metrics.RecordPartitionMaintenance(duration, 0, 0, err)
	}

	if err != nil {
		s.logger.Error("Partition maintenance failed", 
			"error", err,
			"duration_ms", duration.Milliseconds())
		return
	}

	s.logger.Info("Partition maintenance completed", 
		"duration_ms", duration.Milliseconds())
}

// refreshMaterializedViewsWithMetrics wraps refresh with metrics  
func (s *PartitionScheduler) refreshMaterializedViewsWithMetrics() {
	start := time.Now()
	ctx := context.Background()

	var resultJSON []byte
	err := s.usageDB.QueryRowContext(ctx, "SELECT refresh_usage_aggregates()").Scan(&resultJSON)
	
	duration := time.Since(start)
	
	if s.metrics != nil {
		s.metrics.RecordMaterializedViewRefresh(duration, err)
	}

	if err != nil {
		s.logger.Error("Materialized view refresh failed", 
			"error", err,
			"duration_ms", duration.Milliseconds())
		return
	}

	s.logger.Debug("Materialized views refreshed", 
		"duration_ms", duration.Milliseconds())
}

// CollectPartitionMetrics collects metrics about current partitions
func (s *PartitionScheduler) CollectPartitionMetrics(ctx context.Context) error {
	if s.metrics == nil {
		return nil
	}

	partitions, err := s.GetPartitionInfo(ctx)
	if err != nil {
		return err
	}

	s.metrics.RecordPartitionCount(len(partitions))

	for _, p := range partitions {
		if name, ok := p["partition_name"].(string); ok {
			if sizeBytes, ok := p["size_bytes"].(int64); ok {
				s.metrics.RecordPartitionSize(name, sizeBytes)
			}
		}
	}

	return nil
}