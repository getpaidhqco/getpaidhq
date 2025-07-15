# Prometheus Metrics Implementation Specification

## Overview

This specification outlines the implementation of Prometheus metrics for system monitoring and tenant-specific API usage tracking in the Payloop DDD-based Go application. The implementation focuses on HTTP performance metrics and org_id-segmented API usage monitoring without billing/revenue tracking.

## Architecture Requirements

### Domain-Driven Design Compliance
- Follow existing DDD patterns and clean architecture principles
- Metrics collection should not affect domain layer purity
- Use dependency injection with Uber FX
- Implement interfaces in application layer, concrete implementations in infrastructure layer

### Metrics Categories

#### 1. System Performance Metrics (System-wide)
- HTTP request performance and status codes
- Database query performance
- Application health indicators

#### 2. Usage Monitoring Metrics (Tenant-segmented by org_id)
- API usage patterns per organization
- Subscription operations tracking
- Usage records processing metrics

## Implementation Structure

### 1. Domain Layer (`internal/domain/`)

#### 1.1 Metrics Repository Interface
**File:** `internal/domain/repositories/metrics_repository.go`

```go
package repositories

import (
    "context"
    "time"
)

// MetricsRepository defines the interface for metrics collection
type MetricsRepository interface {
    // HTTP Metrics
    RecordHTTPRequest(method, route, statusCode string, duration time.Duration)
    RecordHTTPRequestInFlight(delta int)
    
    // Tenant Usage Metrics
    RecordAPIUsage(ctx context.Context, orgId, endpoint string)
    RecordSubscriptionOperation(ctx context.Context, orgId, operation string)
    RecordUsageRecordProcessing(ctx context.Context, orgId, unitType string, count int)
    
    // System Health Metrics
    RecordServiceHealth(service string, healthy bool)
    RecordDatabaseQuery(operation, table string, duration time.Duration)
    RecordError(service, errorType string)
    
    // Gauge Updates
    UpdateActiveSubscriptions(orgId, status string, delta int)
    UpdateDatabaseConnections(pool, state string, value int)
}
```

### 2. Application Layer (`internal/application/`)

#### 2.1 Metrics Service Interface
**File:** `internal/application/interfaces/metrics_service.go`

```go
package interfaces

import (
    "context"
    "time"
)

// MetricsService provides high-level metrics operations
type MetricsService interface {
    // HTTP Metrics
    TrackHTTPRequest(method, route, statusCode string, duration time.Duration)
    
    // Business Operation Metrics
    TrackSubscriptionCreated(ctx context.Context, orgId string)
    TrackSubscriptionUpdated(ctx context.Context, orgId, operation string)
    TrackUsageAggregation(ctx context.Context, orgId, unitType string, recordCount int, duration time.Duration)
    
    // System Health
    ReportServiceHealth(service string, healthy bool)
    ReportDatabasePerformance(operation, table string, duration time.Duration)
    
    // Error Tracking
    ReportError(ctx context.Context, service, errorType string)
}
```

#### 2.2 Metrics Service Implementation
**File:** `internal/application/services/metrics_service.go`

