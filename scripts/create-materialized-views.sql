-- ================================================
-- PostgreSQL Usage Database - Materialized Views for Analytics
-- ================================================
-- This script creates materialized views for efficient usage data aggregation
-- These provide near real-time analytics without the overhead of live queries

-- ================================================
-- Hourly Usage Aggregates (Real-time Dashboards)
-- ================================================

-- Hourly usage aggregates for real-time dashboards
CREATE MATERIALIZED VIEW usage_hourly AS
SELECT 
    date_trunc('hour', time) AS hour,
    org_id,
    subscription_id,
    subscription_item_id,
    data->>'usage_type' as usage_type,
    SUM(CAST(data->>'quantity' AS DECIMAL)) as total_quantity,
    SUM(CAST(data->>'amount' AS BIGINT)) as total_amount,
    COUNT(*) as event_count,
    MAX(time) as last_event_time
FROM usage_events
GROUP BY hour, org_id, subscription_id, subscription_item_id, data->>'usage_type';

-- Create indexes on the materialized view for fast queries
CREATE UNIQUE INDEX idx_usage_hourly_unique ON usage_hourly (hour, org_id, subscription_item_id, usage_type);
CREATE INDEX idx_usage_hourly_org_time ON usage_hourly (org_id, hour DESC);
CREATE INDEX idx_usage_hourly_subscription ON usage_hourly (subscription_id, hour DESC);

-- ================================================
-- Daily Usage Aggregates (Billing Calculations)
-- ================================================

-- Daily usage aggregates for billing
CREATE MATERIALIZED VIEW usage_daily_billing AS
SELECT 
    date_trunc('day', time) AS day,
    org_id,
    subscription_id,
    subscription_item_id,
    data->>'usage_type' as usage_type,
    date_trunc('month', time) as billing_period,
    SUM(CAST(data->>'quantity' AS DECIMAL)) as daily_quantity,
    SUM(CAST(data->>'amount' AS BIGINT)) as daily_amount,
    COUNT(*) as daily_events,
    MIN(time) as first_event_time,
    MAX(time) as last_event_time
FROM usage_events
GROUP BY day, org_id, subscription_id, subscription_item_id, data->>'usage_type', billing_period;

-- Create indexes on daily billing view
CREATE UNIQUE INDEX idx_usage_daily_billing_unique ON usage_daily_billing (day, org_id, subscription_item_id, usage_type);
CREATE INDEX idx_usage_daily_billing_org_period ON usage_daily_billing (org_id, billing_period, subscription_item_id);
CREATE INDEX idx_usage_daily_billing_period ON usage_daily_billing (billing_period, org_id);

-- ================================================
-- Monthly Usage Summary (Analytics & Reporting)
-- ================================================

-- Monthly summary for analytics
CREATE MATERIALIZED VIEW usage_monthly_summary AS
SELECT 
    date_trunc('month', time) AS month,
    org_id,
    subscription_id,
    subscription_item_id,
    data->>'usage_type' as usage_type,
    SUM(CAST(data->>'quantity' AS DECIMAL)) as monthly_quantity,
    SUM(CAST(data->>'amount' AS BIGINT)) as monthly_amount,
    AVG(CAST(data->>'quantity' AS DECIMAL)) as avg_quantity,
    MAX(CAST(data->>'quantity' AS DECIMAL)) as max_quantity,
    COUNT(DISTINCT DATE(time)) as active_days,
    COUNT(*) as total_events
FROM usage_events
GROUP BY month, org_id, subscription_id, subscription_item_id, data->>'usage_type';

-- Create indexes on monthly summary
CREATE UNIQUE INDEX idx_usage_monthly_summary_unique ON usage_monthly_summary (month, org_id, subscription_item_id, usage_type);
CREATE INDEX idx_usage_monthly_summary_org ON usage_monthly_summary (org_id, month DESC);
CREATE INDEX idx_usage_monthly_summary_subscription ON usage_monthly_summary (subscription_id, month DESC);

-- ================================================
-- Customer Usage Summary (Customer Portal)
-- ================================================

-- Customer-level usage summary for customer portals
CREATE MATERIALIZED VIEW usage_customer_summary AS
SELECT 
    date_trunc('day', time) AS day,
    org_id,
    data->>'customer_id' as customer_id,
    subscription_id,
    data->>'usage_type' as usage_type,
    SUM(CAST(data->>'quantity' AS DECIMAL)) as daily_quantity,
    SUM(CAST(data->>'amount' AS BIGINT)) as daily_amount,
    COUNT(*) as daily_events,
    COUNT(DISTINCT subscription_item_id) as active_items
