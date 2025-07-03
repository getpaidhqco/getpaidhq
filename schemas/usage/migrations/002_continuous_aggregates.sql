-- ================================================
-- TimescaleDB Continuous Aggregates for Usage Analytics
-- ================================================
-- This script creates continuous aggregates (materialized views) for efficient
-- querying of usage data at different time granularities.
-- Continuous aggregates are automatically maintained by TimescaleDB.

-- ================================================
-- Hourly Usage Aggregates
-- ================================================
-- Used for real-time dashboards and near real-time analytics
-- Refreshed every 5 minutes for recent data

CREATE MATERIALIZED VIEW usage_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
    org_id,
    subscription_id,
    subscription_item_id,
    usage_type,
    
    -- Quantity aggregations
    SUM(quantity) as total_quantity,
    AVG(quantity) as avg_quantity,
    MAX(quantity) as max_quantity,
    MIN(quantity) as min_quantity,
    
    -- Amount aggregations
    SUM(calculated_amount) as total_amount,
    AVG(calculated_amount) as avg_amount,
    
    -- Event statistics
    COUNT(*) as event_count,
    COUNT(DISTINCT customer_id) as unique_customers,
    
    -- Time tracking
    MIN(time) as first_event_time,
    MAX(time) as last_event_time
    
FROM usage_events
GROUP BY 
    hour, 
    org_id, 
    subscription_id, 
    subscription_item_id, 
    usage_type;

-- Add refresh policy for hourly aggregates
-- Refresh every 5 minutes for data from 3 hours ago to 5 minutes ago
SELECT add_continuous_aggregate_policy('usage_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '5 minutes',
    schedule_interval => INTERVAL '5 minutes'
);

-- ================================================
-- Daily Usage Aggregates for Billing
-- ================================================
-- Primary source for monthly billing calculations
-- Refreshed every hour for data finalization

CREATE MATERIALIZED VIEW usage_daily_billing
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS day,
    org_id,
    subscription_id,
    subscription_item_id,
    customer_id,
    usage_type,
    
    -- Calculate billing period (month) for each day
    date_trunc('month', time) as billing_period,
    
    -- Daily aggregations for billing
    SUM(quantity) as daily_quantity,
    SUM(calculated_amount) as daily_amount,
    COUNT(*) as daily_events,
    
    -- Quality metrics
    COUNT(DISTINCT reference_id) FILTER (WHERE reference_id IS NOT NULL) as unique_references,
    
    -- Time boundaries
    MIN(time) as first_event_time,
    MAX(time) as last_event_time,
    
    -- Usage patterns
    COUNT(*) FILTER (WHERE EXTRACT(hour FROM time) BETWEEN 9 AND 17) as business_hours_events,
    COUNT(*) FILTER (WHERE EXTRACT(hour FROM time) NOT BETWEEN 9 AND 17) as off_hours_events
    
FROM usage_events
GROUP BY 
    day, 
    org_id, 
    subscription_id, 
    subscription_item_id, 
    customer_id,
    usage_type,
    billing_period;

-- Add refresh policy for daily billing aggregates
-- Refresh every hour for data from 3 days ago to 1 hour ago
SELECT add_continuous_aggregate_policy('usage_daily_billing',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour'
);

-- ================================================
-- Monthly Summary Aggregates
-- ================================================
-- Used for monthly analytics, trends, and historical reporting

CREATE MATERIALIZED VIEW usage_monthly_summary
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 month', time) AS month,
    org_id,
    subscription_id,
    subscription_item_id,
    usage_type,
    
    -- Monthly totals
    SUM(quantity) as monthly_quantity,
    SUM(calculated_amount) as monthly_amount,
    
    -- Statistical measures
    AVG(quantity) as avg_daily_quantity,
    STDDEV(quantity) as quantity_stddev,
    MAX(quantity) as peak_daily_quantity,
    MIN(quantity) as min_daily_quantity,
    
    -- Usage patterns
    COUNT(DISTINCT DATE(time)) as active_days,
    COUNT(*) as total_events,
    COUNT(DISTINCT customer_id) as unique_customers,
    
    -- Growth metrics
    SUM(calculated_amount) / NULLIF(COUNT(DISTINCT DATE(time)), 0) as avg_daily_revenue,
    
    -- Time boundaries
    MIN(time) as month_start,
    MAX(time) as month_end
    
FROM usage_events
GROUP BY 
    month, 
    org_id, 
    subscription_id, 
    subscription_item_id, 
    usage_type;

