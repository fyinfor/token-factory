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

import React, { useState, useRef, useEffect } from 'react';
import { Modal, Form, Typography } from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

const DeactivateSupplierModal = ({ visible, supplier, handleClose, onSuccess }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);

  useEffect(() => {
    if (visible && !supplier) {
      formApiRef.current?.reset();
    }
  }, [visible, supplier]);

  const handleCancel = () => {
    handleClose();
    formApiRef.current?.reset();
  };

  const submit = async (values) => {
    if (!values.reason || values.reason.trim() === '') {
      showError(t('请填写注销原因'));
      return;
    }

    setLoading(true);
    try {
      const res = await API.post(`/api/user/supplier/application/deactivate`, {
        supplier_id: supplier.id,
        reason: values.reason,
      });
      
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('注销成功'));
        handleClose();
        formApiRef.current?.reset();
        if (onSuccess) {
          onSuccess();
        }
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(error.response?.data?.message || t('注销失败'));
    }
    setLoading(false);
  };

  return (
    <Modal
      title={t('注销供应商')}
      visible={visible}
      onOk={() => formApiRef.current?.submitForm()}
      onCancel={handleCancel}
      confirmLoading={loading}
      okButtonProps={{ type: 'danger' }}
    >
      <Form
        getFormApi={(api) => (formApiRef.current = api)}
        onSubmit={submit}
      >
        <div className='mb-4'>
          <Text type='tertiary'>
            {t('确定要注销供应商')}: <Text strong>{supplier?.company_name}</Text> ?
          </Text>
        </div>

        <Form.TextArea
          field='reason'
          label={<Text strong>{t('注销原因')}<Text type='danger'>*</Text></Text>}
          placeholder={t('请填写注销原因，如：不再合作、违规操作等')}
          rules={[{ required: true, message: t('请填写注销原因') }]}
          rows={4}
          maxLength={500}
          showClear
        />
      </Form>
    </Modal>
  );
};

export default DeactivateSupplierModal;
