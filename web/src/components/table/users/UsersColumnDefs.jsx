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
import {
  Button,
  Space,
  Tag,
  Tooltip,
  Progress,
  Popover,
  Typography,
  Dropdown,
} from '@douyinfe/semi-ui';
import { IconMore, IconTick, IconClose } from '@douyinfe/semi-icons';
import { renderGroup, renderNumber, renderQuota } from '../../../helpers';
import { formatDateTimeString } from '../../../helpers/utils';
import { USER_ROLES } from '../../../constants/user.constants';

/**
 * Render user role（身份等级 + 可选「代理」标记）
 */
const renderRole = (role, record, t) => {
  const legacyDistributorRole = role === USER_ROLES.DISTRIBUTOR;
  const isDistributor =
    record.is_distributor === 1 ||
    record.is_distributor === true ||
    legacyDistributorRole;
  const isSupplier = !!record.supplier_id && record.supplier_id !== 0;
  const baseRole = legacyDistributorRole ? USER_ROLES.USER : role;
  let baseTag;
  switch (baseRole) {
    case USER_ROLES.USER:
      baseTag = (
        <Tag color='blue' shape='circle'>
          {t('普通用户')}
        </Tag>
      );
      break;
    case USER_ROLES.ADMIN:
      baseTag = (
        <Tag color='yellow' shape='circle'>
          {t('管理员')}
        </Tag>
      );
      break;
    case USER_ROLES.ROOT:
      baseTag = (
        <Tag color='orange' shape='circle'>
          {t('超级管理员')}
        </Tag>
      );
      break;
    default:
      baseTag = (
        <Tag color='red' shape='circle'>
          {t('未知身份')}
        </Tag>
      );
  }

  return (
    <Space spacing={4}>
      {baseTag}
      {isDistributor && (
        <Tag color='green' shape='circle'>
          {t('代理')}
        </Tag>
      )}
      {isSupplier && (
        <Tag color='purple' shape='circle'>
          {t('供应商')}
        </Tag>
      )}
    </Space>
  );
};

/** users.created_by 枚举展示 */
const renderCreatedBy = (v, t) => {
  if (v == null || v === '') {
    return '—';
  }
  const keyMap = {
    registration: t('用户创建_系统注册'),
    admin: t('用户创建_管理员创建'),
    import: t('用户创建_导入'),
    bootstrap: t('用户创建_安装初始化'),
  };
  return keyMap[v] || v;
};

/** API 返回 RFC3339 时间字符串；旧数据或零值显示为 — */
const renderUserDateTime = (value) => {
  if (value == null || value === '') {
    return '—';
  }
  const d = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1970) {
    return '—';
  }
  return formatDateTimeString(d);
};

/**
 * Render username with remark
 */
const renderUsername = (text, record) => {
  const remark = record.remark;
  if (!remark) {
    return <span>{text}</span>;
  }
  const maxLen = 10;
  const displayRemark =
    remark.length > maxLen ? remark.slice(0, maxLen) + '…' : remark;
  return (
    <Space spacing={2}>
      <span>{text}</span>
      <Tooltip content={remark} position='top' showArrow>
        <Tag color='white' shape='circle' className='!text-xs'>
          <div className='flex items-center gap-1'>
            <div
              className='w-2 h-2 flex-shrink-0 rounded-full'
              style={{ backgroundColor: '#10b981' }}
            />
            {displayRemark}
          </div>
        </Tag>
      </Tooltip>
    </Space>
  );
};

/**
 * Render user statistics
 */
const renderStatistics = (text, record, showEnableDisableModal, t) => {
  const isDeleted = record.DeletedAt !== null;

  // Determine tag text & color like original status column
  let tagColor = 'grey';
  let tagText = t('未知状态');
  if (isDeleted) {
    tagColor = 'red';
    tagText = t('已注销');
  } else if (record.status === 1) {
    tagColor = 'green';
    tagText = t('已启用');
  } else if (record.status === 2) {
    tagColor = 'red';
    tagText = t('已禁用');
  }

  const content = (
    <Tag color={tagColor} shape='circle' size='small'>
      {tagText}
    </Tag>
  );

  const tooltipContent = (
    <div className='text-xs'>
      <div>
        {t('调用次数')}: {renderNumber(record.request_count)}
      </div>
    </div>
  );

  return (
    <Tooltip content={tooltipContent} position='top'>
      {content}
    </Tooltip>
  );
};