-- Add refresh policy for monthly summaries
-- Refresh daily for data from 3 months ago to 1 day ago
SELECT add_continuous_aggregate_policy('usage_monthly_summary',
    start_offset => INTERVAL '3 months',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day'
);

-- ================================================
-- Customer Usage Aggregates
-- ================================================
-- Aggregates by customer for customer-facing dashboards and analytics

CREATE MATERIALIZED VIEW customer_usage_daily
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', time) AS day,
    org_id,
    customer_id,
    
    -- Customer-level aggregations across all subscriptions
    COUNT(DISTINCT subscription_id) as active_subscriptions,
    COUNT(DISTINCT subscription_item_id) as active_items,
    
    -- Total usage across all items
    SUM(quantity) as total_quantity,
    SUM(calculated_amount) as total_amount,
    COUNT(*) as total_events,
    
    -- Usage diversity
    COUNT(DISTINCT usage_type) as usage_types_used,
    
    -- Activity patterns
    MIN(time) as first_activity,
    MAX(time) as last_activity,
    MAX(time) - MIN(time) as activity_duration
    
FROM usage_events
GROUP BY 
    day, 
    org_id, 
    customer_id;

-- Add refresh policy for customer daily aggregates
SELECT add_continuous_aggregate_policy('customer_usage_daily',
    start_offset => INTERVAL '2 days',
    end_offset => INTERVAL '30 minutes',
    schedule_interval => INTERVAL '30 minutes'
);

-- ================================================
-- Usage Type Analytics Aggregates
-- ================================================
-- For understanding usage patterns across different usage types

CREATE MATERIALIZED VIEW usage_type_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', time) AS hour,
    org_id,
    usage_type,
    
    -- Volume metrics
    SUM(quantity) as total_quantity,
    SUM(calculated_amount) as total_amount,
    COUNT(*) as event_count,
    
    -- Distribution metrics
    COUNT(DISTINCT subscription_id) as subscriptions_using,
    COUNT(DISTINCT customer_id) as customers_using,
    
    -- Performance metrics
    AVG(calculated_amount) as avg_event_value,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY calculated_amount) as median_event_value,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY calculated_amount) as p95_event_value,
    
    -- Activity patterns
    MAX(calculated_amount) as peak_event_value,
    MIN(calculated_amount) as min_event_value
    
FROM usage_events
GROUP BY 
    hour, 
    org_id, 
    usage_type;

-- Add refresh policy for usage type analytics
SELECT add_continuous_aggregate_policy('usage_type_hourly',
    start_offset => INTERVAL '6 hours',
    end_offset => INTERVAL '15 minutes',
    schedule_interval => INTERVAL '15 minutes'
);

-- ================================================
-- Compression for Continuous Aggregates
-- ================================================
-- Enable compression for older aggregate data to save storage

-- Compress hourly aggregates after 30 days
ALTER MATERIALIZED VIEW usage_hourly SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, subscription_id',
    timescaledb.compress_orderby = 'hour DESC'
);

SELECT add_compression_policy('usage_hourly', INTERVAL '30 days');

-- Compress daily billing aggregates after 90 days
ALTER MATERIALIZED VIEW usage_daily_billing SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, subscription_id',
    timescaledb.compress_orderby = 'day DESC'
);

SELECT add_compression_policy('usage_daily_billing', INTERVAL '90 days');

-- Compress monthly summaries after 2 years
ALTER MATERIALIZED VIEW usage_monthly_summary SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, subscription_id',
    timescaledb.compress_orderby = 'month DESC'
);

SELECT add_compression_policy('usage_monthly_summary', INTERVAL '2 years');

-- Compress customer daily aggregates after 6 months
ALTER MATERIALIZED VIEW customer_usage_daily SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, customer_id',
    timescaledb.compress_orderby = 'day DESC'
);

SELECT add_compression_policy('customer_usage_daily', INTERVAL '6 months');

-- Compress usage type analytics after 3 months
ALTER MATERIALIZED VIEW usage_type_hourly SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'org_id, usage_type',
    timescaledb.compress_orderby = 'hour DESC'
);

SELECT add_compression_policy('usage_type_hourly', INTERVAL '3 months');

-- ================================================
-- Indexes for Continuous Aggregates
-- ================================================

