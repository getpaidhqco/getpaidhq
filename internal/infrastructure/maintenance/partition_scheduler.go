package maintenance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
)

type PartitionScheduler struct {
	usageDB   *sql.DB
	logger    logger.Logger
	scheduler interfaces.Scheduler
	metrics   MetricsCollector
	config    *Config
}

type MaintenanceResult struct {
	Timestamp         time.Time `json:"timestamp"`
	CreatedPartitions []string  `json:"created_partitions"`
	DroppedPartitions []string  `json:"dropped_partitions"`
	Success           bool      `json:"success"`
	Error             string    `json:"error,omitempty"`
}

func NewPartitionScheduler(usageDB *sql.DB, logger logger.Logger, scheduler interfaces.Scheduler) *PartitionScheduler {
	return &PartitionScheduler{
		usageDB:   usageDB,
		logger:    logger,
		scheduler: scheduler,
		metrics:   &NoOpMetricsCollector{},
		config:    LoadConfigFromEnv(),
	}
}

func (s *PartitionScheduler) Start() error {
	// Validate configuration
	if err := s.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Schedule partition maintenance if enabled
	if s.config.EnablePartitionMaintenance {
		err := s.scheduler.ScheduleTask(s.config.PartitionMaintenanceSchedule, s.performPartitionMaintenance)
		if err != nil {
			return fmt.Errorf("failed to schedule partition maintenance: %w", err)
		}
		s.logger.Info("Scheduled partition maintenance", 
			"schedule", s.config.PartitionMaintenanceSchedule)
	}

	// Schedule materialized view refresh if enabled
	if s.config.EnableMaterializedViewRefresh {
		schedule := fmt.Sprintf("*/%d * * * *", int(s.config.MaterializedViewRefreshInterval.Minutes()))
		err := s.scheduler.ScheduleTask(schedule, s.refreshMaterializedViews)
		if err != nil {
			return fmt.Errorf("failed to schedule materialized view refresh: %w", err)
		}
		s.logger.Info("Scheduled materialized view refresh", 
			"interval", s.config.MaterializedViewRefreshInterval)
	}

	s.logger.Info("Partition scheduler started", 
		"partition_maintenance_enabled", s.config.EnablePartitionMaintenance,
		"view_refresh_enabled", s.config.EnableMaterializedViewRefresh)

	return nil
}

// Note: Stop method removed as it's not part of the Scheduler interface

func (s *PartitionScheduler) performPartitionMaintenance() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	s.logger.Info("Starting partition maintenance")

	var resultJSON []byte
	err := s.usageDB.QueryRowContext(ctx, "SELECT maintain_usage_partitions()").Scan(&resultJSON)
	if err != nil {
		s.logger.Error("Partition maintenance failed", "error", err)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		s.logger.Error("Failed to parse maintenance result", "error", err)
		return
	}

	s.logger.Info("Partition maintenance completed", 
		"created_partitions", result["created_partitions"],
		"dropped_partitions", result["dropped_partitions"])
}

func (s *PartitionScheduler) refreshMaterializedViews() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var resultJSON []byte
	err := s.usageDB.QueryRowContext(ctx, "SELECT refresh_usage_aggregates()").Scan(&resultJSON)
	if err != nil {
		s.logger.Error("Materialized view refresh failed", "error", err)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resultJSON, &result); err != nil {
		s.logger.Error("Failed to parse refresh result", "error", err)
		return
	}

	s.logger.Debug("Materialized views refreshed", 
		"duration_seconds", result["duration_seconds"],
		"views_refreshed", result["views_refreshed"])
}

// ManualMaintenance allows triggering partition maintenance manually
func (s *PartitionScheduler) ManualMaintenance(ctx context.Context) (*MaintenanceResult, error) {
	s.logger.Info("Starting manual partition maintenance")

	var resultJSON []byte
	err := s.usageDB.QueryRowContext(ctx, "SELECT maintain_usage_partitions()").Scan(&resultJSON)
	if err != nil {
		return &MaintenanceResult{
			Timestamp: time.Now(),
			Success:   false,
			Error:     err.Error(),
		}, err
	}

	var dbResult map[string]interface{}
	if err := json.Unmarshal(resultJSON, &dbResult); err != nil {
		return &MaintenanceResult{
			Timestamp: time.Now(),
			Success:   false,
			Error:     fmt.Sprintf("Failed to parse result: %v", err),
		}, err
	}

	// Convert to our result structure
	result := &MaintenanceResult{
		Timestamp: time.Now(),
		Success:   true,
	}

	if createdPartitions, ok := dbResult["created_partitions"].([]interface{}); ok {
		for _, partition := range createdPartitions {
			if partitionStr, ok := partition.(string); ok {
				result.CreatedPartitions = append(result.CreatedPartitions, partitionStr)
			}
		}
	}

	if droppedPartitions, ok := dbResult["dropped_partitions"].([]interface{}); ok {
		for _, partition := range droppedPartitions {
			if partitionStr, ok := partition.(string); ok {
				result.DroppedPartitions = append(result.DroppedPartitions, partitionStr)
			}
		}
	}

	s.logger.Info("Manual partition maintenance completed", 
		"created_partitions", len(result.CreatedPartitions),
		"dropped_partitions", len(result.DroppedPartitions))

	return result, nil
}

// GetPartitionInfo returns current partition status
func (s *PartitionScheduler) GetPartitionInfo(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.usageDB.QueryContext(ctx, "SELECT * FROM get_partition_info('usage_events')")
	if err != nil {
		return nil, fmt.Errorf("failed to get partition info: %w", err)
	}
	defer rows.Close()

	var partitions []map[string]interface{}
	for rows.Next() {
		var partitionName, partitionBounds, sizePretty string
		var startDate, endDate *time.Time
		var rowCount, sizeBytes int64

		err := rows.Scan(&partitionName, &partitionBounds, &startDate, &endDate, &rowCount, &sizeBytes, &sizePretty)
		if err != nil {
			return nil, fmt.Errorf("failed to scan partition info: %w", err)
		}

		partition := map[string]interface{}{
			"partition_name":   partitionName,
			"partition_bounds": partitionBounds,
			"start_date":       startDate,
			"end_date":         endDate,
			"row_count":        rowCount,
			"size_bytes":       sizeBytes,
			"size_pretty":      sizePretty,
		}
		partitions = append(partitions, partition)
	}

	return partitions, nil
}
