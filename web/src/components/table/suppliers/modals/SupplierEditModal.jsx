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

import React, { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { Modal, Form, Row, Col, Button, Typography, Upload, Divider, Input, Progress, Steps } from '@douyinfe/semi-ui';
import { IconUpload } from '@douyinfe/semi-icons';
import { debounce } from 'lodash-es';
import { API, showError, showSuccess, isAdmin } from '../../../../helpers';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import SupplierCapabilityFormFields from '../../../supplier/SupplierCapabilityFormFields';

const { Text } = Typography;

/**
 * 从用户下拉（数值或带 value 的对象）解析出关联用户 ID。
 * @param {number|object|null|undefined} raw 表单中的 user_id 原始值
 * @returns {number|null}
 */
function resolveApplicantUserId(raw) {
  if (raw == null || raw === '') {
    return null;
  }
  if (typeof raw === 'object' && raw !== null) {
    if (raw.value != null && raw.value !== '') {
      return Number(raw.value);
    }
    if (raw.id != null && raw.id !== '') {
      return Number(raw.id);
    }
    return null;
  }
  const n = Number(raw);
  return Number.isFinite(n) && n > 0 ? n : null;
}

/**
 * 新增供应商表单（含技术能力）的初始值。
 * @returns {object}
 */
function getCreateFormInitValues() {
  return {
    user_id: undefined,
    supplier_alias: '',
    company_name: '',
    credit_code: '',
    legal_representative: '',
    company_size: '',
    contact_name: '',
    contact_mobile: '',
    contact_wechat: '',
    core_service_types: [],
    supported_models: [],
    supported_model_notes: '',
    supported_api_endpoints: [],
    supported_api_endpoint_extra: '',
    supported_params: [],
    supported_params_extra: '',
    streaming_supported: 'no',
    streaming_notes: '',
    structured_output_supported: 'no',
    structured_output_notes: '',
    multimodal_types: [],
    multimodal_extra: '',
    pricing_modes: [],
    reference_input_price: '',
    reference_output_price: '',
    failure_billing_mode: 'no_bill',
    failure_billing_notes: '',
    api_base_urls: [],
    openai_compatible: 'yes',
    truth_commitment_confirmed: false,
  };
}

/**
 * 根据表单值组装技术能力 PUT 请求体。
 * @param {object} values Semi Form 值
 * @returns {object}
 */
function buildCapabilityPayload(values) {
  return {
    core_service_types: values.core_service_types || [],
    supported_models: values.supported_models || [],
    supported_model_notes: values.supported_model_notes || '',
    supported_api_endpoints: values.supported_api_endpoints || [],
    supported_api_endpoint_extra: values.supported_api_endpoint_extra || '',
    supported_params: values.supported_params || [],
    supported_params_extra: values.supported_params_extra || '',
    streaming_supported: values.streaming_supported === 'yes',
    streaming_notes: values.streaming_notes || '',
    structured_output_supported: values.structured_output_supported === 'yes',
    structured_output_notes: values.structured_output_notes || '',
    multimodal_types: values.multimodal_types || [],
    multimodal_extra: values.multimodal_extra || '',
    pricing_modes: values.pricing_modes || [],
    reference_input_price: values.reference_input_price || '',
    reference_output_price: values.reference_output_price || '',
    failure_billing_mode: values.failure_billing_mode || '',
    failure_billing_notes: values.failure_billing_notes || '',
    api_base_urls: values.api_base_urls || [],
    openai_compatible: values.openai_compatible === 'yes',
    truth_commitment_confirmed: !!values.truth_commitment_confirmed,
  };
}

/**
 * SupplierEditModal 管理员供应商新增（分步向导）与编辑弹窗。
 * @param {object} props
 * @param {boolean} props.visible 是否显示
 * @param {object|null} props.supplier 当前编辑的供应商；为 null 表示新增
 * @param {function} props.handleClose 关闭回调
 * @param {function} [props.onSuccess] 提交成功回调
 */
const SupplierEditModal = ({ visible, supplier, handleClose, onSuccess }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [fetchingData, setFetchingData] = useState(false);
  const formApiRef = useRef(null);
  const isMobile = useIsMobile();

  const [fileList, setFileList] = useState([]);
  const [businessLicenseUrl, setBusinessLicenseUrl] = useState('');
  const [userOptions, setUserOptions] = useState([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [supplierData, setSupplierData] = useState(null);
  /** 管理员新增时的分步索引 0–3 */
  const [currentStep, setCurrentStep] = useState(0);
  /** 管理员新增向导数据缓存，防止卸载步骤导致值丢失。 */
  const [wizardDraftValues, setWizardDraftValues] = useState({});
  /** 技术能力页真实性承诺勾选态（用于禁用提交按钮）。 */
  const [commitmentChecked, setCommitmentChecked] = useState(false);

  const isCreateWizard = !supplier && isAdmin();

  const loadInitialUsers = useCallback(async () => {
    setLoadingUsers(true);
    try {
      const res = await API.get('/api/user/?p=1&page_size=20');
      const { success, data, message } = res.data;
      if (success) {
        const options = (data.items || []).map(user => ({
          value: user.id,
          label: `${user.username}${user.display_name ? ` (${user.display_name})` : ''} - ID: ${user.id}`,
          username: user.username,
          display_name: user.display_name,
        }));
        setUserOptions(options);
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(error.response?.data?.message || t('加载用户列表失败'));
    }
    setLoadingUsers(false);
  }, [t]);

  /**
   * 远程搜索用户（防抖调用）。
   * @param {string} keyword 关键字
   */
  const handleSearch = async (keyword) => {
    if (!keyword || keyword.trim() === '') {
      return;
    }

    setLoadingUsers(true);
    try {
      const res = await API.get(`/api/user/search?keyword=${encodeURIComponent(keyword)}&page_size=20`);
      const { success, data, message } = res.data;
      if (success) {
        const options = (data.items || []).map(user => ({
          value: user.id,
          label: `${user.username}${user.display_name ? ` (${user.display_name})` : ''} - ID: ${user.id}`,
          username: user.username,
          display_name: user.display_name,
        }));
        setUserOptions(options);
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(error.response?.data?.message || t('加载用户列表失败'));
    }
    setLoadingUsers(false);
  };

  const searchUsers = useMemo(
    () => debounce(handleSearch, 500),
    []
  );

  useEffect(() => {
    return () => {
      searchUsers.cancel();
    };
  }, [searchUsers]);

  const fetchSupplierData = useCallback(async (supplierId) => {
    setFetchingData(true);
    try {
      const apiPath = isAdmin()
        ? `/api/user/supplier/${supplierId}`
        : '/api/user/supplier/application/self';
      const res = await API.get(apiPath, { skipErrorHandler: true });
      const { success, data, message } = res.data;
      if (success) {
        setSupplierData(data);
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(error.response?.data?.message || t('加载供应商数据失败'));
    }
    setFetchingData(false);
  }, [t]);

  useEffect(() => {
    if (visible && supplier && supplier.id) {
      fetchSupplierData(supplier.id);
    } else if (visible && !supplier) {
      setSupplierData(null);
      setCurrentStep(0);
      formApiRef.current?.setValues(getCreateFormInitValues());
      setFileList([]);
      setBusinessLicenseUrl('');
      setWizardDraftValues({});
      setCommitmentChecked(false);
      loadInitialUsers();
    }
  }, [visible, supplier, fetchSupplierData, loadInitialUsers]);

  /**
   * 新增向导步骤切换后回填缓存数据，防止返回上一步时输入内容被清空。
   */
  useEffect(() => {
    if (!isCreateWizard || !visible || !formApiRef.current) {
      return;
    }
    if (Object.keys(wizardDraftValues).length === 0) {
      return;
    }
    formApiRef.current.setValues(wizardDraftValues);
    setCommitmentChecked(!!wizardDraftValues.truth_commitment_confirmed);
  }, [isCreateWizard, currentStep, visible, wizardDraftValues]);

  useEffect(() => {
    if (supplierData) {
      formApiRef.current?.setValues({
        user_id: supplierData.user_id || null,
        supplier_alias: supplierData.supplier_alias || '',
        company_name: supplierData.company_name || '',
        credit_code: supplierData.credit_code || '',
        legal_representative: supplierData.legal_representative || '',
        company_size: supplierData.company_size || '',
        contact_name: supplierData.contact_name || '',
        contact_mobile: supplierData.contact_mobile || '',
        contact_wechat: supplierData.contact_wechat || '',
        core_service_types: supplierData.supplier_capability?.core_service_types || [],
        supported_models: supplierData.supplier_capability?.supported_models || [],
        supported_model_notes: supplierData.supplier_capability?.supported_model_notes || '',
        supported_api_endpoints: supplierData.supplier_capability?.supported_api_endpoints || [],
        supported_api_endpoint_extra: supplierData.supplier_capability?.supported_api_endpoint_extra || '',
        supported_params: supplierData.supplier_capability?.supported_params || [],
        supported_params_extra: supplierData.supplier_capability?.supported_params_extra || '',
        streaming_supported: supplierData.supplier_capability?.streaming_supported ? 'yes' : 'no',
        streaming_notes: supplierData.supplier_capability?.streaming_notes || '',
        structured_output_supported: supplierData.supplier_capability?.structured_output_supported ? 'yes' : 'no',
        structured_output_notes: supplierData.supplier_capability?.structured_output_notes || '',
        multimodal_types: supplierData.supplier_capability?.multimodal_types || [],
        multimodal_extra: supplierData.supplier_capability?.multimodal_extra || '',
        pricing_modes: supplierData.supplier_capability?.pricing_modes || [],
        reference_input_price: supplierData.supplier_capability?.reference_input_price || '',
        reference_output_price: supplierData.supplier_capability?.reference_output_price || '',
        failure_billing_mode: supplierData.supplier_capability?.failure_billing_mode || 'no_bill',
        failure_billing_notes: supplierData.supplier_capability?.failure_billing_notes || '',
        api_base_urls: supplierData.supplier_capability?.api_base_urls || [],
        openai_compatible: supplierData.supplier_capability?.openai_compatible ? 'yes' : 'no',
        truth_commitment_confirmed: supplierData.supplier_capability?.truth_commitment_confirmed || false,
      });
      setCommitmentChecked(!!supplierData.supplier_capability?.truth_commitment_confirmed);
      
      if (supplierData.business_license_url) {
        setBusinessLicenseUrl(supplierData.business_license_url);
      }

      if (supplierData.business_license_file) {
        try {
          const fileInfo = JSON.parse(supplierData.business_license_file);
          setFileList([fileInfo]);
        } catch (e) {
          console.error('Failed to parse business_license_file:', e);
          if (supplierData.business_license_url) {
            setFileList([{
              uid: 'existing',
              name: t('已上传的营业执照'),
              status: 'success',
              url: supplierData.business_license_url,
            }]);
          }
        }
      }

      if (supplierData.user_id) {
        setUserOptions([{
          value: supplierData.user_id,
          label: supplierData.applicant_username || `用户 ${supplierData.user_id}`,
          username: supplierData.applicant_username || `用户${supplierData.user_id}`,
          display_name: supplierData.display_name,
        }]);
      }
    }
  }, [supplierData, t]);

  /**
   * 关闭并重置本地状态。
   */
  const handleCancel = () => {
    handleClose();
    formApiRef.current?.reset();
    setFileList([]);
    setBusinessLicenseUrl('');
    setUserOptions([]);
    setSupplierData(null);
    setCurrentStep(0);
    setWizardDraftValues({});
    setCommitmentChecked(false);
  };

  /**
   * 管理员新增向导：校验当前步并进入下一步。
   */
  const handleWizardNext = async () => {
    const api = formApiRef.current;
    if (!api) {
      return;
    }
    const values = api.getValues();
    setWizardDraftValues((prev) => ({ ...prev, ...values }));
    if (currentStep === 0) {
      const uid = resolveApplicantUserId(values.user_id);
      if (!uid) {
        showError(t('请选择用户'));
        return;
      }
      if (!(values.supplier_alias || '').trim()) {
        showError(t('请填写供应商别名'));
        return;
      }
      setCurrentStep(1);
      return;
    }
    if (currentStep === 1) {
      try {
        await api.validate(['company_name', 'credit_code', 'legal_representative']);
      } catch {
        return;
      }
      if (!businessLicenseUrl) {
        showError(t('请上传营业执照'));
        return;
      }
      setCurrentStep(2);
      return;
    }
    if (currentStep === 2) {
      try {
        await api.validate(['contact_name', 'contact_mobile', 'contact_wechat']);
      } catch {
        return;
      }
      setCurrentStep(3);
    }
  };

  /**
   * 管理员新增向导返回上一步并缓存当前数据。
   */
  const handleWizardPrev = () => {
    const values = formApiRef.current?.getValues() || {};
    setWizardDraftValues((prev) => ({ ...prev, ...values }));
    setCurrentStep((prev) => Math.max(prev - 1, 0));
  };

  /**
   * 提交表单：新增走代提交自动通过 + 技术能力；编辑走 PUT 主体与能力。
   * @param {object} values 表单值
   */
  const submit = async (values) => {
    const mergedValues = isCreateWizard ? { ...wizardDraftValues, ...(values || {}) } : values;
    if (!businessLicenseUrl) {
      showError(t('请上传营业执照'));
      return;
    }

    if (!supplier) {
      const uid = resolveApplicantUserId(mergedValues.user_id);
      if (!uid) {
        showError(t('请选择用户'));
        return;
      }
    }
    if (!mergedValues.truth_commitment_confirmed) {
      showError(t('请先勾选信息真实性承诺'));
      return;
    }

    setLoading(true);
    try {
      const fileInfo = fileList.length > 0 ? fileList[0] : null;
      const businessLicenseFile = fileInfo ? JSON.stringify({
        uid: fileInfo.uid,
        name: fileInfo.name,
        status: fileInfo.status,
        size: fileInfo.size,
        url: businessLicenseUrl,
      }) : '';

      const basePayload = {
        business_license_url: businessLicenseUrl,
        business_license_file: businessLicenseFile,
        company_name: mergedValues.company_name || '',
        company_size: mergedValues.company_size || '',
        contact_mobile: mergedValues.contact_mobile || '',
        contact_name: mergedValues.contact_name || '',
        contact_wechat: mergedValues.contact_wechat || '',
        credit_code: mergedValues.credit_code || '',
        legal_representative: mergedValues.legal_representative || '',
      };

      if (isAdmin()) {
        basePayload.supplier_alias = (mergedValues.supplier_alias || '').trim();
      }

      if (isAdmin() && !basePayload.supplier_alias) {
        showError(t('请填写供应商别名'));
        setLoading(false);
        return;
      }

      let res;
      if (supplier) {
        const apiPath = isAdmin()
          ? `/api/user/supplier/application/${supplier.id}`
          : '/api/user/supplier/application/self';
        res = await API.put(apiPath, basePayload, { skipErrorHandler: true });
      } else {
        const uid = resolveApplicantUserId(mergedValues.user_id);
        const createPayload = {
          ...basePayload,
          applicant_user_id: uid,
        };
        res = await API.post('/api/user/supplier/application', createPayload, { skipErrorHandler: true });
      }
      
      const { success, message, data } = res.data;
      if (success) {
        const capabilityPayload = buildCapabilityPayload(mergedValues);
        const appId = supplier?.id ?? data?.id;
        if (!appId) {
          showError(t('未获取到申请ID，请重试'));
          setLoading(false);
          return;
        }
        const capabilityRes = await API.put(`/api/user/supplier/application/${appId}/capability`, capabilityPayload, { skipErrorHandler: true });
        if (!capabilityRes.data.success) {
          showError(t(capabilityRes.data.message || '技术能力信息保存失败'));
          setLoading(false);
          return;
        }
        showSuccess(supplier ? t('修改成功') : t('添加成功'));
        handleClose();
        formApiRef.current?.reset();
        setFileList([]);
        setBusinessLicenseUrl('');
        setUserOptions([]);
        setSupplierData(null);
        setCurrentStep(0);
        setWizardDraftValues({});
        setCommitmentChecked(false);
        if (onSuccess) {
          onSuccess();
        }
      } else {
        showError(t(message));
      }
    } catch (error) {
      if (error?.response?.status === 403) {
        showError(t('请先申请供应商资质'));
      } else {
        showError(error.response?.data?.message || t('操作失败'));
      }
    }
    setLoading(false);
  };

  /**
   * 营业执照上传逻辑。
   */
  const customRequest = async ({ file, onSuccess, onError }) => {
    const fileInstance = file.fileInstance;
    const isImage = fileInstance.type === 'image/jpeg' || fileInstance.type === 'image/png';
    const isLt5M = fileInstance.size / 1024 / 1024 < 5;

    if (!isImage) {
      showError(t('只支持 JPG/PNG 格式的图片'));
      onError();
      return;
    }

    if (!isLt5M) {
      showError(t('图片大小不能超过 5MB'));
      onError();
      return;
    }

    setFileList([{
      uid: fileInstance.uid,
      name: fileInstance.name,
      status: 'uploading',
      size: fileInstance.size,
    }]);

    try {
      const formData = new FormData();
      formData.append('file', fileInstance);

      const res = await API.post('/api/oss/upload', formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });

      const { success, data, message } = res.data;
      if (success && data?.url) {
        setBusinessLicenseUrl(data.url);
        setFileList([{
          uid: fileInstance.uid,
          name: fileInstance.name,
          status: 'success',
          size: fileInstance.size,
          url: data.url,
        }]);
        onSuccess();
      } else {
        showError(message || t('上传失败'));
        setFileList([]);
        onError();
      }
    } catch (error) {
      showError(error.response?.data?.message || t('上传失败'));
      setFileList([]);
      onError();
    }
  };

  /**
   * 渲染企业主体区块（供应商别名可在编辑页展示，向导第二步不重复别名字段）。
   * @param {object} opts
   * @param {boolean} [opts.showAdminAlias=true] 是否展示供应商别名字段
   */
  const renderEnterpriseSection = ({ showAdminAlias = true } = {}) => (
    <>
      <Divider margin='12px'>
        <Text strong style={{ fontSize: '16px' }}>
          {t('企业主体信息')}
        </Text>
      </Divider>

      <Row gutter={12}>
        {isAdmin() && showAdminAlias && (
          <Col span={24}>
            <Form.Input
              field='supplier_alias'
              label={<Text strong>{t('供应商别名')}</Text>}
              placeholder={t('管理员填写，且全局唯一')}
              rules={[{ required: true, message: t('请输入供应商别名') }]}
              showClear
              maxLength={128}
              extraText={t('供应商别名只能由管理员填写和修改')}
            />
          </Col>
        )}
        <Col span={24}>
          <Form.Input
            field='company_name'
            label={<Text strong>{t('企业/主体名称')}</Text>}
            placeholder={t('填写与营业执照完全一致的全称')}
            rules={[{ required: true, message: t('请输入企业/主体名称') }]}
            showClear
            extraText={t('需与上传执照 100% 匹配')}
          />
        </Col>
        <Col span={24}>
          <Form.Input
            field='credit_code'
            label={<Text strong>{t('统一社会信用代码')}</Text>}
            placeholder={t('填写营业执照上的 18 位代码')}
            rules={[
              { required: true, message: t('请输入统一社会信用代码') },
              { 
                pattern: /^[0-9A-HJ-NPQRTUWXY]{2}\d{6}[0-9A-HJ-NPQRTUWXY]{10}$/,
                message: t('请输入有效的 18 位统一社会信用代码')
              }
            ]}
            showClear
            extraText={t('入驻后不可修改，务必核对准确')}
          />
        </Col>
        <Col span={24}>
          <Form.Upload
            field='license_file'
            label={<Text strong>{t('营业执照')}<Text type='danger'>*</Text></Text>}
            action=''
            accept='.jpg,.jpeg,.png'
            limit={1}
            fileList={fileList}
            onChange={({ fileList: fl }) => setFileList(fl)}
            customRequest={customRequest}
            onRemove={() => {
              setFileList([]);
              setBusinessLicenseUrl('');
            }}
            extraText={t('支持 jpg/png，大小≤5M，信息完整无遮挡')}
          >
            <Button icon={<IconUpload />} theme="light">
              {t('上传文件')}
            </Button>
          </Form.Upload>
        </Col>
        <Col span={24}>
          <Form.Input
            field='legal_representative'
            label={<Text strong>{t('法人/经营者姓名')}</Text>}
            placeholder={t('填写营业执照上的法定代表人/经营者姓名')}
            rules={[{ required: true, message: t('请输入法人/经营者姓名') }]}
            showClear
            extraText={t('与执照信息一致')}
          />
        </Col>
        <Col span={24}>
          <Form.Select
            field='company_size'
            label={<Text strong>{t('企业规模')}</Text>}
            placeholder={t('请选择企业规模')}
            optionList={[
              { label: t('10人以下'), value: '10人以下' },
              { label: t('10-50人'), value: '10-50人' },
              { label: t('50-200人'), value: '50-200人' },
              { label: t('200人以上'), value: '200人以上' },
            ]}
            showClear
          />
        </Col>
      </Row>
    </>
  );

  /**
   * 渲染对接人信息区块。
   */
  const renderContactSection = () => (
    <>
      <Divider margin='20px 12px'>
        <Text strong style={{ fontSize: '16px' }}>
          {t('对接人信息')}
        </Text>
      </Divider>

      <Row gutter={12}>
        <Col span={24}>
          <Form.Input
            field='contact_name'
            label={<Text strong>{t('对接人姓名')}</Text>}
            placeholder={t('填写实际对接的负责人姓名')}
            rules={[{ required: true, message: t('请输入对接人姓名') }]}
            showClear
            extraText={t('平台日常沟通、问题对接')}
          />
        </Col>
        <Col span={24}>
          <Form.Input
            field='contact_mobile'
            label={<Text strong>{t('对接人手机号')}</Text>}
            placeholder={t('填写实名手机号')}
            rules={[
              { required: true, message: t('请输入对接人手机号') },
              { 
                pattern: /^1[3-9]\d{9}$/,
                message: t('请输入有效的手机号')
              }
            ]}
            showClear
            extraText={t('用于紧急联系')}
          />
        </Col>
        <Col span={24}>
          <Form.Input
            field='contact_wechat'
            label={<Text strong>{t('对接人微信/企业微信')}</Text>}
            placeholder={t('填写微信号/企业微信 ID')}
            rules={[{ required: true, message: t('请输入对接人微信/企业微信') }]}
            showClear
            extraText={t('用于拉取服务商专属沟通群')}
          />
        </Col>
      </Row>
    </>
  );

  return (
    <Modal
      title={supplier ? t('修改供应商') : t('新增供应商')}
      visible={visible}
      onOk={isCreateWizard ? null : () => formApiRef.current?.submitForm()}
      onCancel={handleCancel}
      confirmLoading={loading || fetchingData}
      size={isMobile ? 'full-width' : 'large'}
      style={{ maxWidth: isMobile ? '100%' : '800px' }}
      footer={
        isCreateWizard ? (
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8 }}>
            <Button onClick={handleCancel}>{t('取消')}</Button>
            <div style={{ display: 'flex', gap: 8 }}>
              {currentStep > 0 && (
                <Button onClick={handleWizardPrev}>
                  {t('上一步')}
                </Button>
              )}
              {currentStep < 3 ? (
                <Button type='primary' theme='solid' onClick={handleWizardNext}>
                  {t('下一步')}
                </Button>
              ) : (
                <Button
                  type='primary'
                  theme='solid'
                  disabled={!commitmentChecked}
                  loading={loading || fetchingData}
                  onClick={() => formApiRef.current?.submitForm()}
                >
                  {t('提交')}
                </Button>
              )}
            </div>
          </div>
        ) : undefined
      }
    >
      {isCreateWizard && (
        <div style={{ marginBottom: 16 }}>
          <Progress percent={Math.round(((currentStep + 1) / 4) * 100)} showInfo={false} stroke={'var(--semi-color-primary)'} />
          <div style={{ marginTop: 8 }}>
            <Steps type='basic' current={currentStep} size='small'>
              <Steps.Step title={t('关联用户与别名')} />
              <Steps.Step title={t('企业主体')} />
              <Steps.Step title={t('对接人信息')} />
              <Steps.Step title={t('技术能力')} />
            </Steps>
          </div>
        </div>
      )}
      <Form
        initValues={supplier ? {} : getCreateFormInitValues()}
        getFormApi={(api) => (formApiRef.current = api)}
        onSubmit={submit}
      >
        {isCreateWizard && currentStep === 0 && (
          <Row gutter={12}>
            <Col span={24}>
              <Form.Select
                style={{ width: '100%' }}
                filter
                field='user_id'
                label={<Text strong>{t('选择用户')}<Text type='danger'>*</Text></Text>}
                placeholder={t('输入用户名或ID搜索')}
                remote
                onSearch={searchUsers}
                optionList={userOptions}
                loading={loadingUsers}
                showClear
                extraText={t('关联平台用户账号')}
                emptyContent={null}
                onChangeWithObject
              />
            </Col>
            <Col span={24}>
              <Form.Input
                field='supplier_alias'
                label={<Text strong>{t('供应商别名')}</Text>}
                placeholder={t('管理员填写，且全局唯一')}
                rules={[{ required: true, message: t('请输入供应商别名') }]}
                showClear
                maxLength={128}
                extraText={t('提交后将自动审核通过并绑定该别名')}
              />
            </Col>
          </Row>
        )}

        {isCreateWizard && currentStep === 1 && renderEnterpriseSection({ showAdminAlias: false })}

        {isCreateWizard && currentStep === 2 && renderContactSection()}

        {isCreateWizard && currentStep === 3 && (
          <SupplierCapabilityFormFields
            t={t}
            onCommitmentChange={(checked) => setCommitmentChecked(checked)}
          />
        )}

        {supplier && supplierData && (
          <>
            <div style={{ marginBottom: '12px' }}>
              <div style={{ marginBottom: '4px' }}>
                <Text strong>{t('关联用户')}</Text>
              </div>
              <Input
                value={supplierData.applicant_username || `用户 ${supplierData.user_id}`}
                disabled
                style={{ width: '100%' }}
              />
              <Text type="tertiary" size="small" style={{ marginTop: '4px', display: 'block' }}>
                {t('修改时不能更改用户')}
              </Text>
            </div>

            {renderEnterpriseSection({ showAdminAlias: true })}
            {renderContactSection()}
            <SupplierCapabilityFormFields
              t={t}
              onCommitmentChange={(checked) => setCommitmentChecked(checked)}
            />
          </>
        )}
      </Form>
    </Modal>
  );
};

export default SupplierEditModal;