-- Indexes for usage_hourly (real-time dashboards)
CREATE INDEX idx_usage_hourly_org_time ON usage_hourly (org_id, hour DESC);
CREATE INDEX idx_usage_hourly_subscription ON usage_hourly (subscription_id, hour DESC);
CREATE INDEX idx_usage_hourly_item ON usage_hourly (subscription_item_id, hour DESC);
CREATE INDEX idx_usage_hourly_type ON usage_hourly (usage_type, hour DESC);

-- Indexes for usage_daily_billing (billing calculations)
CREATE INDEX idx_daily_billing_org_period ON usage_daily_billing (org_id, billing_period, subscription_item_id);
CREATE INDEX idx_daily_billing_subscription ON usage_daily_billing (subscription_id, day DESC);
CREATE INDEX idx_daily_billing_period ON usage_daily_billing (billing_period, org_id);

-- Indexes for customer usage (customer dashboards)
CREATE INDEX idx_customer_daily_org ON customer_usage_daily (org_id, customer_id, day DESC);
CREATE INDEX idx_customer_daily_customer ON customer_usage_daily (customer_id, day DESC);

-- ================================================
-- Utility Functions for Aggregates
-- ================================================

-- Function to manually refresh all continuous aggregates
-- Useful for ensuring data consistency during billing
CREATE OR REPLACE FUNCTION refresh_all_usage_aggregates()
RETURNS void AS $$
BEGIN
    -- Refresh all continuous aggregates manually
    CALL refresh_continuous_aggregate('usage_hourly', NULL, NULL);
    CALL refresh_continuous_aggregate('usage_daily_billing', NULL, NULL);
    CALL refresh_continuous_aggregate('usage_monthly_summary', NULL, NULL);
    CALL refresh_continuous_aggregate('customer_usage_daily', NULL, NULL);
    CALL refresh_continuous_aggregate('usage_type_hourly', NULL, NULL);
    
    -- Log the refresh
    INSERT INTO usage_event_log (
        org_id,
        event_type,
        triggered_by,
        reason,
        metadata
    ) VALUES (
        'system',
        'aggregates_refreshed',
        'manual_refresh',
        'All continuous aggregates manually refreshed',
        json_build_object(
            'refresh_time', NOW(),
            'aggregates', ARRAY[
                'usage_hourly',
                'usage_daily_billing', 
                'usage_monthly_summary',
                'customer_usage_daily',
                'usage_type_hourly'
            ]
        )
    );
    
    RAISE NOTICE 'All usage continuous aggregates refreshed successfully';
END;
$$ LANGUAGE plpgsql;

-- Update the existing refresh function to include continuous aggregates
CREATE OR REPLACE FUNCTION refresh_usage_aggregates()
RETURNS void AS $$
BEGIN
    -- Refresh continuous aggregates for recent data
    PERFORM refresh_all_usage_aggregates();
    
    RAISE NOTICE 'Usage aggregates refreshed for billing consistency';
END;
$$ LANGUAGE plpgsql;

