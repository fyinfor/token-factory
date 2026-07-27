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
import { Modal, Pagination } from '@douyinfe/semi-ui';
import clsx from 'clsx';
import { Inbox, LoaderCircle, X } from 'lucide-react';
import './invoice-workspace.css';

export const formatInvoiceMoney = (value) =>
  `¥${Number(value || 0).toFixed(2)}`;

export const parseInvoiceProfile = (raw) => {
  if (!raw) return null;
  try {
    return typeof raw === 'string' ? JSON.parse(raw) : raw;
  } catch {
    return null;
  }
};

const statusLabels = {
  pending: '待处理',
  processing: '处理中',
  issued: '已开具',
  rejected: '已驳回',
  cancelled: '已取消',
};

export function InvoiceStatusBadge({ status, t }) {
  return (
    <span className={clsx('invoice-status', `is-${status || 'unknown'}`)}>
      <span aria-hidden='true' />
      {t(statusLabels[status] || status || '未知状态')}
    </span>
  );
}

export function InvoiceSpinner({ label }) {
  return (
    <span className='invoice-spinner' role='status'>
      <LoaderCircle size={16} aria-hidden='true' />
      {label ? <span>{label}</span> : null}
    </span>
  );
}

export function InvoiceEmptyState({ title, description, action }) {
  return (
    <div className='invoice-empty-state'>
      <span className='invoice-empty-icon' aria-hidden='true'>
        <Inbox size={24} />
      </span>
      <strong>{title}</strong>
      {description ? <p>{description}</p> : null}
      {action || null}
    </div>
  );
}

export function InvoicePagination({ page, total, pageSize, onChange, t }) {
  const totalPages = Math.max(1, Math.ceil(Number(total || 0) / pageSize));
  if (totalPages <= 1) return null;
  return (
    <nav className='invoice-pagination' aria-label={t('分页')}>
      <span className='invoice-pagination-summary'>
        {page} / {totalPages}
      </span>
      <Pagination
        className='invoice-semi-pagination'
        currentPage={page}
        pageSize={pageSize}
        total={Number(total || 0)}
        showSizeChanger={false}
        size='small'
        onPageChange={onChange}
      />
    </nav>
  );
}

export function InvoiceDialog({
  open,
  onClose,
  title,
  eyebrow,
  children,
  footer,
  size = 'medium',
}) {
  const width = size === 'large' ? 820 : size === 'small' ? 440 : 620;

  return (
    <Modal
      visible={open}
      className={clsx('invoice-semi-modal', `is-${size}`)}
      width={width}
      centered
      maskClosable
      closeOnEsc
      closable={false}
      onCancel={onClose}
      footer={footer || null}
      footerFill={false}
      header={
        <header className='invoice-dialog-header'>
          <div>
            {eyebrow ? <span>{eyebrow}</span> : null}
            <h2 id='semi-modal-title'>{title}</h2>
          </div>
          <button
            type='button'
            className='invoice-icon-button'
            onClick={onClose}
            aria-label='Close'
            title='Close'
          >
            <X size={18} />
          </button>
        </header>
      }
    >
      {children}
    </Modal>
  );
}