FROM usage_events
GROUP BY day, org_id, data->>'customer_id', subscription_id, data->>'usage_type';

-- Create indexes on customer summary
CREATE UNIQUE INDEX idx_usage_customer_summary_unique ON usage_customer_summary (day, org_id, customer_id, subscription_id, usage_type);
CREATE INDEX idx_usage_customer_summary_customer ON usage_customer_summary (customer_id, day DESC);
CREATE INDEX idx_usage_customer_summary_org_customer ON usage_customer_summary (org_id, customer_id, day DESC);

-- ================================================
-- Usage Type Analytics (Product Analytics)
-- ================================================

-- Usage type analytics for product insights
CREATE MATERIALIZED VIEW usage_type_analytics AS
SELECT 
    date_trunc('day', time) AS day,
    org_id,
    data->>'usage_type' as usage_type,
    COUNT(DISTINCT data->>'customer_id') as unique_customers,
    COUNT(DISTINCT subscription_id) as unique_subscriptions,
    COUNT(DISTINCT subscription_item_id) as unique_items,
    SUM(CAST(data->>'quantity' AS DECIMAL)) as total_quantity,
    SUM(CAST(data->>'amount' AS BIGINT)) as total_amount,
    AVG(CAST(data->>'quantity' AS DECIMAL)) as avg_quantity,
    COUNT(*) as total_events
FROM usage_events
GROUP BY day, org_id, data->>'usage_type';

-- Create indexes on usage type analytics
CREATE UNIQUE INDEX idx_usage_type_analytics_unique ON usage_type_analytics (day, org_id, usage_type);
CREATE INDEX idx_usage_type_analytics_org ON usage_type_analytics (org_id, day DESC);
CREATE INDEX idx_usage_type_analytics_type ON usage_type_analytics (usage_type, day DESC);

-- ================================================
-- Materialized View Management Functions
-- ================================================

-- Function to refresh all materialized views (called by scheduler)
CREATE OR REPLACE FUNCTION refresh_usage_aggregates()
RETURNS jsonb AS $$
DECLARE
    start_time timestamp;
    end_time timestamp;
    refresh_result jsonb;
BEGIN
    start_time := clock_timestamp();

    -- Refresh materialized views concurrently (non-blocking)
    REFRESH MATERIALIZED VIEW CONCURRENTLY usage_hourly;
    REFRESH MATERIALIZED VIEW CONCURRENTLY usage_daily_billing;
    REFRESH MATERIALIZED VIEW CONCURRENTLY usage_monthly_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY usage_customer_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY usage_type_analytics;

    end_time := clock_timestamp();

    -- Build result object
    refresh_result := jsonb_build_object(
        'start_time', start_time,
        'end_time', end_time,
        'duration_seconds', extract(epoch from (end_time - start_time)),
        'views_refreshed', array[
            'usage_hourly',
            'usage_daily_billing', 
            'usage_monthly_summary',
            'usage_customer_summary',
            'usage_type_analytics'
        ]
    );

    -- Log the refresh
    INSERT INTO usage_event_log (
        org_id,
        event_type,
        triggered_by,
        reason,
        metadata
    ) VALUES (
        'system',
        'materialized_views_refreshed',
        'scheduler',
        'Materialized views refreshed for real-time analytics',
        refresh_result
    );

    RETURN refresh_result;
END;
$$ LANGUAGE plpgsql;

