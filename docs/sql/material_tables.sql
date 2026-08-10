-- 素材库本地持久化表结构（SQLite / MySQL / PostgreSQL 通用 TEXT + BIGINT 方案）
-- 生产环境推荐使用 GORM AutoMigrate 自动建表；本文件供 DBA 手工初始化或审计参考。

-- ---------------------------------------------------------------------------
-- 素材组表 material_groups
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS material_groups (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,  -- MySQL: INT AUTO_INCREMENT; PG: SERIAL
    user_id      INTEGER      NOT NULL,
    group_name   VARCHAR(255) NOT NULL,
    description  VARCHAR(512) DEFAULT '',
    group_id     VARCHAR(128) DEFAULT NULL,          -- 上游分组 ID，如 group-20260612195842-84j62
    group_type   VARCHAR(32)  DEFAULT 'virtual',     -- virtual | real
    created_at   BIGINT       DEFAULT 0,
    updated_at   BIGINT       DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_material_group_user ON material_groups (user_id);
CREATE INDEX IF NOT EXISTS idx_material_group_type ON material_groups (group_type);
CREATE INDEX IF NOT EXISTS idx_material_groups_group_id ON material_groups (group_id);

-- ---------------------------------------------------------------------------
-- 素材表 material_assets
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS material_assets (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER      NOT NULL,
    group_id    VARCHAR(128) DEFAULT NULL,
    group_type  VARCHAR(32)  DEFAULT 'virtual',
    asset_id    VARCHAR(128) DEFAULT NULL,           -- 上游素材 ID，如 asset-20260612200812-mmq8g
    name        VARCHAR(255) DEFAULT '',
    asset_type  VARCHAR(32)  DEFAULT '',             -- Image | Video | Audio
    url         TEXT         DEFAULT '',             -- 控制台预览 URL（可按 preview_expires_at 清空）
    preview_expires_at BIGINT DEFAULT 0,             -- 预览过期时间；到期只清预览不删素材
    status      VARCHAR(32)  DEFAULT '',             -- Active | Pending | Failed
    created_at  BIGINT       DEFAULT 0,
    updated_at  BIGINT       DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_material_assets_user_id ON material_assets (user_id);
CREATE INDEX IF NOT EXISTS idx_material_assets_group_id ON material_assets (group_id);
CREATE INDEX IF NOT EXISTS idx_material_assets_group_type ON material_assets (group_type);
CREATE INDEX IF NOT EXISTS idx_material_assets_asset_id ON material_assets (asset_id);
CREATE INDEX IF NOT EXISTS idx_material_assets_preview_expires_at ON material_assets (preview_expires_at);

-- ---------------------------------------------------------------------------
-- 真人认证会话表 material_visual_sessions
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS material_visual_sessions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER      NOT NULL,
    byted_token   VARCHAR(255) DEFAULT NULL,         -- 仅后端轮询，禁止对外日志/响应泄露
    h5_link       TEXT         DEFAULT '',
    qr_code       TEXT         DEFAULT '',
    status        VARCHAR(32)  DEFAULT 'pending',    -- pending | success | failed | expired
    group_id      VARCHAR(128) DEFAULT NULL,
    error_message TEXT         DEFAULT '',
    expires_at    BIGINT       DEFAULT 0,
    created_at    BIGINT       DEFAULT 0,
    updated_at    BIGINT       DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_material_visual_sessions_user_id ON material_visual_sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_material_visual_sessions_byted_token ON material_visual_sessions (byted_token);
CREATE INDEX IF NOT EXISTS idx_material_visual_sessions_status ON material_visual_sessions (status);
CREATE INDEX IF NOT EXISTS idx_material_visual_sessions_group_id ON material_visual_sessions (group_id);
