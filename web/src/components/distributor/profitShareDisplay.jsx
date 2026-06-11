/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Popover, Tooltip } from '@douyinfe/semi-ui';
import { IconHelpCircle } from '@douyinfe/semi-icons';
import {
  formatCommissionRatioPercent,
  renderQuotaFlexible,
} from '../../helpers';

const popupContainer = () =>
  typeof document !== 'undefined' ? document.body : null;

/** 利润分成额度列：常态 2 位小数，极低值自动展示至多 6 位；悬停显示完整 6 位精度。 */
export function renderProfitShareQuotaCell(quota) {
  const q = Number(quota) || 0;
  const main = renderQuotaFlexible(q, 2, 6);
  const exact = renderQuotaFlexible(q, 6, 6);
  if (main === exact) {
    return main;
  }
  return (
    <Tooltip content={exact}>
      <span className='cursor-help border-b border-dotted border-gray-400'>
        {main}
      </span>
    </Tooltip>
  );
}

export function ProfitShareRewardFormulaTooltipContent({ t }) {
  return (
    <div className='w-[460px] max-w-[min(460px,calc(100vw-32px))] text-xs leading-relaxed space-y-1.5 py-0.5'>
      <div className='font-semibold text-[var(--semi-color-text-0)]'>
        {t('利润分成计算公式')}
      </div>
      <div className='text-[var(--semi-color-text-2)]'>
        {t('利润分成计算说明-用户消耗')}
      </div>
      <div className='text-[var(--semi-color-text-2)]'>
        {t('利润分成计算说明-利润')}
      </div>
      <div className='text-[var(--semi-color-text-2)]'>
        {t('利润分成计算说明-分成比例')}
      </div>
      <div className='text-[var(--semi-color-text-2)]'>
        {t('利润分成计算说明-收益')}
      </div>
    </div>
  );
}

/** 「收益金额」列表头：标题旁问号，点击展示计算公式（挂载 body，避免被弹窗裁切）。 */
export function ProfitShareRewardColumnTitle({ t }) {
  return (
    <div className='inline-flex items-center gap-1'>
      <span>{t('收益金额')}</span>
      <Popover
        content={<ProfitShareRewardFormulaTooltipContent t={t} />}
        position='leftTop'
        trigger='click'
        showArrow
        spacing={12}
        getPopupContainer={popupContainer}
      >
        <IconHelpCircle
          className='text-gray-400 cursor-pointer flex-shrink-0 hover:text-[var(--semi-color-primary)]'
          size='small'
          aria-label={t('利润分成计算公式')}
        />
      </Popover>
    </div>
  );
}

/** 收益金额单元格：悬停展示本笔代入公式。 */
export function renderProfitShareRewardCell(row, t) {
  const reward = Number(row?.reward_quota) || 0;
  const slice = Number(row?.markup_slice_quota) || 0;
  const bps = row?.commission_bps;
  const display = renderProfitShareQuotaCell(reward);

  if (slice > 0 && typeof bps === 'number' && bps > 0) {
    const rate = formatCommissionRatioPercent(bps);
    const formula = t('利润分成单行计算', {
      slice: renderQuotaFlexible(slice, 2, 6),
      rate,
      reward: renderQuotaFlexible(reward, 2, 6),
    });
    return (
      <Tooltip
        content={formula}
        position='left'
        showArrow
        getPopupContainer={popupContainer}
      >
        {display}
      </Tooltip>
    );
  }
  return display;
}
