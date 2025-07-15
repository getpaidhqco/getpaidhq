package maintenance

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds configuration for the partition scheduler
type Config struct {
	// PartitionRetentionMonths is how long to keep partitions (default: 60 months / 5 years)
	PartitionRetentionMonths int
	
	// PartitionMonthsAhead is how many months ahead to create partitions (default: 6)
	PartitionMonthsAhead int
	
	// MaterializedViewRefreshInterval is how often to refresh views (default: 5 minutes)
	MaterializedViewRefreshInterval time.Duration
	
	// PartitionMaintenanceSchedule is the cron schedule for partition maintenance (default: "0 2 1 * *")
	PartitionMaintenanceSchedule string
	
	// EnableMaterializedViewRefresh enables automatic materialized view refresh
	EnableMaterializedViewRefresh bool
	
	// EnablePartitionMaintenance enables automatic partition maintenance
	EnablePartitionMaintenance bool
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		PartitionRetentionMonths:        60,
		PartitionMonthsAhead:            6,
		MaterializedViewRefreshInterval: 5 * time.Minute,
		PartitionMaintenanceSchedule:    "0 2 1 * *", // 1st of month at 2:00 AM
		EnableMaterializedViewRefresh:   true,
		EnablePartitionMaintenance:      true,
	}
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() *Config {
	config := DefaultConfig()
	
	if val := os.Getenv("USAGE_PARTITION_RETENTION_MONTHS"); val != "" {
		if months, err := strconv.Atoi(val); err == nil {
			config.PartitionRetentionMonths = months
		}
	}
	
	if val := os.Getenv("USAGE_PARTITION_MONTHS_AHEAD"); val != "" {
		if months, err := strconv.Atoi(val); err == nil {
			config.PartitionMonthsAhead = months
		}
	}
	
	if val := os.Getenv("USAGE_MATERIALIZED_VIEW_REFRESH_INTERVAL"); val != "" {
		if minutes, err := strconv.Atoi(val); err == nil {
			config.MaterializedViewRefreshInterval = time.Duration(minutes) * time.Minute
		}
	}
	
	if val := os.Getenv("USAGE_PARTITION_MAINTENANCE_SCHEDULE"); val != "" {
		config.PartitionMaintenanceSchedule = val
	}
	
	if val := os.Getenv("USAGE_DISABLE_MATERIALIZED_VIEW_REFRESH"); val == "true" {
		config.EnableMaterializedViewRefresh = false
	}
	
	if val := os.Getenv("USAGE_DISABLE_PARTITION_MAINTENANCE"); val == "true" {
		config.EnablePartitionMaintenance = false
	}
	
	return config
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.PartitionRetentionMonths < 12 {
		return fmt.Errorf("partition retention must be at least 12 months")
	}
	
	if c.PartitionMonthsAhead < 1 {
		return fmt.Errorf("must create at least 1 month of partitions ahead")
	}
	
	if c.MaterializedViewRefreshInterval < time.Minute {
		return fmt.Errorf("materialized view refresh interval must be at least 1 minute")
	}
	
	return nil
}