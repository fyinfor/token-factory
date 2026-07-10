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

import React, { useEffect, useState, useRef, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  renderQuota,
  renderQuotaWithPrompt,
  getCurrencyConfig,
} from '../../../../helpers';
import {
  quotaToDisplayAmount,
  displayAmountToQuota,
} from '../../../../helpers/quota';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Button,
  Modal,
  SideSheet,
  Space,
  Spin,
  Typography,
  Card,
  Tag,
  Form,
  Avatar,
  Row,
  Col,
  InputNumber,
  Radio,
  Input,
} from '@douyinfe/semi-ui';
import {
  IconUser,
  IconSave,
  IconClose,
  IconLink,
  IconUserGroup,
  IconPlus,
} from '@douyinfe/semi-icons';
import UserBindingManagementModal from './UserBindingManagementModal';
import { buildAdminUserPhoneFieldRules } from './userPhoneFormRules';
import { buildAdminUserEmailFieldRules } from './userEmailFormRules';
import CountryPhoneInput from '../../../common/form/CountryPhoneInput';

const { Text, Title } = Typography;

const EditUserModal = (props) => {
  const { t } = useTranslation();
  const userId = props.editingUser.id;
  const tagOptions = props.tagOptions || [];
  const intlEnabled = Boolean(props.intlEnabled);
  const [loading, setLoading] = useState(true);
  const [addQuotaModalOpen, setIsModalOpen] = useState(false);
  const [addQuotaLocal, setAddQuotaLocal] = useState('');
  const [addAmountLocal, setAddAmountLocal] = useState('');
  const [addQuotaType, setAddQuotaType] = useState('gift');
  const [addQuotaReference, setAddQuotaReference] = useState('');
  const [addQuotaRemark, setAddQuotaRemark] = useState('');
  const [addQuotaSubmitting, setAddQuotaSubmitting] = useState(false);
  const isMobile = useIsMobile();
  const [groupOptions, setGroupOptions] = useState([]);
  const [bindingModalVisible, setBindingModalVisible] = useState(false);
  const formApiRef = useRef(null);
  const phoneValidateRef = useRef(null);
  // 单独维护手机号：loadUser 完成后用接口返回的回显值，
  // 避免依赖 Semi Form render prop 中的 values 在 setValues 之后首帧未及时下发。
  const [phoneValue, setPhoneValue] = useState('');

  const isEdit = Boolean(userId);

  const mergedTagOptions = useMemo(() => {
    const seen = new Set();
    return tagOptions
      .map(o => o.value || o)
      .filter(tag => {
        if (!tag || seen.has(tag)) return false;
        seen.add(tag);
        return true;
      })
      .map(tag => ({ label: tag, value: tag }));
  }, [tagOptions]);

  /** 编辑用户表单初始字段（加载后与接口返回数据合并）。 */
  const getInitValues = () => ({
    username: '',
    display_name: '',
    phone: '',
    password: '',
    github_id: '',
    oidc_id: '',
    discord_id: '',
    wechat_id: '',
    telegram_id: '',
    linux_do_id: '',
    email: '',
    quota: 0,
    group: 'default',
    tags: [],
    remark: '',
  });

  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/group/`);
      setGroupOptions(res.data.data.map((g) => ({ label: g, value: g })));
    } catch (e) {
      showError(e.message);
    }
  };

  const handleCancel = () => props.handleClose();

  const loadUser = async () => {
    setLoading(true);
    const url = userId ? `/api/user/${userId}` : `/api/user/self`;
    const res = await API.get(url);
    const { success, message, data } = res.data;
    if (success) {
      data.password = '';
      // Convert tags string to array for TagInput
      if (data.tags && typeof data.tags === 'string') {
        data.tags = data.tags.split(',').map(tag => tag.trim()).filter(Boolean);
      } else if (!data.tags) {
        data.tags = [];
      }
      formApiRef.current?.setValues({ ...getInitValues(), ...data });
      // 直接以接口返回的 phone 回显到 CountryPhoneInput，绕开 Semi Form
      // render prop 中 values 首次帧下发不及时导致的手机号空白问题。
      setPhoneValue(data.phone || '');
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    // 切换编辑目标时重置本地手机号，避免上一个用户的号码被带入。
    setPhoneValue('');
    loadUser();
    if (userId) fetchGroups();
    setBindingModalVisible(false);
  }, [props.editingUser.id]);

  const openBindingModal = () => {
    setBindingModalVisible(true);
  };

  const closeBindingModal = () => {
    setBindingModalVisible(false);
  };

  /* ----------------------- submit ----------------------- */
  const submit = async (values) => {
    // 优先使用本地手机号 state（编辑回显的来源），避免 Semi Form
    // 中 values.phone 在某些时序下不是最新输入。
    const effectivePhone = phoneValue || values.phone || '';
    // 手动跑 phone 字段完整校验
    if (phoneValidateRef.current) {
      const err = await phoneValidateRef.current.runFullValidate(
        effectivePhone,
      );
      if (err) {
        showError(err);
        return;
      }
    }
    setLoading(true);
    let payload = { ...values, phone: effectivePhone };
    if (typeof payload.quota === 'string')
      payload.quota = parseInt(payload.quota) || 0;
    if (userId) {
      payload.id = parseInt(userId);
    }
    // Convert tags array to comma-separated string
    if (Array.isArray(payload.tags)) {
      payload.tags = payload.tags.join(',');
    }
    const url = userId ? `/api/user/` : `/api/user/self`;
    const res = await API.put(url, payload);
    const { success, message } = res.data;
    if (success) {
      showSuccess(t('用户信息更新成功！'));
      props.refresh();
      props.handleClose();
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const resetAddQuotaModal = () => {
    setIsModalOpen(false);
    setAddQuotaLocal('');
    setAddAmountLocal('');
    setAddQuotaType('gift');
    setAddQuotaReference('');
    setAddQuotaRemark('');
  };

  const submitAddQuota = async () => {
    const delta = parseInt(addQuotaLocal, 10) || 0;
    if (!userId || delta <= 0) {
      showError(t('请输入有效额度'));
      return;
    }
    setAddQuotaSubmitting(true);
    try {
      if (addQuotaType === 'corporate') {
        const money = Number(addAmountLocal);
        if (!money || money <= 0) {
          showError(t('对公入账请填写金额'));
          return;
        }
        const reference = addQuotaReference.trim();
        const res = await API.post('/api/user/invoice/admin/corporate-topup', {
          user_id: userId,
          money,
          quota: delta,
          reference,
          remark: addQuotaRemark.trim(),
        });
        if (!res.data.success) {
          showError(res.data.message || t('对公入账失败'));
          return;
        }
        showSuccess(t('对公入账成功'));
      } else {
        const res = await API.post('/api/user/invoice/admin/grant-gift', {
          user_id: userId,
          quota: delta,
          remark: addQuotaRemark.trim(),
        });
        if (!res.data.success) {
          showError(res.data.message || t('赠送失败'));
          return;
        }
        showSuccess(t('赠送额度已到账'));
      }
      await loadUser();
      resetAddQuotaModal();
    } catch (e) {
      showError(e);
    } finally {
      setAddQuotaSubmitting(false);
    }
  };

  /* --------------------------- UI --------------------------- */
  return (
    <>
      <SideSheet
        placement='right'
        title={
          <Space>
            <Tag color='blue' shape='circle'>
              {t(isEdit ? '编辑' : '新建')}
            </Tag>
            <Title heading={4} className='m-0'>
              {isEdit ? t('编辑用户') : t('创建用户')}
            </Title>
          </Space>
        }
        bodyStyle={{ padding: 0 }}
        visible={props.visible}
        width={isMobile ? '100%' : 600}
        footer={
          <div className='flex justify-end bg-white'>
            <Space>
              <Button
                theme='solid'
                onClick={() => formApiRef.current?.submitForm()}
                icon={<IconSave />}
                loading={loading}
              >
                {t('提交')}
              </Button>
              <Button
                theme='light'
                type='primary'
                onClick={handleCancel}
                icon={<IconClose />}
              >
                {t('取消')}
              </Button>
            </Space>
          </div>
        }
        closeIcon={null}
        onCancel={handleCancel}
      >
        <Spin spinning={loading}>
          <Form
            initValues={getInitValues()}
            getFormApi={(api) => (formApiRef.current = api)}
            onSubmit={submit}
          >
            {({ values, formApi }) => (
              <div className='p-2 space-y-3'>
                {/* 基本信息 */}
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='blue'
                      className='mr-2 shadow-md'
                    >
                      <IconUser size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('基本信息')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('用户的基本账户信息')}
                      </div>
                    </div>
                  </div>

                  <Row gutter={12}>
                    <Col span={24}>
                      <Form.Input
                        field='username'
                        label={t('用户名')}
                        placeholder={t('请输入新的用户名')}
                        rules={[{ required: true, message: t('请输入用户名') }]}
                        showClear
                      />
                    </Col>

                    <Col span={24}>
                      <Form.Input
                        field='password'
                        label={t('密码')}
                        placeholder={t('请输入新的密码，最短 8 位')}
                        mode='password'
                        showClear
                      />
                    </Col>

                    <Col span={24}>
                      <Form.Input
                        field='display_name'
                        label={t('显示名称')}
                        placeholder={t('请输入新的显示名称')}
                        showClear
                      />
                    </Col>

                    <Col span={24}>
                      <Form.Input
                        field='email'
                        label={t('邮箱')}
                        placeholder={t('请输入邮箱地址')}
                        showClear
                        rules={buildAdminUserEmailFieldRules(t)}
                      />
                    </Col>

                    {/* 手机号：编辑时由 GET 回显；格式 + 异步占用校验。完全脱离 Form.Slot/Form.Input 占位。 */}
                    <Col span={24}>
                      <div
                        style={{
                          marginBottom: 4,
                          fontSize: 14,
                          color: 'var(--semi-color-text-1)',
                        }}
                      >
                        {t('手机号')}
                      </div>
                      <CountryPhoneInput
                        value={phoneValue}
                        onChange={(v) => {
                          setPhoneValue(v);
                          formApi.setValue('phone', v);
                        }}
                        intlEnabled={intlEnabled}
                        placeholder={t('请输入手机号')}
                        rules={buildAdminUserPhoneFieldRules(t, {
                          intlEnabled,
                          excludeUserId: () =>
                            userId || formApiRef.current?.getValue('id'),
                        })}
                        validateRef={phoneValidateRef}
                      />
                    </Col>

                    <Col span={24}>
                      <Form.Select
                        field='tags'
                        label={t('标签')}
                        placeholder={t('请选择或输入标签')}
                        multiple
                        allowCreate
                        filter
                        optionList={mergedTagOptions}
                        showClear
                        style={{ width: '100%' }}
                      />
                    </Col>

                    <Col span={24}>
                      <Form.Input
                        field='remark'
                        label={t('备注')}
                        placeholder={t('请输入备注（仅管理员可见）')}
                        showClear
                      />
                    </Col>
                  </Row>
                </Card>

                {/* 权限设置 */}
                {userId && (
                  <Card className='!rounded-2xl shadow-sm border-0'>
                    <div className='flex items-center mb-2'>
                      <Avatar
                        size='small'
                        color='green'
                        className='mr-2 shadow-md'
                      >
                        <IconUserGroup size={16} />
                      </Avatar>
                      <div>
                        <Text className='text-lg font-medium'>
                          {t('权限设置')}
                        </Text>
                        <div className='text-xs text-gray-600'>
                          {t('用户分组和额度管理')}
                        </div>
                      </div>
                    </div>

                    <Row gutter={12}>
                      <Col span={24}>
                        <Form.Select
                          field='group'
                          label={t('分组')}
                          placeholder={t('请选择分组')}
                          optionList={groupOptions}
                          allowAdditions
                          search
                          rules={[{ required: true, message: t('请选择分组') }]}
                        />
                      </Col>

                      <Col span={10}>
                        <Form.InputNumber
                          field='quota'
                          label={t('剩余额度')}
                          placeholder={t('请输入新的剩余额度')}
                          step={500000}
                          extraText={renderQuotaWithPrompt(values.quota || 0)}
                          rules={[{ required: true, message: t('请输入额度') }]}
                          style={{ width: '100%' }}
                        />
                      </Col>

                      <Col span={14}>
                        <Form.Slot label={t('添加额度')}>
                          <Button
                            icon={<IconPlus />}
                            onClick={() => setIsModalOpen(true)}
                          />
                        </Form.Slot>
                      </Col>
                    </Row>
                  </Card>
                )}

                {/* 绑定信息入口 */}
                {userId && (
                  <Card className='!rounded-2xl shadow-sm border-0'>
                    <div className='flex items-center justify-between gap-3'>
                      <div className='flex items-center min-w-0'>
                        <Avatar
                          size='small'
                          color='purple'
                          className='mr-2 shadow-md'
                        >
                          <IconLink size={16} />
                        </Avatar>
                        <div className='min-w-0'>
                          <Text className='text-lg font-medium'>
                            {t('绑定信息')}
                          </Text>
                          <div className='text-xs text-gray-600'>
                            {t('管理用户已绑定的第三方账户，支持筛选与解绑')}
                          </div>
                        </div>
                      </div>
                      <Button
                        type='primary'
                        theme='outline'
                        onClick={openBindingModal}
                      >
                        {t('管理绑定')}
                      </Button>
                    </div>
                  </Card>
                )}
              </div>
            )}
          </Form>
        </Spin>
      </SideSheet>

      <UserBindingManagementModal
        visible={bindingModalVisible}
        onCancel={closeBindingModal}
        userId={userId}
        isMobile={isMobile}
        formApiRef={formApiRef}
      />

      {/* 添加额度模态框 */}
      <Modal
        centered
        visible={addQuotaModalOpen}
        confirmLoading={addQuotaSubmitting}
        onOk={submitAddQuota}
        onCancel={resetAddQuotaModal}
        closable={null}
        title={
          <div className='flex items-center'>
            <IconPlus className='mr-2' />
            {t('添加额度')}
          </div>
        }
      >
        <div className='mb-4'>
          <Text type='secondary' className='block mb-2'>
            {t('入账类型')}
          </Text>
          <Radio.Group
            value={addQuotaType}
            onChange={(e) => setAddQuotaType(e.target.value)}
          >
            <Radio value='gift'>{t('赠送（不可开票）')}</Radio>
            <Radio value='corporate'>{t('对公入账（可开票）')}</Radio>
          </Radio.Group>
        </div>
        {addQuotaType === 'corporate' ? (
          <div className='mb-3'>
            <div className='mb-1'>
              <Text size='small'>{t('对公凭证号/流水号')}</Text>
            </div>
            <Input
              value={addQuotaReference}
              onChange={setAddQuotaReference}
              placeholder={t('银行转账流水号等（可选）')}
            />
          </div>
        ) : null}
        {getCurrencyConfig().type !== 'TOKENS' && (
          <div className='mb-3'>
            <div className='mb-1'>
              <Text size='small'>{t('金额')}</Text>
              <Text size='small' type='tertiary'>
                {' '}
                ({t('仅用于换算，实际保存的是额度')})
              </Text>
            </div>
            <InputNumber
              prefix={getCurrencyConfig().symbol}
              placeholder={t('输入金额')}
              value={addAmountLocal}
              precision={2}
              onChange={(val) => {
                setAddAmountLocal(val);
                setAddQuotaLocal(
                  val != null && val !== ''
                    ? displayAmountToQuota(Math.abs(val)) * Math.sign(val)
                    : '',
                );
              }}
              style={{ width: '100%' }}
              showClear
            />
          </div>
        )}
        <div>
          <div className='mb-1'>
            <Text size='small'>{t('额度')}</Text>
          </div>
          <InputNumber
            placeholder={t('输入额度')}
            value={addQuotaLocal}
            onChange={(val) => {
              setAddQuotaLocal(val);
              setAddAmountLocal(
                val != null && val !== ''
                  ? Number(
                      (
                        quotaToDisplayAmount(Math.abs(val)) * Math.sign(val)
                      ).toFixed(2),
                    )
                  : '',
              );
            }}
            style={{ width: '100%' }}
            showClear
            step={500000}
          />
        </div>
        <div className='mt-3'>
          <div className='mb-1'>
            <Text size='small'>{t('备注')}</Text>
          </div>
          <Input
            value={addQuotaRemark}
            onChange={setAddQuotaRemark}
            placeholder={t('可选')}
          />
        </div>
      </Modal>
    </>
  );
};

export default EditUserModal;
