-- Add deleted_at timestamp to media table for soft delete / trash functionality
ALTER TABLE media ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP NULL;
