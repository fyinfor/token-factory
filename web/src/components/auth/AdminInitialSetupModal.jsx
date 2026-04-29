/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, {
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { Modal, Form, Typography, Space, Button } from '@douyinfe/semi-ui';
import { IconLock } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import {
  API,
  showError,
  showSuccess,
  mergeSelfResponseIntoLocalUser,
} from '../../helpers';
import { buildAdminUserPhoneFieldRules } from '../table/users/modals/userPhoneFormRules';

const { Text, Title } = Typography;

/**
 * 管理员代建账号首次登录后的强制引导：修改密码；若创建时未填手机号则必须绑定（占用校验与后台一致）。
 */
const AdminInitialSetupModal = () => {
  const { t } = useTranslation();
  const [state, dispatch] = useContext(UserContext);
  const formApiRef = useRef(null);
  const [submitting, setSubmitting] = useState(false);

  /** 合并 Context 与 localStorage，避免刷新后尚未 dispatch 时拿不到用户。 */
  const user = useMemo(() => {
    if (state.user) return state.user;
    try {
      const raw = localStorage.getItem('user');
      return raw ? JSON.parse(raw) : null;
    } catch {
      return null;
    }
  }, [state.user]);

  const visible = Boolean(user?.require_admin_initial_setup);
  const phoneRequired = Boolean(user?.admin_setup_phone_required);

  /** 弹窗打开时重置表单，避免残留上次输入。 */
  useEffect(() => {
    if (!visible) return;
    formApiRef.current?.reset();
  }, [visible, user?.id]);

  /** 提交首次设置：改密 + 必要时绑定手机，成功后刷新 /api/user/self 并合并本地态。 */
  const handleSubmit = useCallback(
    async (values) => {
      setSubmitting(true);
      try {
        const payload = {
          new_password: values.new_password,
          confirm_password: values.confirm_password,
        };
        if (phoneRequired) {
          payload.phone = (values.phone || '').trim();
        }
        const res = await API.post(
          '/api/user/self/admin_initial_setup',
          payload,
          { skipErrorHandler: true },
        );
        if (!res.data?.success) {
          showError(res.data?.message || t('操作失败'));
          return;
        }
        showSuccess(t('设置已保存'));
        const selfRes = await API.get('/api/user/self');
        if (selfRes.data?.success && selfRes.data.data) {
          mergeSelfResponseIntoLocalUser(selfRes.data.data, dispatch);
        }
      } catch (e) {
        const msg =
          e?.response?.data?.message || e.message || t('操作失败');
        showError(msg);
      } finally {
        setSubmitting(false);
      }
    },
    [dispatch, phoneRequired, t],
  );

  return (
    <Modal
      title={
        <Space>
          <IconLock />
          <Title heading={5} className='!m-0'>
            {t('首次登录安全设置')}
          </Title>
        </Space>
      }
      visible={visible}
      maskClosable={false}
      closable={false}
      footer={null}
      width={480}
      centered
      zIndex={1100}
    >
      <Text type='secondary' className='block mb-4'>
        {t('管理员为您创建了账号，请修改密码后再继续使用。')}
        {phoneRequired
          ? ` ${t('创建时未填写手机号，请绑定本人中国大陆手机号。')}`
          : ''}
      </Text>
      <Form
        getFormApi={(api) => {
          formApiRef.current = api;
        }}
        onSubmit={handleSubmit}
        onSubmitFail={(errs) => {
          const first = Object.values(errs)[0];
          if (first) showError(Array.isArray(first) ? first[0] : first);
          formApiRef.current?.scrollToError();
        }}
      >
        <Form.Input
          field='new_password'
          label={t('新密码')}
          mode='password'
          placeholder={t('请输入 8～20 位新密码')}
          rules={[
            { required: true, message: t('请输入新密码') },
            { min: 8, max: 20, message: t('密码长度须在 8～20 位之间') },
          ]}
        />
        <Form.Input
          field='confirm_password'
          label={t('确认新密码')}
          mode='password'
          placeholder={t('请再次输入新密码')}
          rules={[
            { required: true, message: t('请确认新密码') },
            {
              validator: (rule, value) => {
                const p = formApiRef.current?.getValue('new_password');
                if ((value || '') !== (p || '')) {
                  return new Error(t('两次输入的密码不一致'));
                }
                return true;
              },
            },
          ]}
        />
        {phoneRequired && (
          <Form.Input
            field='phone'
            label={t('手机号')}
            placeholder={t('请输入手机号')}
            showClear
            rules={[
              { required: true, message: t('请输入手机号') },
              ...buildAdminUserPhoneFieldRules(t, {
                checkPhoneUrl: '/api/user/self/phone_available',
              }),
            ]}
          />
        )}
        <Button
          block
          type='primary'
          theme='solid'
          htmlType='submit'
          loading={submitting}
        >
          {t('完成并继续')}
        </Button>
      </Form>
    </Modal>
  );
};

export default AdminInitialSetupModal;
