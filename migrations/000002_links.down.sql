-- Phase 2: Rollback links table
-- Migration: 000002_links.down.sql

DROP TRIGGER IF EXISTS trigger_links_updated_at ON links;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS links;