```go
package services

import (
    "context"
    "payloop/internal/application/interfaces"
    "payloop/internal/application/lib/logger"
    "payloop/internal/domain/repositories"
    "time"
)

type MetricsService struct {
    metricsRepository repositories.MetricsRepository
    logger           logger.Logger
}

func NewMetricsService(
    metricsRepository repositories.MetricsRepository,
    logger logger.Logger,
) interfaces.MetricsService {
    return &MetricsService{
        metricsRepository: metricsRepository,
        logger:           logger,
    }
}

func (m *MetricsService) TrackHTTPRequest(method, route, statusCode string, duration time.Duration) {
    m.metricsRepository.RecordHTTPRequest(method, route, statusCode, duration)
}

func (m *MetricsService) TrackSubscriptionCreated(ctx context.Context, orgId string) {
    m.metricsRepository.RecordSubscriptionOperation(ctx, orgId, "create")
    m.metricsRepository.UpdateActiveSubscriptions(orgId, "active", 1)
}

func (m *MetricsService) TrackSubscriptionUpdated(ctx context.Context, orgId, operation string) {
    m.metricsRepository.RecordSubscriptionOperation(ctx, orgId, operation)
}

func (m *MetricsService) TrackUsageAggregation(ctx context.Context, orgId, unitType string, recordCount int, duration time.Duration) {
    m.metricsRepository.RecordUsageRecordProcessing(ctx, orgId, unitType, recordCount)
    // Record aggregation performance
    m.logger.Debugf("Usage aggregation completed for org %s: %d records in %v", orgId, recordCount, duration)
}

func (m *MetricsService) ReportServiceHealth(service string, healthy bool) {
    m.metricsRepository.RecordServiceHealth(service, healthy)
}

func (m *MetricsService) ReportDatabasePerformance(operation, table string, duration time.Duration) {
    m.metricsRepository.RecordDatabaseQuery(operation, table, duration)
}

func (m *MetricsService) ReportError(ctx context.Context, service, errorType string) {
    m.metricsRepository.RecordError(service, errorType)
}
```

### 3. Infrastructure Layer (`internal/infrastructure/`)

#### 3.1 Prometheus Client Implementation
**File:** `internal/infrastructure/metrics/prometheus_repository.go`

```go
package metrics

import (
    "context"
    "strconv"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "payloop/internal/domain/repositories"
)

type PrometheusRepository struct {
    // HTTP Metrics
    httpRequestsTotal    *prometheus.CounterVec
    httpRequestDuration  *prometheus.HistogramVec
    httpRequestsInFlight prometheus.Gauge

    // Tenant Usage Metrics
    apiUsageTotal             *prometheus.CounterVec
    subscriptionOperationsTotal *prometheus.CounterVec
    usageRecordsProcessedTotal  *prometheus.CounterVec
    
    // System Metrics
    serviceHealth         *prometheus.GaugeVec
    databaseQueryDuration *prometheus.HistogramVec
    errorsTotal          *prometheus.CounterVec
    
    // Gauge Metrics
    activeSubscriptions   *prometheus.GaugeVec
    databaseConnections   *prometheus.GaugeVec
}

func NewPrometheusRepository() repositories.MetricsRepository {
    return &PrometheusRepository{
        // HTTP Metrics
        httpRequestsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "payloop_http_requests_total",
                Help: "Total number of HTTP requests",
            },
            []string{"method", "route", "status_code"},
        ),
        httpRequestDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "payloop_http_request_duration_seconds",
                Help:    "HTTP request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "route"},
        ),
        httpRequestsInFlight: promauto.NewGauge(
            prometheus.GaugeOpts{
                Name: "payloop_http_requests_in_flight",
                Help: "Current number of HTTP requests being processed",
            },
        ),

        // Tenant Usage Metrics
        apiUsageTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "payloop_api_usage_total",
                Help: "Total API calls per organization and endpoint",
            },
            []string{"org_id", "endpoint"},
        ),
        subscriptionOperationsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "payloop_subscription_operations_total",
                Help: "Total subscription operations per organization",
            },
            []string{"org_id", "operation"},
        ),
        usageRecordsProcessedTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "payloop_usage_records_processed_total",
                Help: "Total usage records processed per organization",
            },
            []string{"org_id", "unit_type"},
        ),

        // System Metrics
        serviceHealth: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "payloop_service_health",
                Help: "Service health status (1 = healthy, 0 = unhealthy)",
            },
            []string{"service"},
        ),
        databaseQueryDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "payloop_database_query_duration_seconds",
                Help:    "Database query duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"operation", "table"},
        ),
        errorsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "payloop_errors_total",
                Help: "Total number of errors by service and type",
            },
            []string{"service", "error_type"},
        ),

        // Gauge Metrics
        activeSubscriptions: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "payloop_active_subscriptions",
                Help: "Number of active subscriptions per organization",
            },
            []string{"org_id", "status"},
        ),
        databaseConnections: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Name: "payloop_database_connections",
                Help: "Number of database connections by pool and state",
            },
            []string{"pool", "state"},
        ),
    }
}

// HTTP Metrics Implementation
func (p *PrometheusRepository) RecordHTTPRequest(method, route, statusCode string, duration time.Duration) {
    p.httpRequestsTotal.WithLabelValues(method, route, statusCode).Inc()
    p.httpRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}

func (p *PrometheusRepository) RecordHTTPRequestInFlight(delta int) {
    p.httpRequestsInFlight.Add(float64(delta))
}

// Tenant Usage Metrics Implementation
func (p *PrometheusRepository) RecordAPIUsage(ctx context.Context, orgId, endpoint string) {
    p.apiUsageTotal.WithLabelValues(orgId, endpoint).Inc()
}

func (p *PrometheusRepository) RecordSubscriptionOperation(ctx context.Context, orgId, operation string) {
    p.subscriptionOperationsTotal.WithLabelValues(orgId, operation).Inc()
}

func (p *PrometheusRepository) RecordUsageRecordProcessing(ctx context.Context, orgId, unitType string, count int) {
    p.usageRecordsProcessedTotal.WithLabelValues(orgId, unitType).Add(float64(count))
}

// System Health Metrics Implementation
func (p *PrometheusRepository) RecordServiceHealth(service string, healthy bool) {
    value := 0.0
    if healthy {
        value = 1.0
    }
    p.serviceHealth.WithLabelValues(service).Set(value)
}

func (p *PrometheusRepository) RecordDatabaseQuery(operation, table string, duration time.Duration) {
    p.databaseQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

func (p *PrometheusRepository) RecordError(service, errorType string) {
    p.errorsTotal.WithLabelValues(service, errorType).Inc()
}

// Gauge Updates
func (p *PrometheusRepository) UpdateActiveSubscriptions(orgId, status string, delta int) {
    p.activeSubscriptions.WithLabelValues(orgId, status).Add(float64(delta))
}

func (p *PrometheusRepository) UpdateDatabaseConnections(pool, state string, value int) {
    p.databaseConnections.WithLabelValues(pool, state).Set(float64(value))
}
```

