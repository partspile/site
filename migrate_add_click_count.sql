-- Migration: Add required title column to Ad table
ALTER TABLE Ad ADD COLUMN title TEXT NOT NULL DEFAULT 'Untitled';

-- Add title column to ArchivedAd
ALTER TABLE ArchivedAd ADD COLUMN title TEXT;

-- Backfill title for already archived ads if possible (if Ad still exists)
-- (This will only work for ads that are still in Ad, so may not backfill all)
UPDATE ArchivedAd SET title = (SELECT title FROM Ad WHERE Ad.id = ArchivedAd.id) WHERE title IS NULL; 