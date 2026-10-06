-- +goose Up
-- Community (slice 4): staff forum (POS COPILOT platform workstream).
-- See docs/23_COMMUNITY_JOBS_PROFILES.md. Categories hold posts; posts hold
-- comments and per-user likes (PK (post_id, user_id), so a like is a row and
-- "did I like this" is a lookup). All FORCE RLS + tenant policy like every
-- other tenant table. NOTE: excluded from the sync change feed (awareness
-- data, not register state) and from the platform backup catalog (matching
-- the other community tables).

CREATE TABLE forum_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name),
    CONSTRAINT forum_categories_name_required CHECK (length(btrim(name)) > 0)
);

CREATE TABLE forum_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES forum_categories(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_closed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, id),
    CONSTRAINT forum_posts_title_required CHECK (length(btrim(title)) > 0)
);

CREATE TABLE forum_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    post_id UUID NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT forum_comments_body_required CHECK (length(btrim(body)) > 0)
);

CREATE TABLE forum_post_likes (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    post_id UUID NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

ALTER TABLE forum_categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE forum_categories FORCE ROW LEVEL SECURITY;
CREATE POLICY forum_categories_tenant_policy ON forum_categories
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE forum_posts ENABLE ROW LEVEL SECURITY;
ALTER TABLE forum_posts FORCE ROW LEVEL SECURITY;
CREATE POLICY forum_posts_tenant_policy ON forum_posts
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE forum_comments ENABLE ROW LEVEL SECURITY;
ALTER TABLE forum_comments FORCE ROW LEVEL SECURITY;
CREATE POLICY forum_comments_tenant_policy ON forum_comments
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE forum_post_likes ENABLE ROW LEVEL SECURITY;
ALTER TABLE forum_post_likes FORCE ROW LEVEL SECURITY;
CREATE POLICY forum_post_likes_tenant_policy ON forum_post_likes
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

CREATE INDEX forum_categories_tenant_idx ON forum_categories (tenant_id, is_active, sort_order);
CREATE INDEX forum_posts_category_idx ON forum_posts (tenant_id, category_id, created_at DESC);
CREATE INDEX forum_posts_list_idx ON forum_posts (tenant_id, created_at DESC);
CREATE INDEX forum_comments_post_idx ON forum_comments (tenant_id, post_id, created_at);
CREATE INDEX forum_post_likes_post_idx ON forum_post_likes (tenant_id, post_id);

-- +goose Down
DROP POLICY IF EXISTS forum_post_likes_tenant_policy ON forum_post_likes;
DROP TABLE IF EXISTS forum_post_likes;
DROP POLICY IF EXISTS forum_comments_tenant_policy ON forum_comments;
DROP TABLE IF EXISTS forum_comments;
DROP POLICY IF EXISTS forum_posts_tenant_policy ON forum_posts;
DROP TABLE IF EXISTS forum_posts;
DROP POLICY IF EXISTS forum_categories_tenant_policy ON forum_categories;
DROP TABLE IF EXISTS forum_categories;