#### 3.2 Prometheus HTTP Middleware
**File:** `internal/infrastructure/metrics/http_middleware.go`

```go
package metrics

import (
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "payloop/internal/application/interfaces"
    "payloop/internal/application/lib/logger"
)

type PrometheusMiddleware struct {
    metricsService interfaces.MetricsService
    logger        logger.Logger
}

func NewPrometheusMiddleware(
    metricsService interfaces.MetricsService,
    logger logger.Logger,
) PrometheusMiddleware {
    return PrometheusMiddleware{
        metricsService: metricsService,
        logger:        logger,
    }
}

func (m PrometheusMiddleware) Setup() {
    // This middleware is applied globally, setup is handled by the middleware system
}

func (m PrometheusMiddleware) Handler() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        // Increment in-flight requests
        // Note: This would need to be implemented in the metrics service if needed
        
        // Process request
        c.Next()
        
        // Calculate duration
        duration := time.Since(start)
        
        // Record HTTP metrics
        m.metricsService.TrackHTTPRequest(
            c.Request.Method,
            c.FullPath(),
            strconv.Itoa(c.Writer.Status()),
            duration,
        )
        
        // Record API usage if org_id is available in context
        if orgId := extractOrgIdFromContext(c); orgId != "" {
            // Record tenant-specific API usage
            endpoint := c.FullPath()
            if endpoint == "" {
                endpoint = c.Request.URL.Path
            }
            
            ctx := c.Request.Context()
            // This would be implemented in the repository layer
            // m.metricsService.TrackAPIUsage(ctx, orgId, endpoint)
        }
    }
}

// extractOrgIdFromContext extracts org_id from the Gin context
// This should align with the existing authentication middleware
func extractOrgIdFromContext(c *gin.Context) string {
    // Implementation depends on how org_id is stored in context
    // Common patterns:
    
    // Option 1: From JWT claims in context
    if claims, exists := c.Get("claims"); exists {
        if claimsMap, ok := claims.(map[string]interface{}); ok {
            if orgId, ok := claimsMap["org_id"].(string); ok {
                return orgId
            }
        }
    }
    
    // Option 2: From user context
    if user, exists := c.Get("user"); exists {
        if userObj, ok := user.(interface{ GetOrgId() string }); ok {
            return userObj.GetOrgId()
        }
    }
    
    // Option 3: From query parameter or header (fallback)
    if orgId := c.GetHeader("X-Org-Id"); orgId != "" {
        return orgId
    }
    
    return ""
}
```

