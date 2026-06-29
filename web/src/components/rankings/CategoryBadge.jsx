/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import { Tag } from '@douyinfe/semi-ui';
import { getRankingCategoryLabel, getRankingCategoryStyle } from '../../helpers/rankings';

// 在排行榜三块区域中按 category 显示分类徽章。
// 当 category 为 'all'（即全局 Tab）下，会按各模型自身分类显示具体徽章；
// 选中具体分类 Tab 时，徽章仍展示以加强视觉一致性，但颜色和当前 Tab 一致。
const CategoryBadge = memo(({ category, size = 'small' }) => {
  if (!category || category === 'all') return null;
  const label = getRankingCategoryLabel(category, null);
  const style = getRankingCategoryStyle(category);
  return (
    <Tag
      size={size}
      style={{
        color: style.color,
        backgroundColor: style.background,
        borderColor: 'transparent',
        marginLeft: 4,
        flexShrink: 0,
      }}
    >
      {label}
    </Tag>
  );
});

CategoryBadge.displayName = 'CategoryBadge';

export default CategoryBadge;
