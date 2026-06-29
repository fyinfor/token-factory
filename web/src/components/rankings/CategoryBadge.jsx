/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import { Tag } from '@douyinfe/semi-ui';
import { getRankingCategoryLabel, getRankingCategoryStyle } from '../../helpers/rankings';

// 在排行榜三块区域中按 category 显示分类徽章。
// 与 system 其他位置（playground / 供应商能力）一致：分类常量取值 text / image / video。
// 'all' 不会到达此处（'all' 仅表示「全部」Tab 的筛选值，不作为模型的分类）。
// 在「全部」Tab 下，按各模型自身分类显示具体徽章；在子分类 Tab 下，徽章颜色与当前 Tab 一致。
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
