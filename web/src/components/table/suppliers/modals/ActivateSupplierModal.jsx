/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useRef, useState } from 'react';
import { Form, Modal, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

/**
 * ActivateSupplierModal 管理员启用已注销供应商。
 */
const ActivateSupplierModal = ({
  visible,
  supplier,
  handleClose,
  onSuccess,
}) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);

  /**
   * 提交启用供应商请求。
   */
  const submit = async (values) => {
    setLoading(true);
    try {
      const res = await API.post('/api/user/supplier/application/activate', {
        supplier_id: supplier?.id,
        reason: (values.reason || '').trim(),
      });
      const { success, message } = res.data || {};
      if (success) {
        showSuccess(t('启用成功'));
        handleClose();
        formApiRef.current?.reset();
        onSuccess?.();
      } else {
        showError(t(message || '启用失败'));
      }
    } catch (error) {
      showError(error.response?.data?.message || t('启用失败'));
    }
    setLoading(false);
  };

  return (
    <Modal
      title={t('启用供应商')}
      visible={visible}
      onOk={() => formApiRef.current?.submitForm()}
      onCancel={() => {
        handleClose();
        formApiRef.current?.reset();
      }}
      confirmLoading={loading}
    >
      <Form getFormApi={(api) => (formApiRef.current = api)} onSubmit={submit}>
        <div className='mb-4'>
          <Text type='tertiary'>
            {t('确定要启用供应商')}：
            <Text strong>{supplier?.company_name}</Text> ?
          </Text>
        </div>
        <Form.TextArea
          field='reason'
          label={<Text strong>{t('启用说明')}</Text>}
          placeholder={t('可选：填写启用说明')}
          rows={3}
          maxLength={500}
          showClear
        />
      </Form>
    </Modal>
  );
};

export default ActivateSupplierModal;
