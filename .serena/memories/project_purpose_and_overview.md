# Payloop Project Overview

## Purpose
Payloop is a smart recurring payment processing framework designed to provide flexible and extensible subscription management capabilities. It's a comprehensive subscription billing platform built using Domain-Driven Design (DDD) principles with clean architecture patterns.

## Key Features
- **Subscription Management**: Complete lifecycle management with billing anchors, pause/resume, and recovery workflows
- **Payment Processing**: Multi-provider support (Paystack, Checkout.com) with secure vault encryption  
- **Invoice System**: Automated PDF generation with ChromeDP and Liquid templates
- **Customer Management**: Cohort-based segmentation with secure payment method storage
- **Usage-Based Billing**: Support for traditional, usage-based, and hybrid billing models
- **Dual Database Architecture**: Operational and reporting databases with CDC synchronization
- **AI Integration**: Model Context Protocol (MCP) for AI-friendly invoice operations
- **Webhook System**: Reliable outgoing webhook delivery via Temporal workflows
- **Document Storage**: AWS S3 integration for secure PDF storage and retrieval
- **Email Integration**: Transactional email notifications via Loops
- **Security Vault**: Encrypted storage for sensitive payment tokens using AES or AWS Secrets Manager

## Business Domains
- **Subscription Management**: Subscriptions, customers, payments, invoices
- **Payment Processing**: Multiple payment providers with secure vault
- **Invoice Generation**: PDF generation with templating system
- **Event System**: Topics, webhooks, queue processing
- **Usage-Based Billing**: API calls, data transfer, transaction fees, active users, storage

## Architecture Approach
- Domain-Driven Design (DDD)
- Clean Architecture
- Layered architecture with strict separation of concerns
- Dependency injection using Uber FX
- Multi-tenancy with organization-level isolation