#### 3.3 Metrics Module for FX DI
**File:** `internal/infrastructure/metrics/module.go`

```go
package metrics

import (
    "go.uber.org/fx"
    "payloop/internal/application/interfaces"
    "payloop/internal/domain/repositories"
)

// Module exports the metrics dependencies for FX DI container
var Module = fx.Options(
    // Repository
    fx.Provide(fx.Annotate(
        NewPrometheusRepository,
        fx.As(new(repositories.MetricsRepository)),
    )),
    
    // Application Service
    fx.Provide(fx.Annotate(
        NewMetricsService,
        fx.As(new(interfaces.MetricsService)),
    )),
    
    // Middleware
    fx.Provide(NewPrometheusMiddleware),
)
```

### 4. Middleware Integration

#### 4.1 Update Middlewares Module
**File:** `internal/api/middlewares/middlewares.go` (Update existing file)

Add to the existing middlewares constructor:

```go
// Add to imports
import (
    "payloop/internal/infrastructure/metrics"
)

// Update NewMiddlewares function to include PrometheusMiddleware
func NewMiddlewares(
    corsMiddleware CorsMiddleware,
    dbTrxMiddleware DatabaseTrx,
    authMiddleware AuthnWrapperMiddleware,
    authzMiddleware cedar.CedarMiddleware,
    sentryMiddleware SentryMiddleware,
    prometheusMiddleware metrics.PrometheusMiddleware, // Add this
) Middlewares {
    return Middlewares{
        corsMiddleware,
        dbTrxMiddleware,
        authMiddleware,
        authzMiddleware,
        sentryMiddleware,
        prometheusMiddleware, // Add this
    }
}
```

#### 4.2 Update Request Handler
**File:** `internal/lib/request_handler.go` (Update existing file)

Add Prometheus middleware to the Gin engine:

```go
// Add to NewRequestHandler function after existing middleware setup:

// Add Prometheus middleware
if prometheusMiddleware, exists := c.Get("prometheusMiddleware"); exists {
    if pm, ok := prometheusMiddleware.(func() gin.HandlerFunc); ok {
        engine.Use(pm())
    }
}
```

### 5. Bootstrap Integration

#### 5.1 Update Bootstrap Modules
**File:** `internal/application/bootstrap/modules.go` (Update existing file)

Add metrics module to the CommonModules:

```go
import (
    "payloop/internal/infrastructure/metrics"
)

var CommonModules = fx.Options(
    // ... existing modules ...
    metrics.Module,
)
```

### 6. Service Integration Examples

#### 6.1 Update Subscription Service
**File:** `internal/application/services/subscription_service.go` (Update existing)

Add metrics tracking to key operations:

