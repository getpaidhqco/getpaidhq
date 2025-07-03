-- ================================================
-- TimescaleDB Initialization for Usage Database
-- ================================================
-- This script initializes TimescaleDB with the usage_events hypertable
-- and supporting tables for high-performance time-series usage recording.

-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- ================================================
-- Main Usage Events Table (Hypertable)
-- ================================================

-- Create the main usage events table
-- This will be converted to a hypertable for time-series optimization
CREATE TABLE usage_events (
    time TIMESTAMPTZ NOT NULL,
    org_id TEXT NOT NULL,
    subscription_id TEXT NOT NULL,
    subscription_item_id TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    usage_type TEXT NOT NULL,
    quantity NUMERIC(15, 4),
    transaction_value BIGINT,
    calculated_amount BIGINT NOT NULL,
    reference_id TEXT,
    reference_type TEXT,
    metadata JSONB,
    
    -- Composite primary key optimized for TimescaleDB partitioning
    PRIMARY KEY (org_id, subscription_item_id, time)
);

-- Convert to hypertable with daily partitioning
-- This enables TimescaleDB's time-series optimizations
SELECT create_hypertable(
    'usage_events', 
    'time',
    chunk_time_interval => INTERVAL '1 day',
    create_default_indexes => FALSE -- We'll create custom indexes
);

-- ================================================
-- Indexes for Common Query Patterns
-- ================================================

-- Time-based queries (most common for analytics)
CREATE INDEX idx_usage_events_time ON usage_events (time DESC);

-- Organization-scoped time queries
CREATE INDEX idx_usage_events_org_time ON usage_events (org_id, time DESC);

-- Subscription-level usage queries
CREATE INDEX idx_usage_events_subscription ON usage_events (org_id, subscription_id, time DESC);

-- Subscription item specific queries (billing calculations)
CREATE INDEX idx_usage_events_subscription_item ON usage_events (subscription_item_id, time DESC);

-- Customer usage analytics
CREATE INDEX idx_usage_events_customer ON usage_events (org_id, customer_id, time DESC);

-- Usage type analytics
CREATE INDEX idx_usage_events_usage_type ON usage_events (usage_type, time DESC);

-- Deduplication lookups (performance critical)
CREATE INDEX idx_usage_events_reference ON usage_events (reference_id, reference_type) 
WHERE reference_id IS NOT NULL;

-- Billing period queries (monthly aggregations)
CREATE INDEX idx_usage_events_billing_period ON usage_events (
    org_id, 
    date_trunc('month', time), 
    subscription_item_id
);

-- ================================================
-- Compression Configuration
-- ================================================

-- Enable compression for the hypertable
-- This can reduce storage by 90%+ for older data
ALTER TABLE usage_events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, subscription_id',
    timescaledb.compress_orderby = 'time DESC, subscription_item_id'
);

-- Add compression policy (compress data older than 7 days)
-- Recent data stays uncompressed for fast writes and updates
SELECT add_compression_policy('usage_events', INTERVAL '7 days');

-- ================================================
-- Data Retention Configuration
-- ================================================

-- Add retention policy (automatically delete data older than 5 years)
-- This prevents the database from growing indefinitely
SELECT add_retention_policy('usage_events', INTERVAL '5 years');

-- ================================================
-- Usage Processing Status Table
-- ================================================

-- Table to track billing processing status
-- This is a regular PostgreSQL table (not a hypertable) for fast CRUD operations
CREATE TABLE usage_processing_status (
    org_id TEXT NOT NULL,
    subscription_item_id TEXT NOT NULL,
    billing_period TEXT NOT NULL, -- Format: "2024-01" for monthly billing
    total_quantity NUMERIC(15, 4) NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL DEFAULT 0,
    event_count INTEGER NOT NULL DEFAULT 0,
    processed BOOLEAN NOT NULL DEFAULT FALSE,
    processed_at TIMESTAMPTZ,
    invoice_id TEXT,
    first_event_time TIMESTAMPTZ NOT NULL,
    last_event_time TIMESTAMPTZ NOT NULL,
    last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (org_id, subscription_item_id, billing_period)
);

