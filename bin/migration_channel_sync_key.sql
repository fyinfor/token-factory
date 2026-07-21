-- channels.sync_key: 跨环境导入/导出同步唯一编号（默认约 6 位字母数字）
-- 一般由服务启动时 AutoMigrate + BackfillChannelSyncKeys 完成；本脚本供手工迁移参考。

ALTER TABLE channels ADD COLUMN IF NOT EXISTS sync_key VARCHAR(64) NOT NULL DEFAULT '';

-- 为历史空值补全（应用层 BackfillChannelSyncKeys 会生成约 6 位短码）
-- UPDATE ... 请走应用 backfill，避免手工碰撞。

CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_sync_key_unique ON channels (sync_key);
