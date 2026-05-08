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
import { Button, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { timestamp2string, renderQuota, renderQuotaWithPrompt } from '../../../helpers';

const { Text } = Typography;

/**
 * 下载远程文件：通过 fetch 转 blob 后触发浏览器下载。
 * @param {string} url 文件 URL
 * @param {string} fallbackName 下载兜底文件名
 */
const downloadFileFromUrl = async (url, fallbackName) => {
  if (!url) {
    return;
  }
  try {
    const resp = await fetch(url);
    if (!resp.ok) {
      window.open(url, '_blank', 'noopener,noreferrer');
      return;
    }
    const blob = await resp.blob();
    const objectUrl = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = objectUrl;
    link.download = fallbackName || 'logo';
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(objectUrl);
  } catch (e) {
    window.open(url, '_blank', 'noopener,noreferrer');
  }
};

export const getSuppliersColumns = (
  t,
  openEdit,
  handleDeactivate,
  handleActivate,
  openDashboard,
) => {
  return [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 100,
      render: (status) => {
        if (status === 1) {
          return <Tag color='green'>{t('已启用')}</Tag>;
        } else if (status === 3) {
          return <Tag color='red'>{t('已注销')}</Tag>;
        }
        return <Tag>{t('未知')}</Tag>;
      },
    },
    {
      title: t('用户名'),
      dataIndex: 'applicant_username',
      width: 100,
    },
    // 对接人账户剩余额度（内部点数 + 与使用日志一致的花费换算）
    {
      title: t('剩余额度'),
      dataIndex: 'applicant_quota',
      width: 140,
      render: (v) => (
        <div>
          <div>{v ?? 0}</div>
          <Text type='tertiary' size='small'>
            {renderQuota(v ?? 0)}
          </Text>
        </div>
      ),
    },
    // 对接人历史累计已用额度（内部点数 + 等价花费提示）
    {
      title: t('历史消耗'),
      dataIndex: 'applicant_used_quota',
      width: 160,
      render: (v) => (
        <div>
          <div>{v ?? 0}</div>
          <Text type='tertiary' size='small'>
            {renderQuotaWithPrompt(v ?? 0)}
          </Text>
        </div>
      ),
    },
    {
      title: t('企业/主体名称'),
      dataIndex: 'company_name',
      width: 200,
    },
    {
      title: t('企业Logo'),
      dataIndex: 'company_logo_url',
      width: 220,
      render: (text, record) =>
        text ? (
          <Space>
            <img
              src={text}
              alt={t('企业Logo')}
              style={{
                width: 32,
                height: 32,
                borderRadius: 6,
                objectFit: 'cover',
              }}
            />
            <Button
              theme='light'
              type='primary'
              size='small'
              onClick={() =>
                downloadFileFromUrl(
                  text,
                  `${record?.supplier_alias || record?.company_name || 'company'}-logo`,
                )
              }
            >
              {t('下载')}
            </Button>
          </Space>
        ) : (
          '-'
        ),
    },
    {
      title: t('供应商别名'),
      dataIndex: 'supplier_alias',
      width: 160,
      render: (text) => text || '-',
    },
    {
      title: t('统一社会信用代码'),
      dataIndex: 'credit_code',
      width: 180,
    },
    {
      title: t('法人/经营者姓名'),
      dataIndex: 'legal_representative',
      width: 150,
    },
    {
      title: t('企业规模'),
      dataIndex: 'company_size',
      width: 120,
    },
    {
      title: t('对接人姓名'),
      dataIndex: 'contact_name',
      width: 120,
    },
    {
      title: t('对接人手机号'),
      dataIndex: 'contact_mobile',
      width: 130,
    },
    {
      title: t('对接人微信/企业微信'),
      dataIndex: 'contact_wechat',
      width: 150,
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_at',
      width: 160,
      render: (text) => timestamp2string(text),
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      width: 260,
      fixed: 'right',
      render: (text, record) => {
        const isDeactivated = record.status === 3;
        return (
          <Space>
            <Button
              theme='light'
              type='primary'
              size='small'
              disabled={isDeactivated}
              onClick={() => openEdit(record)}
            >
              {t('修改')}
            </Button>
            <Button
              theme='light'
              type={isDeactivated ? 'primary' : 'danger'}
              size='small'
              onClick={() =>
                isDeactivated
                  ? handleActivate(record)
                  : handleDeactivate(record)
              }
            >
              {isDeactivated ? t('启用') : t('注销')}
            </Button>
            <Button
              theme='light'
              type='secondary'
              size='small'
              onClick={() => openDashboard(record)}
            >
              {t('数据看板')}
            </Button>
          </Space>
        );
      },
    },
  ];
};