-- Indexes for billing processing queries
CREATE INDEX idx_usage_processing_billing ON usage_processing_status (
    org_id, processed, billing_period
);
CREATE INDEX idx_usage_processing_invoice ON usage_processing_status (invoice_id) 
WHERE invoice_id IS NOT NULL;
CREATE INDEX idx_usage_processing_time ON usage_processing_status (processed_at) 
WHERE processed_at IS NOT NULL;

-- ================================================
-- Usage Event Log Table
-- ================================================

-- Audit trail for usage processing events
CREATE TABLE usage_event_log (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    org_id TEXT NOT NULL,
    event_type TEXT NOT NULL, -- 'recorded', 'processed', 'corrected', 'refunded'
    subscription_id TEXT,
    subscription_item_id TEXT,
    customer_id TEXT,
    invoice_id TEXT,
    amount BIGINT,
    quantity NUMERIC(15, 4),
    event_count INTEGER,
    billing_period TEXT,
    triggered_by TEXT, -- User or system identifier
    reason TEXT,
    metadata JSONB
);

-- Indexes for audit trail queries
CREATE INDEX idx_usage_event_log_time ON usage_event_log (timestamp DESC);
CREATE INDEX idx_usage_event_log_org ON usage_event_log (org_id, event_type, timestamp DESC);
CREATE INDEX idx_usage_event_log_item ON usage_event_log (subscription_item_id, timestamp DESC);
CREATE INDEX idx_usage_event_log_invoice ON usage_event_log (invoice_id) WHERE invoice_id IS NOT NULL;
CREATE INDEX idx_usage_event_log_period ON usage_event_log (billing_period, event_type) 
WHERE billing_period IS NOT NULL;

-- ================================================
-- Usage Retention Policies Table
-- ================================================

-- Configuration for automated data lifecycle management
CREATE TABLE usage_retention_policies (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    org_id TEXT NOT NULL,
    retention_period TEXT NOT NULL, -- PostgreSQL interval: "5 years"
    compression_period TEXT NOT NULL, -- PostgreSQL interval: "7 days"
    usage_type TEXT, -- Optional: apply to specific usage types
    customer_id TEXT, -- Optional: customer-specific policies
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    last_applied TIMESTAMPTZ,
    next_application TIMESTAMPTZ,
    
    -- Prevent duplicate policies
    UNIQUE (org_id, usage_type, customer_id)
);

-- Indexes for policy management
CREATE INDEX idx_retention_policies_org ON usage_retention_policies (org_id, active);
CREATE INDEX idx_retention_policies_schedule ON usage_retention_policies (next_application) 
WHERE next_application IS NOT NULL;

-- ================================================
-- Performance Optimization Functions
-- ================================================

-- Function to refresh all materialized views
-- This will be called during billing processing for data consistency
CREATE OR REPLACE FUNCTION refresh_usage_aggregates()
RETURNS void AS $$
BEGIN
    -- Note: Continuous aggregates will be created in the next migration
    -- This function will be updated to refresh them
    RAISE NOTICE 'Usage aggregates refresh function created. Continuous aggregates will be added in next migration.';
END;
$$ LANGUAGE plpgsql;