// Render separate quota usage column
const renderQuotaUsage = (text, record, t) => {
  const { Paragraph } = Typography;
  const used = parseInt(record.used_quota) || 0;
  const remain = parseInt(record.quota) || 0;
  const total = used + remain;
  const percent = total > 0 ? (remain / total) * 100 : 0;
  const popoverContent = (
    <div className='text-xs p-2'>
      <Paragraph copyable={{ content: renderQuota(used) }}>
        {t('已用额度')}: {renderQuota(used)}
      </Paragraph>
      <Paragraph copyable={{ content: renderQuota(remain) }}>
        {t('剩余额度')}: {renderQuota(remain)} ({percent.toFixed(0)}%)
      </Paragraph>
      <Paragraph copyable={{ content: renderQuota(total) }}>
        {t('总额度')}: {renderQuota(total)}
      </Paragraph>
    </div>
  );
  return (
    <Popover content={popoverContent} position='top'>
      <Tag color='white' shape='circle'>
        <div className='flex flex-col items-end'>
          <span className='text-xs leading-none'>{`${renderQuota(remain)} / ${renderQuota(total)}`}</span>
          <Progress
            percent={percent}
            aria-label='quota usage'
            format={() => `${percent.toFixed(0)}%`}
            style={{ width: '100%', marginTop: '1px', marginBottom: 0 }}
          />
        </div>
      </Tag>
    </Popover>
  );
};

/**
 * Render invite information
 */
const renderInviteInfo = (text, record, t) => {
  return (
    <div>
      <Space spacing={1}>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('邀请')}: {renderNumber(record.aff_count)}
        </Tag>
        <Tag color='white' shape='circle' className='!text-xs'>
          {t('收益')}: {renderQuota(record.aff_history_quota)}
        </Tag>
        <Tag color='white' shape='circle' className='!text-xs'>
          {record.inviter_id === 0
            ? t('无邀请人')
            : `${t('邀请人')}: ${record.inviter_id}`}
        </Tag>
      </Space>
    </div>
  );
};

const renderStudentStatus = (record, t) => {
  if (record.is_student === 1 || record.is_student === true) {
    return (
      <Tag color='green' shape='circle' className='!text-xs'>
        {t('学员')}
      </Tag>
    );
  }
  if (record.student_status === 1) {
    return (
      <Tag color='orange' shape='circle' className='!text-xs'>
        {t('待审批')}
      </Tag>
    );
  }
  if (record.student_status === 3) {
    return (
      <Tag color='grey' shape='circle' className='!text-xs'>
        {t('已拒绝')}
      </Tag>
    );
  }
  return (
    <Tag color='white' shape='circle' className='!text-xs'>
      {t('非学员')}
    </Tag>
  );
};

/**
 * Render operations column
 */
const renderOperations = (
  text,
  record,
  {
    setEditingUser,
    setShowEditUser,
    showPromoteModal,
    showDemoteModal,
    showEnableDisableModal,
    showDeleteModal,
    showResetPasskeyModal,
    showResetTwoFAModal,
    showUserSubscriptionsModal,
    manageUser,
    studentView,
    t,
  },
) => {
  if (record.DeletedAt !== null) {
    return <></>;
  }

  if (studentView === 'pending') {
    return (
      <Space>
        <Button
          type='primary'
          size='small'
          onClick={() => manageUser(record.id, 'approve_student', record)}
        >
          {t('同意')}
        </Button>
        <Button
          type='danger'
          theme='light'
          size='small'
          onClick={() => manageUser(record.id, 'reject_student', record)}
        >
          {t('拒绝')}
        </Button>
      </Space>
    );
  }

  const moreMenu = [
    {
      node: 'item',
      name: t('订阅管理'),
      onClick: () => showUserSubscriptionsModal(record),
    },
    {
      node: 'divider',
    },
    {
      node: 'item',
      name: t('重置 Passkey'),
      onClick: () => showResetPasskeyModal(record),
    },
    {
      node: 'item',
      name: t('重置 2FA'),
      onClick: () => showResetTwoFAModal(record),
    },
    {
      node: 'divider',
    },
    ...(record.student_status === 1
      ? [
          {
            node: 'item',
            name: t('通过学员'),
            onClick: () => manageUser(record.id, 'approve_student', record),
          },
          {
            node: 'item',
            name: t('拒绝学员'),
            type: 'danger',
            onClick: () => manageUser(record.id, 'reject_student', record),
          },
          {
            node: 'divider',
          },
        ]
      : []),
    ...((record.is_student === 1 || record.is_student === true) &&
    record.role < USER_ROLES.ADMIN
      ? [
          {
            node: 'item',
            name: t('撤销学员'),
            type: 'danger',
            onClick: () => manageUser(record.id, 'unset_student', record),
          },
          {
            node: 'divider',
          },
        ]
      : []),
    ...(record.role === USER_ROLES.USER &&
    record.status === 1 &&
    record.student_status !== 1 &&
    !(record.is_student === 1 || record.is_student === true)
      ? [
          {
            node: 'item',
            name: t('设为学员'),
            onClick: () => manageUser(record.id, 'set_student', record),
          },
          {
            node: 'divider',
          },
        ]
      : []),
    {
      node: 'item',
      name: t('注销'),
      type: 'danger',
      onClick: () => showDeleteModal(record),
    },
  ];

  return (
    <Space>
      {record.status === 1 ? (
        <Button
          type='danger'
          size='small'
          onClick={() => showEnableDisableModal(record, 'disable')}
        >
          {t('禁用')}
        </Button>
      ) : (
        <Button
          size='small'
          onClick={() => showEnableDisableModal(record, 'enable')}
        >
          {t('启用')}
        </Button>
      )}
      <Button
        type='tertiary'
        size='small'
        onClick={() => {
          setEditingUser(record);
          setShowEditUser(true);
        }}
      >
        {t('编辑')}
      </Button>
      <Button
        type='warning'
        size='small'
        onClick={() => showPromoteModal(record)}
      >
        {t('提升')}
      </Button>
      <Button
        type='secondary'
        size='small'
        onClick={() => showDemoteModal(record)}
      >
        {t('降级')}
      </Button>
      {record.role === USER_ROLES.USER &&
        record.is_distributor !== 1 &&
        record.is_distributor !== true &&
        record.role < USER_ROLES.ADMIN && (
          <Button
            type='primary'
            theme='light'
            size='small'
            onClick={() => manageUser(record.id, 'set_distributor', record)}
          >
            {t('设为代理')}
          </Button>
        )}
      {(record.is_distributor === 1 ||
        record.is_distributor === true ||
        record.role === USER_ROLES.DISTRIBUTOR) &&
        record.role < USER_ROLES.ADMIN && (
          <Button
            type='tertiary'
            size='small'
            onClick={() => manageUser(record.id, 'unset_distributor', record)}
          >
            {t('取消代理')}
          </Button>
        )}
      <Dropdown menu={moreMenu} trigger='click' position='bottomRight'>
        <Button type='tertiary' size='small' icon={<IconMore />} />
      </Dropdown>
    </Space>
  );
};

