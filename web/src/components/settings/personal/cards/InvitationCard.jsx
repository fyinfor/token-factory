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

import React, { useEffect, useState } from 'react';
import {
  Avatar,
  Button,
  Card,
  Input,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import { Copy, Gift, Users } from 'lucide-react';
import {
  API,
  copy,
  renderQuota,
  showError,
  showSuccess,
} from '../../../../helpers';

const InvitationCard = ({ t }) => {
  const [loading, setLoading] = useState(true);
  const [invitation, setInvitation] = useState(null);

  useEffect(() => {
    let cancelled = false;

    const loadInvitation = async () => {
      try {
        const res = await API.get('/api/user/aff');
        if (cancelled) return;
        if (res.data.success) {
          setInvitation({
            aff_code: res.data.aff_code || res.data.data,
            aff_count: res.data.aff_count || 0,
            inviter_reward_quota: res.data.inviter_reward_quota || 0,
          });
        } else {
          showError(res.data.message || t('加载失败'));
        }
      } catch (error) {
        if (!cancelled) showError(t('加载失败'));
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    loadInvitation();
    return () => {
      cancelled = true;
    };
  }, [t]);

  const affLink = invitation?.aff_code
    ? `${window.location.origin}/r/${invitation.aff_code}`
    : '';

  const handleCopy = async () => {
    if (!affLink) return;
    if (await copy(affLink)) {
      showSuccess(t('邀请链接已复制到剪贴板'));
    } else {
      showError(t('复制失败'));
    }
  };

  return (
    <Card className='!rounded-2xl'>
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='green' className='mr-3 shadow-md'>
          <Gift size={16} />
        </Avatar>
        <div>
          <Typography.Text className='text-lg font-medium'>
            {t('邀请奖励')}
          </Typography.Text>
          <div className='text-xs text-gray-600'>
            {t('邀请好友获得额外奖励')}
          </div>
        </div>
      </div>

      <Spin spinning={loading}>
        <div className='grid grid-cols-1 sm:grid-cols-2 gap-3 mb-4'>
          <div className='rounded-xl bg-slate-50 dark:bg-slate-800/60 p-4'>
            <div className='flex items-center gap-2 text-sm text-gray-500 mb-2'>
              <Users size={16} />
              {t('邀请人数')}
            </div>
            <Typography.Title heading={4} className='!mb-0'>
              {invitation?.aff_count || 0}
            </Typography.Title>
          </div>
          <div className='rounded-xl bg-slate-50 dark:bg-slate-800/60 p-4'>
            <div className='flex items-center gap-2 text-sm text-gray-500 mb-2'>
              <Gift size={16} />
              {t('邀请奖励')}
            </div>
            <Typography.Title heading={4} className='!mb-0'>
              {renderQuota(invitation?.inviter_reward_quota || 0)}
            </Typography.Title>
            {invitation?.inviter_reward_quota > 0 ? (
              <Typography.Text type='tertiary' size='small'>
                {t('每成功邀请一位好友注册，奖励直接到账')}
              </Typography.Text>
            ) : null}
          </div>
        </div>

        <Input
          value={affLink}
          readOnly
          prefix={t('邀请链接')}
          className='!rounded-lg'
          suffix={
            <Button
              type='primary'
              theme='solid'
              disabled={!affLink}
              icon={<Copy size={14} />}
              onClick={handleCopy}
              className='!rounded-lg'
            >
              {t('复制')}
            </Button>
          }
        />
      </Spin>
    </Card>
  );
};

export default InvitationCard;