-- Function to get usage summary for a billing period
-- Optimized for billing calculations
CREATE OR REPLACE FUNCTION get_billing_period_usage(
    p_org_id TEXT,
    p_billing_period_start TIMESTAMPTZ,
    p_billing_period_end TIMESTAMPTZ
)
RETURNS TABLE (
    subscription_id TEXT,
    subscription_item_id TEXT,
    usage_type TEXT,
    total_quantity NUMERIC,
    total_amount BIGINT,
    event_count BIGINT,
    first_usage TIMESTAMPTZ,
    last_usage TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ue.subscription_id,
        ue.subscription_item_id,
        ue.usage_type,
        COALESCE(SUM(ue.quantity), 0) as total_quantity,
        SUM(ue.calculated_amount) as total_amount,
        COUNT(*) as event_count,
        MIN(ue.time) as first_usage,
        MAX(ue.time) as last_usage
    FROM usage_events ue
    WHERE ue.org_id = p_org_id
      AND ue.time >= p_billing_period_start
      AND ue.time < p_billing_period_end
    GROUP BY ue.subscription_id, ue.subscription_item_id, ue.usage_type
    ORDER BY ue.subscription_id, ue.subscription_item_id;
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- Database Statistics and Monitoring
-- ================================================

-- Enable detailed statistics collection for query optimization
ALTER SYSTEM SET track_activities = on;
ALTER SYSTEM SET track_counts = on;
ALTER SYSTEM SET track_io_timing = on;
ALTER SYSTEM SET track_functions = all;

-- ================================================
-- Initial Data and Configuration
-- ================================================

-- Insert default retention policy for all organizations
-- This can be customized per organization later
INSERT INTO usage_retention_policies (
    org_id,
    retention_period,
    compression_period,
    active,
    created_by
) VALUES (
    'default',
    '5 years',
    '7 days',
    true,
    'system'
) ON CONFLICT DO NOTHING;

-- ================================================
-- Comments and Documentation
-- ================================================

-- Add table comments for documentation
COMMENT ON TABLE usage_events IS 'High-volume usage events stored as TimescaleDB hypertable. Partitioned by time for optimal performance.';
COMMENT ON TABLE usage_processing_status IS 'Tracks billing processing status for usage aggregations. Used during monthly billing cycles.';
COMMENT ON TABLE usage_event_log IS 'Audit trail for all usage processing events. Provides complete history for debugging and compliance.';
COMMENT ON TABLE usage_retention_policies IS 'Configuration for automated data lifecycle management and compression policies.';

-- Add column comments for critical fields
COMMENT ON COLUMN usage_events.time IS 'Event timestamp - used as partition key for TimescaleDB hypertable';
COMMENT ON COLUMN usage_events.org_id IS 'Organization identifier - part of composite primary key for data isolation';
COMMENT ON COLUMN usage_events.calculated_amount IS 'Final calculated amount in cents - used for billing aggregations';
COMMENT ON COLUMN usage_events.reference_id IS 'External reference for idempotency and deduplication';

-- ================================================
-- Success Verification
-- ================================================

-- Verify TimescaleDB extension is working
SELECT extname, extversion FROM pg_extension WHERE extname = 'timescaledb';

-- Verify hypertable creation
SELECT hypertable_name, num_chunks FROM timescaledb_information.hypertables 
WHERE hypertable_name = 'usage_events';

-- Show compression and retention policies
SELECT * FROM timescaledb_information.compression_settings 
WHERE hypertable_name = 'usage_events';

SELECT * FROM timescaledb_information.drop_chunks_policies 
WHERE hypertable_name = 'usage_events';

-- Log successful initialization
INSERT INTO usage_event_log (
    org_id,
    event_type,
    triggered_by,
    reason,
    metadata
) VALUES (
    'system',
    'database_initialized',
    'migration',
    'TimescaleDB usage database initialized successfully',
    '{"migration": "001_initialize_timescaledb", "timestamp": "' || NOW()::TEXT || '"}'
);

-- Print success message
DO $$
BEGIN
    RAISE NOTICE '================================================';
    RAISE NOTICE 'TimescaleDB Usage Database Initialization Complete';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Hypertable: usage_events created with daily partitioning';
    RAISE NOTICE 'Compression: Enabled for data older than 7 days';
    RAISE NOTICE 'Retention: Data automatically deleted after 5 years';
    RAISE NOTICE 'Indexes: Optimized for common usage query patterns';
    RAISE NOTICE '================================================';
END $$;