-- Function to get materialized view statistics
CREATE OR REPLACE FUNCTION get_materialized_view_stats()
RETURNS TABLE (
    view_name text,
    row_count bigint,
    size_bytes bigint,
    size_pretty text,
    last_refresh timestamp
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        c.relname::text as view_name,
        c.reltuples::bigint as row_count,
        pg_total_relation_size(c.oid) as size_bytes,
        pg_size_pretty(pg_total_relation_size(c.oid)) as size_pretty,
        COALESCE(s.last_refresh, NULL) as last_refresh
    FROM pg_class c
    LEFT JOIN pg_stat_user_tables s ON s.relname = c.relname
    WHERE c.relkind = 'm'  -- materialized views
    AND c.relname LIKE 'usage_%'
    ORDER BY c.relname;
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- Billing Integration Functions
-- ================================================

-- Function to get billing summary from materialized views
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
      AND udb.billing_period = date_trunc('month', p_billing_period)
    GROUP BY udb.subscription_id, udb.subscription_item_id, udb.usage_type
    ORDER BY udb.subscription_id, udb.subscription_item_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get customer usage summary
CREATE OR REPLACE FUNCTION get_customer_usage_summary(
    p_org_id TEXT,
    p_customer_id TEXT,
    p_start_date DATE,
    p_end_date DATE
)
RETURNS TABLE (
    day DATE,
    subscription_id TEXT,
    usage_type TEXT,
    daily_quantity NUMERIC,
    daily_amount BIGINT,
    daily_events BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        ucs.day::date,
        ucs.subscription_id,
        ucs.usage_type,
        ucs.daily_quantity,
        ucs.daily_amount,
        ucs.daily_events
    FROM usage_customer_summary ucs
    WHERE ucs.org_id = p_org_id
      AND ucs.customer_id = p_customer_id
      AND ucs.day >= p_start_date
      AND ucs.day <= p_end_date
    ORDER BY ucs.day DESC, ucs.subscription_id, ucs.usage_type;
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- Automated Refresh Schedule Setup
-- ================================================

-- Function to set up materialized view refresh schedule
-- Note: pg_cron is NOT available on AWS RDS - use alternative scheduling
CREATE OR REPLACE FUNCTION setup_materialized_view_refresh_job()
RETURNS text AS $$
DECLARE
    job_result text;
BEGIN
    -- Check if pg_cron extension is available (not on AWS RDS)
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_cron') THEN
        -- Enable pg_cron if not already enabled
        CREATE EXTENSION IF NOT EXISTS pg_cron;

        -- Schedule refresh every 5 minutes
        SELECT cron.schedule(
            'usage-materialized-view-refresh',
            '*/5 * * * *',
            'SELECT refresh_usage_aggregates();'
        ) INTO job_result;

        RETURN 'Scheduled pg_cron job for materialized view refresh: ' || job_result;
    ELSE
        RETURN 'pg_cron extension not available (e.g., AWS RDS). Use alternative scheduling:'||
               ' 1) Application-level scheduler, 2) AWS Lambda + EventBridge every 5 minutes';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RETURN 'Could not set up pg_cron job: ' || SQLERRM || '. Use alternative scheduling method.';
END;
$$ LANGUAGE plpgsql;

-- Try to set up automated refresh (will show alternative options on RDS)
SELECT setup_materialized_view_refresh_job();

-- ================================================
-- Verification and Initial Population
-- ================================================

-- Initial population of materialized views (if data exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM usage_events LIMIT 1) THEN
        RAISE NOTICE 'Populating materialized views with existing data...';
        PERFORM refresh_usage_aggregates();
        RAISE NOTICE 'Materialized views populated successfully';
    ELSE
        RAISE NOTICE 'No usage data found. Materialized views will be populated as data arrives.';
    END IF;
END $$;

-- Show materialized view status
SELECT 
    'MATERIALIZED VIEW STATUS' as info_type,
    view_name,
    row_count,
    size_pretty,
    last_refresh
FROM get_materialized_view_stats()
ORDER BY view_name;

-- ================================================
-- Documentation and Comments
-- ================================================

COMMENT ON MATERIALIZED VIEW usage_hourly IS 
'Hourly usage aggregates for real-time dashboards. Refreshed every 5 minutes.';

COMMENT ON MATERIALIZED VIEW usage_daily_billing IS 
'Daily usage aggregates optimized for billing calculations and monthly summaries.';

COMMENT ON MATERIALIZED VIEW usage_monthly_summary IS 
'Monthly usage statistics for analytics and reporting dashboards.';

COMMENT ON MATERIALIZED VIEW usage_customer_summary IS 
'Customer-level daily usage summary for customer portals and self-service analytics.';

COMMENT ON MATERIALIZED VIEW usage_type_analytics IS 
'Usage type analytics for product insights and feature utilization tracking.';

COMMENT ON FUNCTION refresh_usage_aggregates() IS 
'Refreshes all usage-related materialized views. Should be run every 5 minutes via cron.';

COMMENT ON FUNCTION get_monthly_billing_summary(text, date) IS 
'Get aggregated usage data for billing calculations for a specific organization and billing period.';

-- Print final status
DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'PostgreSQL Usage Database Materialized Views Setup Complete';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Views created: usage_hourly, usage_daily_billing, usage_monthly_summary';
    RAISE NOTICE 'Customer views: usage_customer_summary, usage_type_analytics';
    RAISE NOTICE 'Automated refresh: Set up external scheduling (see above message)';
    RAISE NOTICE 'Manual refresh: SELECT refresh_usage_aggregates();';
    RAISE NOTICE 'View stats: SELECT * FROM get_materialized_view_stats();';
    RAISE NOTICE '================================================';
END $$;