-- Function to get billing summary from aggregates
-- Optimized function for monthly billing calculations
CREATE OR REPLACE FUNCTION get_monthly_billing_summary(
    p_org_id TEXT,
    p_billing_period DATE
)
RETURNS TABLE (
    subscription_id TEXT,
    subscription_item_id TEXT,
    usage_type TEXT,
    total_quantity NUMERIC,
    total_amount BIGINT,
    daily_events BIGINT,
    active_days BIGINT,
    first_usage TIMESTAMPTZ,
    last_usage TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        udb.subscription_id,
        udb.subscription_item_id,
        udb.usage_type,
        SUM(udb.daily_quantity) as total_quantity,
        SUM(udb.daily_amount) as total_amount,
        SUM(udb.daily_events) as daily_events,
        COUNT(DISTINCT udb.day) as active_days,
        MIN(udb.first_event_time) as first_usage,
        MAX(udb.last_event_time) as last_usage
    FROM usage_daily_billing udb
    WHERE udb.org_id = p_org_id
      AND udb.billing_period = p_billing_period
    GROUP BY 
        udb.subscription_id, 
        udb.subscription_item_id, 
        udb.usage_type
    ORDER BY 
        udb.subscription_id, 
        udb.subscription_item_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get real-time usage summary
-- For customer dashboards and real-time analytics
CREATE OR REPLACE FUNCTION get_realtime_usage_summary(
    p_org_id TEXT,
    p_subscription_item_id TEXT,
    p_hours_back INTEGER DEFAULT 24
)
RETURNS TABLE (
    hour TIMESTAMPTZ,
    total_quantity NUMERIC,
    total_amount BIGINT,
    event_count BIGINT,
    avg_amount NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        uh.hour,
        uh.total_quantity,
        uh.total_amount,
        uh.event_count,
        uh.avg_amount
    FROM usage_hourly uh
    WHERE uh.org_id = p_org_id
      AND uh.subscription_item_id = p_subscription_item_id
      AND uh.hour >= NOW() - (p_hours_back || ' hours')::INTERVAL
    ORDER BY uh.hour DESC;
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- Data Quality and Monitoring Views
-- ================================================

-- View for monitoring continuous aggregate lag
CREATE VIEW aggregate_refresh_lag AS
SELECT 
    view_name,
    completed_threshold,
    NOW() - completed_threshold as lag_duration,
    CASE 
        WHEN NOW() - completed_threshold > INTERVAL '1 hour' THEN 'WARNING'
        WHEN NOW() - completed_threshold > INTERVAL '2 hours' THEN 'CRITICAL'
        ELSE 'OK'
    END as status
FROM timescaledb_information.continuous_aggregate_stats
WHERE view_schema = current_schema();

-- View for usage data quality metrics
CREATE VIEW usage_data_quality AS
SELECT 
    date_trunc('day', time) as day,
    org_id,
    COUNT(*) as total_events,
    COUNT(DISTINCT subscription_id) as unique_subscriptions,
    COUNT(DISTINCT customer_id) as unique_customers,
    COUNT(*) FILTER (WHERE reference_id IS NOT NULL) as events_with_reference,
    COUNT(*) FILTER (WHERE quantity IS NULL AND usage_type != 'percentage') as missing_quantity,
    COUNT(*) FILTER (WHERE calculated_amount <= 0) as zero_amount_events,
    AVG(calculated_amount) as avg_amount,
    MAX(time) - MIN(time) as time_span
FROM usage_events
WHERE time >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY day, org_id
ORDER BY day DESC, org_id;

-- ================================================
-- Comments and Documentation
-- ================================================

COMMENT ON MATERIALIZED VIEW usage_hourly IS 'Hourly usage aggregates for real-time dashboards, refreshed every 5 minutes';
COMMENT ON MATERIALIZED VIEW usage_daily_billing IS 'Daily usage aggregates optimized for monthly billing calculations';
COMMENT ON MATERIALIZED VIEW usage_monthly_summary IS 'Monthly usage summaries for analytics and historical reporting';
COMMENT ON MATERIALIZED VIEW customer_usage_daily IS 'Customer-level daily usage aggregates for customer dashboards';
COMMENT ON MATERIALIZED VIEW usage_type_hourly IS 'Usage type analytics for understanding usage patterns';

-- ================================================
-- Success Verification and Logging
-- ================================================

-- Verify continuous aggregates are created
SELECT 
    view_name,
    refresh_lag,
    completed_threshold
FROM timescaledb_information.continuous_aggregate_stats
WHERE view_schema = current_schema();

-- Log successful creation
INSERT INTO usage_event_log (
    org_id,
    event_type,
    triggered_by,
    reason,
    metadata
) VALUES (
    'system',
    'continuous_aggregates_created',
    'migration',
    'Continuous aggregates created for efficient usage analytics',
    json_build_object(
        'migration', '002_continuous_aggregates',
        'timestamp', NOW(),
        'aggregates_created', ARRAY[
            'usage_hourly',
            'usage_daily_billing',
            'usage_monthly_summary', 
            'customer_usage_daily',
            'usage_type_hourly'
        ]
    )
);

-- Print success message
DO $$
BEGIN
    RAISE NOTICE '================================================';
    RAISE NOTICE 'TimescaleDB Continuous Aggregates Created Successfully';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Real-time aggregates: usage_hourly (5-min refresh)';
    RAISE NOTICE 'Billing aggregates: usage_daily_billing (1-hour refresh)';
    RAISE NOTICE 'Analytics aggregates: usage_monthly_summary (daily refresh)';
    RAISE NOTICE 'Customer aggregates: customer_usage_daily (30-min refresh)';
    RAISE NOTICE 'Type analytics: usage_type_hourly (15-min refresh)';
    RAISE NOTICE 'Compression: Enabled for all aggregates';
    RAISE NOTICE 'Functions: Billing and real-time query helpers created';
    RAISE NOTICE '================================================';
END $$;