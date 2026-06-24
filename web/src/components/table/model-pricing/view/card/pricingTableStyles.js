/**
 * 计费表格共享样式常量与工具函数
 *
 * 所有计费表格（按量计费、阶梯计费、视频计费）统一视觉风格、表格结构、
 * 间距、边框、配色，全局样式复用，不出现差异化。
 */

/* ============================================================
 * 通用表格样式常量
 * ============================================================ */

/** 表格外层容器样式 */
export const PRICING_TABLE_WRAPPER_STYLE = {
  borderColor: 'var(--semi-color-border)',
  backgroundColor: 'var(--semi-color-fill-0)',
};

/** 表头行背景色 */
export const PRICING_TABLE_HEAD_BG = 'var(--semi-color-fill-0)';

/** 表头单元格样式 */
export const PRICING_TABLE_HEAD_CELL_STYLE = {
  borderBottom: '1px solid var(--semi-color-border)',
  color: 'var(--semi-color-text-2)',
};

/** 数据行边框 */
export const PRICING_TABLE_ROW_BORDER = '1px solid var(--semi-color-border)';

/** 数据行背景色 */
export const PRICING_TABLE_BODY_BG = 'var(--semi-color-bg-2)';

/* ============================================================
 * 折扣视觉样式
 * 折扣单元格：字体加粗、字号放大、高饱和度鲜艳色
 * ============================================================ */

/** 折扣高亮样式（有折扣时） */
export const DISCOUNT_HIGHLIGHT_STYLE = {
  fontWeight: 700,
  fontSize: 11,
  color: '#E74C3C',
};

/** 折扣低对比度样式（无折扣时） */
export const DISCOUNT_MUTED_STYLE = {
  fontWeight: 400,
  fontSize: 11,
  color: 'var(--semi-color-text-3)',
};

/** 折扣 Tag 内联样式（高饱和度鲜艳色、加粗） */
export const DISCOUNT_TAG_STYLE = {
  fontWeight: 700,
  fontSize: 11,
  color: '#E74C3C',
};

/* ============================================================
 * 边界隐藏逻辑
 *
 * 判断条件：平台价 > 官方价 并且 折扣数值 <= 0
 * 满足条件时：隐藏官方价、折扣两整列，表格仅保留价格项、平台价
 * 不满足条件：正常展示全部四列
 * ============================================================ */

/**
 * 判断某一行是否应该隐藏官方价和折扣列
 * @param {number} platformPriceUsd - 平台价 USD 值
 * @param {number} officialPriceUsd - 官方价 USD 值
 * @param {number|null} discount - 折扣百分比（如 30 表示 30% off）
 * @returns {boolean} true 表示应隐藏官方价和折扣列
 */
export const shouldHideOfficialAndDiscount = (
  platformPriceUsd,
  officialPriceUsd,
  discount,
) => {
  const platform = Number(platformPriceUsd) || 0;
  const official = Number(officialPriceUsd) || 0;
  // 边界条件：平台价 > 官方价 且 折扣 ≤ 0（含 null/undefined/0）
  return platform > official && (discount == null || discount <= 0);
};

/**
 * 对一组数据行批量判断是否应隐藏官方价/折扣列
 * 若任意一行不应隐藏，则全局不隐藏（保持四列一致性）
 * @param {Array<{platformPriceUsd: number, officialPriceUsd: number, discount: number|null}>} rows
 * @returns {boolean} true 表示全局隐藏官方价和折扣列
 */
export const shouldHideOfficialColumnsForRows = (rows) => {
  if (!Array.isArray(rows) || rows.length === 0) return false;
  return rows.every(
    (row) =>
      shouldHideOfficialAndDiscount(
        row.platformPriceUsd,
        row.officialPriceUsd,
        row.discount,
      ),
  );
};

/* ============================================================
 * 表头列定义（i18n）
 * ============================================================ */

/** 按量计费表头列（完整四列） */
export const getFlatPricingColumns = (t) => ({
  label: t('价格项'),
  platform: t('平台价 / M'),
  official: t('官方价 / M'),
  discount: t('折扣'),
});

/** 按次计费表头列（完整四列） */
export const getFixedPricingColumns = (t) => ({
  label: t('价格项'),
  platform: t('平台价 / 次'),
  official: t('官方价 / 次'),
  discount: t('折扣'),
});

/** 视频计费表头列（完整四列） */
export const getVideoPricingColumns = (t) => ({
  label: t('计费类型'),
  platform: t('平台价'),
  official: t('官方价'),
  discount: t('折扣'),
});

/** 阶梯计费表头列（完整四列，首列通常为 token 区间） */
export const getTierPricingColumns = (t) => ({
  label: t('区间'),
  platform: t('平台价 / M'),
  official: t('官方价 / M'),
  discount: t('折扣'),
});

/* ============================================================
 * 通用表格单元格 class
 * ============================================================ */

export const TABLE_CELL_CLASS = {
  thLeft: 'px-2 py-1 text-left font-semibold',
  thCenter: 'px-2 py-1 text-center font-medium',
  tdLabel: 'px-2 py-1.5 font-semibold',
  tdPlatform: 'px-2 py-1.5 text-center font-bold',
  tdOfficial: 'px-2 py-1.5 text-center font-medium',
  tdDiscount: 'px-2 py-1.5 text-center',
};
