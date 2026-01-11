-- Phase 5: Webhook system rollback
-- Migration: 000006_webhooks.down.sql

DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS webhook_endpoints;
