-- Down migration for sharing feature tables

DROP TABLE IF EXISTS public_share_accesses;
DROP TABLE IF EXISTS public_shares;
DROP TABLE IF EXISTS album_shares;
DROP TABLE IF EXISTS media_shares;
DROP TABLE IF EXISTS shares;
