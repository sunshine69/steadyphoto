-- Migration to add sharing feature tables

-- shares: Groups user-to-user shares together
CREATE TABLE IF NOT EXISTS shares (
    id UUID PRIMARY KEY,
    sharer_user_id UUID NOT NULL REFERENCES users(id),
    sharee_user_id UUID NOT NULL REFERENCES users(id),
    shared_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- media_shares: Individual media items shared with a user
CREATE TABLE IF NOT EXISTS media_shares (
    id UUID PRIMARY KEY,
    share_id UUID NOT NULL REFERENCES shares(id) ON DELETE CASCADE,
    media_id UUID NOT NULL REFERENCES media(id),
    UNIQUE(share_id, media_id)
);

-- album_shares: Albums shared with a user
CREATE TABLE IF NOT EXISTS album_shares (
    id UUID PRIMARY KEY,
    share_id UUID NOT NULL REFERENCES shares(id) ON DELETE CASCADE,
    album_id UUID NOT NULL REFERENCES albums(id),
    UNIQUE(share_id, album_id)
);

-- public_shares: Public share links for media or albums
CREATE TABLE IF NOT EXISTS public_shares (
    id UUID PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    sharer_user_id UUID NOT NULL REFERENCES users(id),
    resource_type VARCHAR(10) NOT NULL CHECK (resource_type IN ('media', 'album')),
    resource_id UUID NOT NULL,
    password_hash TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    access_count INT NOT NULL DEFAULT 0
);

-- public_share_accesses: Access log for security auditing
CREATE TABLE IF NOT EXISTS public_share_accesses (
    id UUID PRIMARY KEY,
    public_share_id UUID NOT NULL REFERENCES public_shares(id),
    ip_address INET NOT NULL,
    accessed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_shares_sharee ON shares(sharee_user_id);
CREATE INDEX IF NOT EXISTS idx_shares_sharer ON shares(sharer_user_id);
CREATE INDEX IF NOT EXISTS idx_media_shares_media ON media_shares(media_id);
CREATE INDEX IF NOT EXISTS idx_album_shares_album ON album_shares(album_id);
CREATE INDEX IF NOT EXISTS idx_public_shares_token ON public_shares(token);
CREATE INDEX IF NOT EXISTS idx_public_shares_resource ON public_shares(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_public_share_accesses_share ON public_share_accesses(public_share_id);
