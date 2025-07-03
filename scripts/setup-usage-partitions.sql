-- ================================================
-- PostgreSQL Usage Database - Time-Based Partitioning Setup
-- ================================================
-- This script sets up time-based partitioning for usage_events table
-- Starting from July 2025 with automatic partition management

-- ================================================
-- Partition Management Functions
-- ================================================

-- Function to create a monthly partition
CREATE OR REPLACE FUNCTION create_monthly_partition(
    table_name text, 
    partition_date date
)
RETURNS text AS $$
DECLARE
    partition_name text;
    start_date date;
    end_date date;
    sql_statement text;
BEGIN
    -- Calculate partition boundaries (first day of month to first day of next month)
    start_date := date_trunc('month', partition_date)::date;
    end_date := (start_date + interval '1 month')::date;
    
    -- Generate partition name (e.g., usage_events_2025_07)
    partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM');
    
    -- Check if partition already exists
    IF EXISTS (
        SELECT 1 FROM pg_tables 
        WHERE schemaname = 'public' 
        AND tablename = partition_name
    ) THEN
        RAISE NOTICE 'Partition % already exists, skipping creation', partition_name;
        RETURN partition_name;
    END IF;
    
    -- Create the partition
    sql_statement := format(
        'CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
        partition_name, table_name, start_date, end_date
    );
    
    EXECUTE sql_statement;
    
    -- Create partition-specific indexes for better performance
    EXECUTE format(
        'CREATE INDEX %I ON %I (org_id, subscription_item_id, time DESC)',
        'idx_' || partition_name || '_org_item_time', partition_name
    );
    
    EXECUTE format(
        'CREATE INDEX %I ON %I (time DESC)',
        'idx_' || partition_name || '_time', partition_name
    );
    
    EXECUTE format(
        'CREATE INDEX %I ON %I (reference_id, reference_type) WHERE reference_id IS NOT NULL',
        'idx_' || partition_name || '_reference', partition_name
    );
    
    RAISE NOTICE 'Created partition % for period % to %', partition_name, start_date, end_date;
    RETURN partition_name;
END;
$$ LANGUAGE plpgsql;

-- Function to ensure partitions exist for a range of months
CREATE OR REPLACE FUNCTION ensure_partitions_exist(
    table_name text,
    start_date date,
    months_ahead integer DEFAULT 6
)
RETURNS text[] AS $$
DECLARE
    current_date date;
    created_partitions text[] := '{}';
    partition_name text;
    i integer;
BEGIN
    current_date := date_trunc('month', start_date)::date;
    
    FOR i IN 0..months_ahead LOOP
        partition_name := create_monthly_partition(
            table_name, 
            current_date + (i || ' months')::interval
        );
        created_partitions := array_append(created_partitions, partition_name);
    END LOOP;
    
    RETURN created_partitions;
END;
$$ LANGUAGE plpgsql;

-- Function to drop old partitions based on retention policy
CREATE OR REPLACE FUNCTION cleanup_old_partitions(
    table_name text,
    retention_months integer DEFAULT 60 -- 5 years default
)
RETURNS text[] AS $$
DECLARE
    cutoff_date date;
    partition_record record;
    partition_name text;
    dropped_partitions text[] := '{}';
    partition_date date;
BEGIN
    cutoff_date := date_trunc('month', CURRENT_DATE - (retention_months || ' months')::interval)::date;
    
    -- Find partitions to drop
    FOR partition_record IN 
        SELECT tablename 
        FROM pg_tables 
        WHERE schemaname = 'public'
        AND tablename LIKE table_name || '_%' 
        AND tablename ~ '^' || table_name || '_[0-9]{4}_[0-9]{2}$'
    LOOP
        partition_name := partition_record.tablename;
        
        -- Extract date from partition name (e.g., usage_events_2020_01 -> 2020-01-01)
        BEGIN
            partition_date := to_date(
                substring(partition_name from '([0-9]{4}_[0-9]{2})$'), 
                'YYYY_MM'
            );
            
            -- Drop if older than cutoff
            IF partition_date < cutoff_date THEN
                EXECUTE format('DROP TABLE IF EXISTS %I', partition_name);
                dropped_partitions := array_append(dropped_partitions, partition_name);
                RAISE NOTICE 'Dropped old partition: %', partition_name;
            END IF;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE WARNING 'Could not parse partition date from %: %', partition_name, SQLERRM;
        END;
    END LOOP;
    
    RETURN dropped_partitions;
