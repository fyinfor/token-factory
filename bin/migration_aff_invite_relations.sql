-- 邀请分销：为每个被邀请人单独配置充值奖励比例（与 GORM 模型 aff_invite_relations 一致）
-- 若已使用应用启动时的 AutoMigrate，可跳过手工执行；本脚本供 DBA 审阅或离线部署使用。

-- ========== MySQL / MariaDB ==========
CREATE TABLE IF NOT EXISTS `aff_invite_relations` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `inviter_id` bigint NOT NULL,
  `invitee_user_id` bigint NOT NULL,
  `commission_ratio_bps` bigint NOT NULL DEFAULT 0 COMMENT '万分比，100=1%%，10000=100%%',
  `created_at` bigint DEFAULT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_aff_inv_pair` (`inviter_id`,`invitee_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ========== PostgreSQL ==========
-- CREATE TABLE IF NOT EXISTS aff_invite_relations (
--   id BIGSERIAL PRIMARY KEY,
--   inviter_id BIGINT NOT NULL,
--   invitee_user_id BIGINT NOT NULL,
--   commission_ratio_bps BIGINT NOT NULL DEFAULT 0,
--   created_at BIGINT,
--   updated_at BIGINT,
--   CONSTRAINT idx_aff_inv_pair UNIQUE (inviter_id, invitee_user_id)
-- );

-- ========== SQLite ==========
-- CREATE TABLE IF NOT EXISTS `aff_invite_relations` (
--   `id` integer PRIMARY KEY AUTOINCREMENT,
--   `inviter_id` integer NOT NULL,
--   `invitee_user_id` integer NOT NULL,
--   `commission_ratio_bps` integer NOT NULL DEFAULT 0,
--   `created_at` integer,
--   `updated_at` integer
-- );
-- CREATE UNIQUE INDEX IF NOT EXISTS `idx_aff_inv_pair` ON `aff_invite_relations` (`inviter_id`, `invitee_user_id`);

-- 可选：系统默认分销比例（与 options 表 key 一致，亦可仅在后台「运营设置」中配置）
-- INSERT INTO options (`key`, `value`) VALUES ('AffiliateDefaultCommissionBps', '0')
--   ON DUPLICATE KEY UPDATE `value` = `value`;  -- MySQL
-- PostgreSQL: INSERT ... ON CONFLICT DO NOTHING;
