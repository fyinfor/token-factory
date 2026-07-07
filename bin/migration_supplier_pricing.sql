-- 供应商分层定价表（由 GORM AutoMigrate 同步；字段注释见 model/supplier_model_pricing_tables.go 中 gorm comment）
-- 以下为运维审计 / 手工补表注释参考（MySQL 8+）。PostgreSQL 请改用 COMMENT ON TABLE/COLUMN。

-- =============================================================================
-- 表说明
-- =============================================================================
-- supplier_model_pricings：供应商全局模型定价，独立于管理员 Option；
--   唯一约束 (supplier_application_id, model_name)，计费优先级低于渠道表、高于平台全局。
-- supplier_channel_model_pricings：供应商渠道模型定价；
--   唯一约束 (supplier_application_id, channel_id, model_name)，计费优先级最高。

-- =============================================================================
-- MySQL：表级 COMMENT（已建表后可执行）
-- =============================================================================
-- ALTER TABLE supplier_model_pricings COMMENT='供应商全局模型定价（按供应商申请+模型名唯一，与平台 Option 分离）';
-- ALTER TABLE supplier_channel_model_pricings COMMENT='供应商渠道模型定价（按供应商+渠道+模型名唯一，优先级高于全局表与平台 Option）';
