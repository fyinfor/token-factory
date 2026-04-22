-- 渠道表：价格折扣（百分数，100=无折扣，60=按原价×0.6 计费）
-- GORM AutoMigrate 会自动添加；以下为手动执行时的三库参考语句。

-- MySQL
-- ALTER TABLE `channels` ADD COLUMN `price_discount_percent` DOUBLE NULL DEFAULT 100 COMMENT '渠道计费折扣百分数' AFTER `supplier_name`;
-- UPDATE `channels` SET `price_discount_percent` = 100 WHERE `price_discount_percent` IS NULL;

-- PostgreSQL
-- ALTER TABLE "channels" ADD COLUMN IF NOT EXISTS "price_discount_percent" DOUBLE PRECISION DEFAULT 100;
-- UPDATE "channels" SET "price_discount_percent" = 100 WHERE "price_discount_percent" IS NULL;

-- SQLite
-- 若使用 GORM 自动迁移，通常无需手工执行。参考：
-- ALTER TABLE `channels` ADD COLUMN `price_discount_percent` REAL NOT NULL DEFAULT 100;
