-- Migration: Add Unique Constraint to Albums Table
-- Description: Ensures that an album name is unique per user, preventing duplicate names within a single user's collection.

-- 1. Cleanup existing duplicates (if any) before applying the constraint.
-- In a production environment, this would be handled carefully or via a manual data cleanup process.
-- For the purpose of this migration script being idempotent and safe:
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM (
            SELECT id, ROW_NUMBER() OVER(PARTITION BY user_id, name ORDER BY created_at DESC) as rn
            FROM albums
        ) t WHERE rn > 1
    ) THEN
        RAISE NOTICE 'Duplicates found in albums table. Cleaning up duplicates...';
        DELETE FROM albums 
        WHERE id IN (
            SELECT id FROM (
                SELECT id, ROW_NUMBER() OVER(PARTITION BY user_id, name ORDER BY created_at DESC) as rn
                FROM albums
            ) t WHERE rn > 1
        );
    ELSE
        RAISE NOTICE 'No duplicates found in albums table.';
    END IF;
END $$;

-- 2. Apply the unique constraint
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'unique_user_album_name'
    ) THEN
        ALTER TABLE albums ADD CONSTRAINT unique_user_album_name UNIQUE (user_id, name);
        RAISE NOTICE 'Constraint "unique_user_album_name" added successfully.';
    ELSE
        RAISE NOTICE 'Constraint "unique_user_album_name" already exists. Skipping.';
    END IF;
END $$;