/**
 * Get users table column definitions
 */
export const getUsersColumns = ({
  t,
  setEditingUser,
  setShowEditUser,
  showPromoteModal,
  showDemoteModal,
  showEnableDisableModal,
  showDeleteModal,
  showResetPasskeyModal,
  showResetTwoFAModal,
  showUserSubscriptionsModal,
  manageUser,
  studentView,
}) => {
  return [
    {
      title: 'ID',
      dataIndex: 'id',
    },
    {
      title: t('用户名'),
      dataIndex: 'username',
      render: (text, record) => renderUsername(text, record),
    },
    {
      title: t('手机号'),
      dataIndex: 'phone',
      render: (v) => <span>{v || '—'}</span>,
    },
    {
      title: t('状态'),
      dataIndex: 'info',
      render: (text, record, index) =>
        renderStatistics(text, record, showEnableDisableModal, t),
    },
    {
      title: t('剩余额度/总额度'),
      key: 'quota_usage',
      render: (text, record) => renderQuotaUsage(text, record, t),
    },
    {
      title: t('分组'),
      dataIndex: 'group',
      render: (text, record, index) => {
        return <div>{renderGroup(text)}</div>;
      },
    },
    {
      title: t('角色'),
      dataIndex: 'role',
      render: (text, record, index) => {
        return <div>{renderRole(text, record, t)}</div>;
      },
    },
    {
      title: t('学员状态'),
      dataIndex: 'student_status',
      render: (text, record) => renderStudentStatus(record, t),
    },
    {
      title: t('邀请信息'),
      dataIndex: 'invite',
      render: (text, record, index) => renderInviteInfo(text, record, t),
    },
    {
      title: t('创建人'),
      dataIndex: 'created_by',
      width: 120,
      render: (v) => (
        <span className='text-xs whitespace-nowrap'>
          {renderCreatedBy(v, t)}
        </span>
      ),
    },
    {
      title: t('注册时间'),
      dataIndex: 'created_at',
      width: 152,
      render: (v) => (
        <span className='text-xs whitespace-nowrap'>
          {renderUserDateTime(v)}
        </span>
      ),
    },
    {
      title: t('修改时间'),
      dataIndex: 'updated_at',
      width: 152,
      render: (v) => (
        <span className='text-xs whitespace-nowrap'>
          {renderUserDateTime(v)}
        </span>
      ),
    },
    {
      title: t('上次登录'),
      dataIndex: 'last_login_at',
      width: 152,
      render: (v) => (
        <span className='text-xs whitespace-nowrap'>
          {renderUserDateTime(v)}
        </span>
      ),
    },
    {
      title: '',
      dataIndex: 'operate',
      fixed: 'right',
      width: 200,
      render: (text, record, index) =>
        renderOperations(text, record, {
          setEditingUser,
          setShowEditUser,
          showPromoteModal,
          showDemoteModal,
          showEnableDisableModal,
          showDeleteModal,
          showResetPasskeyModal,
          showResetTwoFAModal,
          showUserSubscriptionsModal,
          manageUser,
          studentView,
          t,
        }),
    },
  ];
};
