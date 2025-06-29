-- Migration: Add required title column to Ad table
ALTER TABLE Ad ADD COLUMN title TEXT NOT NULL DEFAULT 'Untitled'; 