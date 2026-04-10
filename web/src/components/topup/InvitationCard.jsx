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

import React, { useState, useCallback, useEffect } from 'react';
import {
  Avatar,
  Typography,
  Card,
  Button,
  Input,
  Badge,
  Space,
  Modal,
  Table,
  InputNumber,
} from '@douyinfe/semi-ui';
import { Copy, Users, BarChart2, TrendingUp, Gift, Zap } from 'lucide-react';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const InvitationCard = ({
  t,
  userState,
  renderQuota,
  setOpenTransfer,
  affLink,
  handleAffLinkClick,
}) => {
  const [inviteModalOpen, setInviteModalOpen] = useState(false);
  const [inviteLoading, setInviteLoading] = useState(false);
  const [inviteRows, setInviteRows] = useState([]);
  const [inviteTotal, setInviteTotal] = useState(0);
  const [invitePage, setInvitePage] = useState(1);
  const [invitePageSize, setInvitePageSize] = useState(10);
  const [defaultCommissionBps, setDefaultCommissionBps] = useState(0);
  const [savingId, setSavingId] = useState(null);

  const loadInvitees = useCallback(
    async (p = 1, ps = invitePageSize) => {
      setInviteLoading(true);
      try {
        const res = await API.get(
          `/api/user/aff_invitees?p=${p}&page_size=${ps}`,
        );
        const { success, message, data } = res.data;
        if (!success) {
          showError(message || t('加载失败'));
          return;
        }
        const items = data?.items || [];
        setInviteRows(
          items.map((row) => ({
            ...row,
            _bps: row.commission_ratio_bps,
          })),
        );
        setInviteTotal(data?.total ?? 0);
        setDefaultCommissionBps(data?.default_commission_ratio_bps ?? 0);
        setInvitePage(p);
      } catch {
        showError(t('加载失败'));
      } finally {
        setInviteLoading(false);
      }
    },
    [invitePageSize, t],
  );

  // 仅在打开弹窗时拉第一页；翻页、改 pageSize 由 Table pagination 回调触发，避免重复请求
  useEffect(() => {
    if (inviteModalOpen) {
      loadInvitees(1, invitePageSize);
    }
  }, [inviteModalOpen]);

  const updateRowBps = (inviteeId, value) => {
    setInviteRows((prev) =>
      prev.map((r) =>
        r.invitee_id === inviteeId ? { ...r, _bps: value } : r,
      ),
    );
  };

  const saveCommission = async (record) => {
    const bps = Math.round(Number(record._bps));
    if (Number.isNaN(bps) || bps < 0 || bps > 10000) {
      showError(t('分销比例范围错误'));
      return;
    }
    setSavingId(record.invitee_id);
    try {
      const res = await API.put('/api/user/aff_invitees/commission', {
        invitee_id: record.invitee_id,
        commission_ratio_bps: bps,
      });
      const { success, message } = res.data;
      if (!success) {
        showError(message || t('保存失败'));
        return;
      }
      showSuccess(t('保存成功'));
      setInviteRows((prev) =>
        prev.map((r) =>
          r.invitee_id === record.invitee_id
            ? { ...r, commission_ratio_bps: bps, _bps: bps }
            : r,
        ),
      );
    } catch {
      showError(t('保存失败'));
    } finally {
      setSavingId(null);
    }
  };

  const inviteColumns = [
    {
      title: t('被邀请用户ID'),
      dataIndex: 'invitee_id',
      width: 110,
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
    },
    {
      title: t('显示名称'),
      dataIndex: 'display_name',
      render: (v) => (v && String(v).trim() ? v : '—'),
    },
    {
      title: t('分销比例'),
      dataIndex: '_bps',
      width: 220,
      render: (_, record) => (
        <div className='flex items-center gap-2 flex-wrap'>
          <InputNumber
            min={0}
            max={10000}
            value={record._bps ?? 0}
            onChange={(v) => updateRowBps(record.invitee_id, v)}
            className='!w-28'
          />
          <Button
            size='small'
            type='primary'
            loading={savingId === record.invitee_id}
            onClick={() => saveCommission(record)}
          >
            {t('保存')}
          </Button>
        </div>
      ),
    },
  ];

  return (
    <Card className='!rounded-2xl shadow-sm border-0'>
      {/* 卡片头部 */}
      <div className='flex items-center mb-4'>
        <Avatar size='small' color='green' className='mr-3 shadow-md'>
          <Gift size={16} />
        </Avatar>
        <div>
          <Typography.Text className='text-lg font-medium'>
            {t('邀请奖励')}
          </Typography.Text>
          <div className='text-xs'>{t('邀请好友获得额外奖励')}</div>
        </div>
      </div>

      {/* 收益展示区域 */}
      <Space vertical style={{ width: '100%' }}>
        {/* 统计数据统一卡片 */}
        <Card
          className='!rounded-xl w-full'
          cover={
            <div
              className='relative h-30'
              style={{
                '--palette-primary-darkerChannel': '0 75 80',
                backgroundImage: `linear-gradient(0deg, rgba(var(--palette-primary-darkerChannel) / 80%), rgba(var(--palette-primary-darkerChannel) / 80%)), url('/cover-4.webp')`,
                backgroundSize: 'cover',
                backgroundPosition: 'center',
                backgroundRepeat: 'no-repeat',
              }}
            >
              {/* 标题和按钮 */}
              <div className='relative z-10 h-full flex flex-col justify-between p-4'>
                <div className='flex justify-between items-center'>
                  <Text strong style={{ color: 'white', fontSize: '16px' }}>
                    {t('收益统计')}
                  </Text>
                  <Button
                    type='primary'
                    theme='solid'
                    size='small'
                    disabled={
                      !userState?.user?.aff_quota ||
                      userState?.user?.aff_quota <= 0
                    }
                    onClick={() => setOpenTransfer(true)}
                    className='!rounded-lg'
                  >
                    <Zap size={12} className='mr-1' />
                    {t('划转到余额')}
                  </Button>
                </div>

                {/* 统计数据 */}
                <div className='grid grid-cols-3 gap-6 mt-4'>
                  {/* 待使用收益 */}
                  <div className='text-center'>
                    <div
                      className='text-base sm:text-2xl font-bold mb-2'
                      style={{ color: 'white' }}
                    >
                      {renderQuota(userState?.user?.aff_quota || 0)}
                    </div>
                    <div className='flex items-center justify-center text-sm'>
                      <TrendingUp
                        size={14}
                        className='mr-1'
                        style={{ color: 'rgba(255,255,255,0.8)' }}
                      />
                      <Text
                        style={{
                          color: 'rgba(255,255,255,0.8)',
                          fontSize: '12px',
                        }}
                      >
                        {t('待使用收益')}
                      </Text>
                    </div>
                  </div>

                  {/* 总收益 */}
                  <div className='text-center'>
                    <div
                      className='text-base sm:text-2xl font-bold mb-2'
                      style={{ color: 'white' }}
                    >
                      {renderQuota(userState?.user?.aff_history_quota || 0)}
                    </div>
                    <div className='flex items-center justify-center text-sm'>
                      <BarChart2
                        size={14}
                        className='mr-1'
                        style={{ color: 'rgba(255,255,255,0.8)' }}
                      />
                      <Text
                        style={{
                          color: 'rgba(255,255,255,0.8)',
                          fontSize: '12px',
                        }}
                      >
                        {t('总收益')}
                      </Text>
                    </div>
                  </div>

                  {/* 邀请人数 */}
                  <div
                    className='text-center cursor-pointer rounded-lg outline-none transition-opacity hover:opacity-95 focus-visible:ring-2 focus-visible:ring-white/70 focus-visible:ring-offset-2 focus-visible:ring-offset-transparent'
                    role='button'
                    tabIndex={0}
                    aria-label={t('邀请人数')}
                    onClick={() => setInviteModalOpen(true)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault();
                        setInviteModalOpen(true);
                      }
                    }}
                  >
                    <div
                      className='text-base sm:text-2xl font-bold mb-2'
                      style={{ color: 'white' }}
                    >
                      {userState?.user?.aff_count || 0}
                    </div>
                    <div className='flex items-center justify-center text-sm'>
                      <Users
                        size={14}
                        className='mr-1'
                        style={{ color: 'rgba(255,255,255,0.8)' }}
                      />
                      <Text
                        style={{
                          color: 'rgba(255,255,255,0.8)',
                          fontSize: '12px',
                        }}
                      >
                        {t('邀请人数')}
                      </Text>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          }
        >
          {/* 邀请链接部分 */}
          <Input
            value={affLink}
            readonly
            className='!rounded-lg'
            prefix={t('邀请链接')}
            suffix={
              <Button
                type='primary'
                theme='solid'
                onClick={handleAffLinkClick}
                icon={<Copy size={14} />}
                className='!rounded-lg'
              >
                {t('复制')}
              </Button>
            }
          />
        </Card>

        {/* 奖励说明 */}
        <Card
          className='!rounded-xl w-full'
          title={<Text type='tertiary'>{t('奖励说明')}</Text>}
        >
          <div className='space-y-3'>
            <div className='flex items-start gap-2'>
              <Badge dot type='success' />
              <Text type='tertiary' className='text-sm'>
                {t('邀请好友注册，好友充值后您可获得相应奖励')}
              </Text>
            </div>

            <div className='flex items-start gap-2'>
              <Badge dot type='success' />
              <Text type='tertiary' className='text-sm'>
                {t('通过划转功能将奖励额度转入到您的账户余额中')}
              </Text>
            </div>

            <div className='flex items-start gap-2'>
              <Badge dot type='success' />
              <Text type='tertiary' className='text-sm'>
                {t('邀请的好友越多，获得的奖励越多')}
              </Text>
            </div>
          </div>
        </Card>
      </Space>

      <Modal
        title={
          <div className='flex items-center gap-2'>
            <Users size={18} />
            {t('邀请人列表')}
          </div>
        }
        visible={inviteModalOpen}
        onCancel={() => setInviteModalOpen(false)}
        footer={null}
        width={720}
        centered
        maskClosable
      >
        <Text type='tertiary' className='text-xs block mb-3'>
          {t('分销比例邀请说明行', { bps: defaultCommissionBps })}
        </Text>
        <Table
          rowKey='invitee_id'
          columns={inviteColumns}
          dataSource={inviteRows}
          loading={inviteLoading}
          pagination={{
            currentPage: invitePage,
            pageSize: invitePageSize,
            total: inviteTotal,
            showSizeChanger: true,
            pageSizeOpts: [10, 20, 50],
            onPageChange: (p) => loadInvitees(p, invitePageSize),
            onPageSizeChange: (ps) => {
              setInvitePageSize(ps);
              setInvitePage(1);
              loadInvitees(1, ps);
            },
          }}
        />
      </Modal>
    </Card>
  );
};

export default InvitationCard;