END;
$$ LANGUAGE plpgsql;

-- Function for comprehensive partition maintenance
CREATE OR REPLACE FUNCTION maintain_usage_partitions(
    table_name text DEFAULT 'usage_events',
    months_ahead integer DEFAULT 6,
    retention_months integer DEFAULT 60
)
RETURNS jsonb AS $$
DECLARE
    created_partitions text[];
    dropped_partitions text[];
    maintenance_result jsonb;
    current_month date;
BEGIN
    current_month := date_trunc('month', CURRENT_DATE)::date;
    
    -- Create future partitions
    created_partitions := ensure_partitions_exist(table_name, current_month, months_ahead);
    
    -- Clean up old partitions
    dropped_partitions := cleanup_old_partitions(table_name, retention_months);
    
    -- Update table statistics for better query planning
    EXECUTE format('ANALYZE %I', table_name);
    
    -- Build result
    maintenance_result := jsonb_build_object(
        'timestamp', NOW(),
        'table_name', table_name,
        'created_partitions', to_jsonb(created_partitions),
        'dropped_partitions', to_jsonb(dropped_partitions),
        'months_ahead', months_ahead,
        'retention_months', retention_months
    );
    
    -- Log maintenance activity
    INSERT INTO usage_event_log (
        org_id, event_type, triggered_by, reason, metadata
    ) VALUES (
        'system', 'partition_maintenance', 'auto_maintenance',
        'Automated partition maintenance completed',
        maintenance_result
    );
    
    RETURN maintenance_result;
END;
$$ LANGUAGE plpgsql;