```go
// Add to constructor
func NewSubscriptionService(
    // ... existing parameters ...
    metricsService interfaces.MetricsService, // Add this
) interfaces.SubscriptionService {
    // ... existing code ...
    
    return SubscriptionService{
        // ... existing fields ...
        metricsService: metricsService, // Add this
    }
}

// Update Create method
func (s SubscriptionService) Create(ctx context.Context, input entities.CreateSubscriptionInput) (entities.Subscription, error) {
    s.logger.Info("Creating new subscription", "orgId", input.OrgId)

    subscription := entities.NewFromCreateInput(input)
    subscription, err := s.subscriptionRepository.Create(ctx, subscription)

    if err != nil {
        s.logger.Error("Failed create subscriptions", err.Error())
        s.metricsService.ReportError(ctx, "subscription_service", "create_failed")
        return entities.Subscription{}, err
    }

    // Track successful subscription creation
    s.metricsService.TrackSubscriptionCreated(ctx, input.OrgId)

    _ = s.pubsub.Publish(subscription.OrgId, topic.TopicSubscriptionCreated, subscription)

    return subscription, nil
}

// Update other methods similarly...
```

#### 6.2 Update Billing Service
**File:** `internal/application/services/billing_service.go` (Update existing)

Add usage aggregation metrics:

```go
// Add to constructor and aggregateUsage method
func (b *BillingService) aggregateUsage(records []entities.UsageRecord, aggregationType entities.AggregationType) float64 {
    start := time.Now()
    
    // ... existing aggregation logic ...
    
    // Track usage aggregation metrics
    if len(records) > 0 {
        duration := time.Since(start)
        b.metricsService.TrackUsageAggregation(
            context.Background(), // or get from caller
            records[0].OrgId,
            string(records[0].UnitType),
            len(records),
            duration,
        )
    }
    
    return result
}
```

### 7. Prometheus Endpoint

#### 7.1 Health Routes Update
**File:** `internal/api/routes/health_routes.go` (Update existing)

Add Prometheus metrics endpoint:

```go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func (h HealthRoutes) Setup() {
    healthAPI := h.handler.Gin.Group("/health")
    {
        healthAPI.GET("/", h.controller.Health)
        healthAPI.GET("/ready", h.controller.Ready)
        // Add Prometheus metrics endpoint
        healthAPI.GET("/metrics", gin.WrapH(promhttp.Handler()))
    }
}
```

## Implementation Checklist

### Phase 1: Core Infrastructure
- [ ] Create domain repository interface (`metrics_repository.go`)
- [ ] Create application service interface (`metrics_service.go`)
- [ ] Implement Prometheus repository (`prometheus_repository.go`)
- [ ] Implement metrics service (`metrics_service.go`)
- [ ] Create FX DI module (`module.go`)

### Phase 2: HTTP Middleware
- [ ] Implement Prometheus middleware (`http_middleware.go`)
- [ ] Update middlewares module to include Prometheus middleware
- [ ] Update request handler to apply Prometheus middleware
- [ ] Update bootstrap modules to include metrics module

### Phase 3: Service Integration
- [ ] Update subscription service with metrics tracking
- [ ] Update billing service with usage aggregation metrics
- [ ] Add metrics endpoint to health routes
- [ ] Test org_id extraction from authentication context

### Phase 4: Testing & Validation
- [ ] Verify metrics are collected at `/health/metrics` endpoint
- [ ] Test tenant-specific metrics with different org_ids
- [ ] Validate HTTP performance metrics
- [ ] Test error tracking and service health metrics

## Dependencies

Add to `go.mod`:
```go
require (
    github.com/prometheus/client_golang v1.17.0
)
```

## Configuration

No additional configuration required. Metrics will be available at:
- **Metrics Endpoint:** `http://localhost:8081/health/metrics`
- **Default Scrape Interval:** 15s (configurable in Prometheus config)

## Notes

1. **Org ID Extraction:** The `extractOrgIdFromContext` function needs to be implemented based on the existing authentication middleware pattern.

2. **Error Handling:** All metrics operations should be non-blocking and should not affect business logic flow.

3. **Performance:** Prometheus metrics have minimal performance overhead but should be monitored in production.

4. **Cardinality:** Monitor the number of unique org_ids to ensure metric cardinality remains manageable.

5. **Labels:** Keep label cardinality low to avoid Prometheus performance issues. Current design uses org_id, which should be manageable for typical SaaS applications.