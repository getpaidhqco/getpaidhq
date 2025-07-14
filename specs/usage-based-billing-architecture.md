# Usage-Based Billing Architecture Implementation Specification

## Overview

This specification outlines the implementation of an event-driven PostgreSQL architecture for usage-based billing that separates high-volume usage recording from core business operations while maintaining data consistency through event sourcing and time-series optimized PostgreSQL features.

## Architecture Goals

1. **Future-Proof Scalability**: Built to handle growth from MVP (few transactions/day) to high volume
2. **Logical Separation**: Isolate usage data from core business operations
3. **Billing Accuracy**: Ensure eventual consistency with 30-minute settlement window
4. **Real-time Analytics**: Support near real-time usage dashboards
5. **Scalability**: Independent scaling of usage recording and billing systems

## System Architecture

### Core Components

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   API       │────▶│   Event     │────▶│  Usage DB    │────▶│   Main DB   │
│   (Usage)   │     │  Publisher  │     │(PostgreSQL)  │     │  (Billing)  │
└─────────────┘     └─────────────┘     └──────────────┘     └─────────────┘
                                              │                     ▲
                                              └─────────────────────┘
                                             2:30AM Billing Process
```

### Data Flow

1. **Usage Recording**: API → Event Publisher → PostgreSQL Usage DB (Real-time)
2. **Analytics**: PostgreSQL Materialized Views (5-minute refresh)
3. **Billing**: Scheduled job at 2:30 AM queries finalized aggregates
4. **Customer Dashboards**: Query materialized views for near real-time data

## Database Architecture

### PostgreSQL Usage Database

**Purpose**: High-volume usage event storage and time-series analytics with partitioning
**Port**: 5433 (or separate RDS instance)
**Database**: `payloop_usage`

### PostgreSQL Main Database

**Purpose**: Core business logic, subscriptions, invoices, customers
**Port**: 5432  
**Database**: `payloop`

### Event Publisher

**Purpose**: Event streaming and decoupling
**Interface**: `DurableEventPublisher`
**Implementation**: Can be Kafka, NATS, or any other event streaming system
**Topics**: `usage-events`, `usage-processed`


## Design decisions
- Usage events are immutable and stored in a time-series optimized manner
- Time-based isolation - billing queries use explicit time periods
- Customer-level locking - prevents concurrent billing operations
- Event Deduplication happens at Ingestion (possibly Uses Redis or memory-based dedup with TTL)
- Time-Period Boundaries
   - Each subscription item queries events for its specific billing period
   - Same events can be queried by multiple subscriptions (intentional)