-- Function to get partition information for monitoring
CREATE OR REPLACE FUNCTION get_partition_info(table_name text DEFAULT 'usage_events')
RETURNS TABLE (
    partition_name text,
    partition_bounds text,
    start_date date,
    end_date date,
    row_count bigint,
    size_bytes bigint,
    size_pretty text
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.tablename::text,
        pg_get_expr(c.relpartbound, c.oid) as bounds,
        -- Extract start date from partition name
        CASE 
            WHEN t.tablename ~ '_[0-9]{4}_[0-9]{2}$' THEN
                to_date(substring(t.tablename from '([0-9]{4}_[0-9]{2})$'), 'YYYY_MM')
            ELSE NULL
        END as start_date,
        -- Calculate end date (start of next month)
        CASE 
            WHEN t.tablename ~ '_[0-9]{4}_[0-9]{2}$' THEN
                (to_date(substring(t.tablename from '([0-9]{4}_[0-9]{2})$'), 'YYYY_MM') + interval '1 month')::date
            ELSE NULL
        END as end_date,
        COALESCE(s.n_tup_ins - s.n_tup_del, 0) as row_count,
        pg_total_relation_size(c.oid) as size_bytes,
        pg_size_pretty(pg_total_relation_size(c.oid)) as size_pretty
    FROM pg_tables t
    JOIN pg_class c ON c.relname = t.tablename
    LEFT JOIN pg_stat_user_tables s ON s.relname = t.tablename
    WHERE t.schemaname = 'public'
    AND t.tablename LIKE table_name || '%'
    AND t.tablename ~ '^' || table_name || '(_[0-9]{4}_[0-9]{2})?$'
    ORDER BY 
        CASE WHEN t.tablename = table_name THEN 0 ELSE 1 END,
        start_date NULLS FIRST;
END;
$$ LANGUAGE plpgsql;

-- ================================================
-- Initialize Partitions for July 2025 onwards
-- ================================================

-- Create initial partitions starting from July 2025
-- This creates partitions for July 2025 through January 2026 (7 months)
DO $$
DECLARE
    created_partitions text[];
    start_date date := '2025-07-01'::date;
BEGIN
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Initializing Usage Database Partitions';
    RAISE NOTICE 'Starting from: %', start_date;
    RAISE NOTICE '================================================';
    
    -- Ensure the main table exists (should be created by Prisma)
    IF NOT EXISTS (
        SELECT 1 FROM pg_tables 
        WHERE schemaname = 'public' 
        AND tablename = 'usage_events'
    ) THEN
        RAISE EXCEPTION 'Table usage_events does not exist. Please run Prisma migrations first.';
    END IF;
    
    -- Create partitions for July 2025 and next 6 months
    created_partitions := ensure_partitions_exist('usage_events', start_date, 6);
    
    RAISE NOTICE 'Created % partitions: %', array_length(created_partitions, 1), created_partitions;
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Partition initialization completed successfully';
    RAISE NOTICE '================================================';
END $$;

-- ================================================
-- Automated Maintenance Schedule Setup
-- ================================================

-- Function to set up pg_cron job (if pg_cron extension is available)
CREATE OR REPLACE FUNCTION setup_partition_maintenance_job()
RETURNS text AS $$
DECLARE
    job_result text;
BEGIN
    -- Check if pg_cron extension is available
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_cron') THEN
        -- Enable pg_cron if not already enabled
        CREATE EXTENSION IF NOT EXISTS pg_cron;
        
        -- Schedule maintenance to run on the 1st of every month at 2:00 AM
        SELECT cron.schedule(
            'usage-partition-maintenance',
            '0 2 1 * *',
            'SELECT maintain_usage_partitions();'
        ) INTO job_result;
        
        RETURN 'Scheduled pg_cron job: ' || job_result;
    ELSE
        RETURN 'pg_cron extension not available. Set up external cron job to run: SELECT maintain_usage_partitions();';
    END IF;
EXCEPTION
    WHEN OTHERS THEN
        RETURN 'Could not set up pg_cron job: ' || SQLERRM || '. Set up external cron job instead.';
END;
$$ LANGUAGE plpgsql;

-- Try to set up automated maintenance
SELECT setup_partition_maintenance_job();

-- ================================================
-- Verification and Documentation
-- ================================================

-- Show current partition status
SELECT 
    'PARTITION STATUS' as info_type,
    partition_name,
    start_date,
    end_date,
    size_pretty
FROM get_partition_info('usage_events')
WHERE partition_name != 'usage_events'
ORDER BY start_date;

-- Show maintenance function availability
SELECT 
    'MAINTENANCE FUNCTIONS' as info_type,
    routine_name as function_name,
    'Available' as status
FROM information_schema.routines 
WHERE routine_schema = 'public' 
AND routine_name IN (
    'maintain_usage_partitions',
    'create_monthly_partition',
    'ensure_partitions_exist',
    'cleanup_old_partitions',
    'get_partition_info'
)
ORDER BY routine_name;

-- ================================================
-- Usage Instructions
-- ================================================

COMMENT ON FUNCTION maintain_usage_partitions(text, integer, integer) IS 
'Comprehensive partition maintenance function. Run monthly via cron.
Example: SELECT maintain_usage_partitions(''usage_events'', 6, 60);';

COMMENT ON FUNCTION get_partition_info(text) IS 
'Get detailed information about table partitions for monitoring.
Example: SELECT * FROM get_partition_info(''usage_events'');';

COMMENT ON FUNCTION create_monthly_partition(text, date) IS 
'Create a single monthly partition for the given date.
Example: SELECT create_monthly_partition(''usage_events'', ''2025-08-01'');';

-- Print final status
DO $$
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'PostgreSQL Usage Database Partitioning Setup Complete';
    RAISE NOTICE '================================================';
    RAISE NOTICE 'Partitions created for: July 2025 - January 2026';
    RAISE NOTICE 'Automated maintenance: Check pg_cron setup above';
    RAISE NOTICE 'Manual maintenance: SELECT maintain_usage_partitions();';
    RAISE NOTICE 'Monitor partitions: SELECT * FROM get_partition_info();';
    RAISE NOTICE '================================================';
END $$;