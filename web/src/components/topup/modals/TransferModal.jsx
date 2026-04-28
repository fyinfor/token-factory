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

import React, { useMemo } from 'react';
import { Modal, Typography, Input, InputNumber } from '@douyinfe/semi-ui';
import { CreditCard } from 'lucide-react';
import {
  quotaToDisplayInputAmount,
  displayInputAmountToQuota,
} from '../../../helpers';

function FieldLabel({ required, children, className = '' }) {
  return (
    <Typography.Text
      size='small'
      strong
      className={`block mb-1.5 text-[var(--semi-color-text-0)] ${className}`.trim()}
    >
      {required ? (
        <span
          className='text-[var(--semi-color-danger)] mr-1 font-normal'
          aria-hidden
        >
          *
        </span>
      ) : null}
      {children}
    </Typography.Text>
  );
}

const TransferModal = ({
  t,
  openTransfer,
  transfer,
  handleTransferCancel,
  userState,
  renderQuota: renderQuotaProp,
  getQuotaPerUnit: getQuotaPerUnitProp,
  transferAmount,
  setTransferAmount,
}) => {
  const qpu = getQuotaPerUnitProp() || 1;
  const aff = userState?.user?.aff_quota || 0;
  const isTokens = useMemo(() => {
    if (typeof window === 'undefined') return false;
    return (localStorage.getItem('quota_display_type') || 'USD') === 'TOKENS';
  }, [openTransfer]);

  const minDisplay = quotaToDisplayInputAmount(qpu);
  const maxDisplay = aff > 0 ? quotaToDisplayInputAmount(aff) : undefined;

  const onFiatChange = (v) => {
    if (v == null || Number.isNaN(Number(v))) {
      setTransferAmount(qpu);
      return;
    }
    let quota = displayInputAmountToQuota(v);
    if (quota < qpu) quota = qpu;
    if (aff > 0 && quota > aff) quota = aff;
    setTransferAmount(quota);
  };

  return (
    <Modal
      title={
        <div className='flex items-center'>
          <CreditCard className='mr-2' size={18} />
          {t('划转邀请额度')}
        </div>
      }
      visible={openTransfer}
      onOk={transfer}
      onCancel={handleTransferCancel}
      maskClosable={false}
      centered
    >
      <div className='space-y-5'>
        <div>
          <FieldLabel>{t('可用邀请额度')}</FieldLabel>
          <Input
            value={renderQuotaProp(userState?.user?.aff_quota)}
            disabled
            className='!rounded-lg bg-[var(--semi-color-fill-0)]'
          />
        </div>
        <div>
          <div className='mb-1.5 flex flex-wrap items-baseline gap-x-1.5 gap-y-0'>
            <span
              className='text-[var(--semi-color-danger)] text-sm font-normal leading-snug'
              aria-hidden
            >
              *
            </span>
            <Typography.Text
              strong
              size='small'
              className='text-[var(--semi-color-text-0)] leading-snug'
            >
              {t('划转额度')}
            </Typography.Text>
            <Typography.Text
              type='tertiary'
              size='small'
              className='leading-snug'
            >
              {t('最低')}
              {renderQuotaProp(getQuotaPerUnitProp())}
            </Typography.Text>
          </div>
          {isTokens ? (
            <>
              <InputNumber
                min={qpu}
                max={aff > 0 ? aff : undefined}
                value={transferAmount}
                onChange={(v) => setTransferAmount(v ?? qpu)}
                className='w-full !rounded-lg'
              />
              <Typography.Text
                type='tertiary'
                size='small'
                className='block mt-2 leading-relaxed'
              >
                {t(
                  '额度以系统内部点数记账，与上方展示一致；最小划转为一档点数。',
                )}
              </Typography.Text>
            </>
          ) : (
            <>
              <InputNumber
                min={minDisplay}
                max={maxDisplay}
                value={quotaToDisplayInputAmount(transferAmount)}
                onChange={onFiatChange}
                precision={2}
                step={0.01}
                className='w-full !rounded-lg'
              />
              <Typography.Text
                type='tertiary'
                size='small'
                className='block mt-2 leading-relaxed'
              >
                {t('输入数值与钱包展示货币一致，对应')}
                {renderQuotaProp(transferAmount)}
              </Typography.Text>
            </>
          )}
        </div>
      </div>
    </Modal>
  );
};

export default TransferModal;
