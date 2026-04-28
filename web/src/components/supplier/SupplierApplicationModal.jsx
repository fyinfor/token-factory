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
import React, { useState, useRef, useEffect, useMemo } from 'react';
import {
  Modal,
  Form,
  Row,
  Col,
  Button,
  Typography,
  Upload,
  Divider,
  Progress,
  Steps,
} from '@douyinfe/semi-ui';
import { IconUpload } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import SupplierCapabilityFormFields from './SupplierCapabilityFormFields';

const { Text } = Typography;

const SupplierApplicationModal = ({ visible, handleClose }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);
  const isMobile = useIsMobile();

  const [fileList, setFileList] = useState([]);
  const [businessLicenseUrl, setBusinessLicenseUrl] = useState('');
  const [logoFileList, setLogoFileList] = useState([]);
  const [companyLogoUrl, setCompanyLogoUrl] = useState('');
  const [fetchingData, setFetchingData] = useState(false);
  const [hasExistingApplication, setHasExistingApplication] = useState(false);
  const [applicationId, setApplicationId] = useState(null);
  const [capabilityData, setCapabilityData] = useState(null);
  const [currentStep, setCurrentStep] = useState(0);
  const [wizardDraftValues, setWizardDraftValues] = useState({});
  const [commitmentChecked, setCommitmentChecked] = useState(false);

  useEffect(() => {
    if (visible) {
      fetchApplicationData();
    } else {
      formApiRef.current?.reset();
      setFileList([]);
      setBusinessLicenseUrl('');
      setLogoFileList([]);
      setCompanyLogoUrl('');
      setHasExistingApplication(false);
      setApplicationId(null);
      setCapabilityData(null);
      setCurrentStep(0);
      setWizardDraftValues({});
      setCommitmentChecked(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible]);

  /**
   * 步骤切换后把缓存值回填到当前挂载字段，避免“上一步”时数据丢失。
   */
  useEffect(() => {
    if (!visible || !formApiRef.current) {
      return;
    }
    if (Object.keys(wizardDraftValues).length === 0) {
      return;
    }
    formApiRef.current.setValues(wizardDraftValues);
    setCommitmentChecked(!!wizardDraftValues.truth_commitment_confirmed);
  }, [currentStep, visible, wizardDraftValues]);

  const fetchApplicationData = async () => {
    setFetchingData(true);
    try {
      const res = await API.get('/api/user/supplier/application/self', {
        skipErrorHandler: true,
      });
      if (res.data.success && res.data.data) {
        const data = res.data.data;
        setHasExistingApplication(true);
        setApplicationId(data.id);
        // 接口 /self 已附带 supplier_capability；单独拉取失败时仍可用于回显。
        let fetchedCapabilityData = data.supplier_capability || null;
        try {
          const capabilityRes = await API.get(
            `/api/user/supplier/application/${data.id}/capability`,
            { skipErrorHandler: true },
          );
          if (capabilityRes.data.success && capabilityRes.data.data) {
            fetchedCapabilityData = capabilityRes.data.data;
          }
        } catch (capabilityError) {
          console.error('Failed to fetch capability data:', capabilityError);
        }
        setCapabilityData(fetchedCapabilityData);
        formApiRef.current?.setValues({
          company_name: data.company_name || '',
          credit_code: data.credit_code || '',
          legal_representative: data.legal_representative || '',
          company_size: data.company_size || '',
          contact_name: data.contact_name || '',
          contact_mobile: data.contact_mobile || '',
          contact_wechat: data.contact_wechat || '',
          core_service_types: fetchedCapabilityData?.core_service_types || [],
          supported_models: fetchedCapabilityData?.supported_models || [],
          supported_model_notes:
            fetchedCapabilityData?.supported_model_notes || '',
          supported_api_endpoints:
            fetchedCapabilityData?.supported_api_endpoints || [],
          supported_api_endpoint_extra:
            fetchedCapabilityData?.supported_api_endpoint_extra || '',
          supported_params: fetchedCapabilityData?.supported_params || [],
          supported_params_extra:
            fetchedCapabilityData?.supported_params_extra || '',
          streaming_supported: fetchedCapabilityData?.streaming_supported
            ? 'yes'
            : 'no',
          streaming_notes: fetchedCapabilityData?.streaming_notes || '',
          structured_output_supported:
            fetchedCapabilityData?.structured_output_supported ? 'yes' : 'no',
          structured_output_notes:
            fetchedCapabilityData?.structured_output_notes || '',
          multimodal_types: fetchedCapabilityData?.multimodal_types || [],
          multimodal_extra: fetchedCapabilityData?.multimodal_extra || '',
          pricing_modes: fetchedCapabilityData?.pricing_modes || [],
          reference_input_price:
            fetchedCapabilityData?.reference_input_price || '',
          reference_output_price:
            fetchedCapabilityData?.reference_output_price || '',
          failure_billing_mode:
            fetchedCapabilityData?.failure_billing_mode || 'no_bill',
          failure_billing_notes:
            fetchedCapabilityData?.failure_billing_notes || '',
          api_base_urls: fetchedCapabilityData?.api_base_urls || [],
          openai_compatible: fetchedCapabilityData?.openai_compatible
            ? 'yes'
            : 'no',
          truth_commitment_confirmed:
            fetchedCapabilityData?.truth_commitment_confirmed || false,
        });
        setWizardDraftValues({
          company_name: data.company_name || '',
          credit_code: data.credit_code || '',
          legal_representative: data.legal_representative || '',
          company_size: data.company_size || '',
          contact_name: data.contact_name || '',
          contact_mobile: data.contact_mobile || '',
          contact_wechat: data.contact_wechat || '',
          core_service_types: fetchedCapabilityData?.core_service_types || [],
          supported_models: fetchedCapabilityData?.supported_models || [],
          supported_model_notes:
            fetchedCapabilityData?.supported_model_notes || '',
          supported_api_endpoints:
            fetchedCapabilityData?.supported_api_endpoints || [],
          supported_api_endpoint_extra:
            fetchedCapabilityData?.supported_api_endpoint_extra || '',
          supported_params: fetchedCapabilityData?.supported_params || [],
          supported_params_extra:
            fetchedCapabilityData?.supported_params_extra || '',
          streaming_supported: fetchedCapabilityData?.streaming_supported
            ? 'yes'
            : 'no',
          streaming_notes: fetchedCapabilityData?.streaming_notes || '',
          structured_output_supported:
            fetchedCapabilityData?.structured_output_supported ? 'yes' : 'no',
          structured_output_notes:
            fetchedCapabilityData?.structured_output_notes || '',
          multimodal_types: fetchedCapabilityData?.multimodal_types || [],
          multimodal_extra: fetchedCapabilityData?.multimodal_extra || '',
          pricing_modes: fetchedCapabilityData?.pricing_modes || [],
          reference_input_price:
            fetchedCapabilityData?.reference_input_price || '',
          reference_output_price:
            fetchedCapabilityData?.reference_output_price || '',
          failure_billing_mode:
            fetchedCapabilityData?.failure_billing_mode || 'no_bill',
          failure_billing_notes:
            fetchedCapabilityData?.failure_billing_notes || '',
          api_base_urls: fetchedCapabilityData?.api_base_urls || [],
          openai_compatible: fetchedCapabilityData?.openai_compatible
            ? 'yes'
            : 'no',
          truth_commitment_confirmed:
            fetchedCapabilityData?.truth_commitment_confirmed || false,
        });
        setCommitmentChecked(
          !!fetchedCapabilityData?.truth_commitment_confirmed,
        );

        // 等 Form 挂载完成后再写一次，避免 ref 未就绪时 setValues 被跳过。
        requestAnimationFrame(() => {
          formApiRef.current?.setValues({
            company_name: data.company_name || '',
            credit_code: data.credit_code || '',
            legal_representative: data.legal_representative || '',
            company_size: data.company_size || '',
            contact_name: data.contact_name || '',
            contact_mobile: data.contact_mobile || '',
            contact_wechat: data.contact_wechat || '',
            core_service_types: fetchedCapabilityData?.core_service_types || [],
            supported_models: fetchedCapabilityData?.supported_models || [],
            supported_model_notes:
              fetchedCapabilityData?.supported_model_notes || '',
            supported_api_endpoints:
              fetchedCapabilityData?.supported_api_endpoints || [],
            supported_api_endpoint_extra:
              fetchedCapabilityData?.supported_api_endpoint_extra || '',
            supported_params: fetchedCapabilityData?.supported_params || [],
            supported_params_extra:
              fetchedCapabilityData?.supported_params_extra || '',
            streaming_supported: fetchedCapabilityData?.streaming_supported
              ? 'yes'
              : 'no',
            streaming_notes: fetchedCapabilityData?.streaming_notes || '',
            structured_output_supported:
              fetchedCapabilityData?.structured_output_supported ? 'yes' : 'no',
            structured_output_notes:
              fetchedCapabilityData?.structured_output_notes || '',
            multimodal_types: fetchedCapabilityData?.multimodal_types || [],
            multimodal_extra: fetchedCapabilityData?.multimodal_extra || '',
            pricing_modes: fetchedCapabilityData?.pricing_modes || [],
            reference_input_price:
              fetchedCapabilityData?.reference_input_price || '',
            reference_output_price:
              fetchedCapabilityData?.reference_output_price || '',
            failure_billing_mode:
              fetchedCapabilityData?.failure_billing_mode || 'no_bill',
            failure_billing_notes:
              fetchedCapabilityData?.failure_billing_notes || '',
            api_base_urls: fetchedCapabilityData?.api_base_urls || [],
            openai_compatible: fetchedCapabilityData?.openai_compatible
              ? 'yes'
              : 'no',
            truth_commitment_confirmed:
              fetchedCapabilityData?.truth_commitment_confirmed || false,
          });
        });

        if (data.business_license_url) {
          setBusinessLicenseUrl(data.business_license_url);
        }
        if (data.company_logo_url) {
          setCompanyLogoUrl(data.company_logo_url);
          setLogoFileList([
            {
              uid: 'existing-logo',
              name: t('已上传的企业Logo'),
              status: 'success',
              url: data.company_logo_url,
            },
          ]);
        }

        if (data.business_license_file) {
          try {
            const fileInfo = JSON.parse(data.business_license_file);
            console.log('fileInfo', fileInfo);
            setFileList([fileInfo]);
          } catch (e) {
            console.error('Failed to parse business_license_file:', e);
            if (data.business_license_url) {
              setFileList([
                {
                  uid: 'existing',
                  name: t('已上传的营业执照'),
                  status: 'success',
                  url: data.business_license_url,
                },
              ]);
            }
          }
        }
      } else {
        setHasExistingApplication(false);
        setApplicationId(null);
      }
    } catch (error) {
      console.error('Failed to fetch application data:', error);
      setHasExistingApplication(false);
      setApplicationId(null);
      setCapabilityData(null);
      setCurrentStep(0);
      setWizardDraftValues({});
      setCommitmentChecked(false);
    } finally {
      setFetchingData(false);
    }
  };

  /**
   * defaultFormInitValues 多步表单初始空值；用 useMemo 固定引用，避免每次渲染 initValues 变化导致表单被重置。
   */
  const defaultFormInitValues = useMemo(
    () => ({
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
    }),
    [],
  );

  const handleCancel = () => {
    handleClose();
    formApiRef.current?.reset();
    setFileList([]);
    setBusinessLicenseUrl('');
    setLogoFileList([]);
    setCompanyLogoUrl('');
    setHasExistingApplication(false);
    setApplicationId(null);
    setCapabilityData(null);
    setCurrentStep(0);
    setWizardDraftValues({});
    setCommitmentChecked(false);
  };

  /**
   * handleStepNext 校验当前步骤后进入下一步，并缓存步骤数据防止卸载丢失。
   */
  const handleStepNext = async () => {
    const api = formApiRef.current;
    if (!api) {
      return;
    }
    const currentValues = api.getValues();
    setWizardDraftValues((prev) => ({ ...prev, ...currentValues }));
    if (currentStep === 0) {
      try {
        await api.validate([
          'company_name',
          'credit_code',
          'legal_representative',
        ]);
      } catch {
        return;
      }
      if (!businessLicenseUrl) {
        showError(t('请上传营业执照'));
        return;
      }
      if (!companyLogoUrl) {
        showError(t('请上传企业Logo'));
        return;
      }
      setCurrentStep(1);
      return;
    }
    if (currentStep === 1) {
      try {
        await api.validate([
          'contact_name',
          'contact_mobile',
          'contact_wechat',
        ]);
      } catch {
        return;
      }
      setCurrentStep(2);
    }
  };

  /**
   * handleStepPrev 返回上一步并缓存当前步骤数据。
   */
  const handleStepPrev = () => {
    const currentValues = formApiRef.current?.getValues() || {};
    setWizardDraftValues((prev) => ({ ...prev, ...currentValues }));
    setCurrentStep((prev) => Math.max(prev - 1, 0));
  };

  const submit = async (values) => {
    const mergedValues = { ...wizardDraftValues, ...(values || {}) };
    if (!businessLicenseUrl) {
      showError(t('请上传营业执照'));
      return;
    }
    if (!companyLogoUrl) {
      showError(t('请上传企业Logo'));
      return;
    }
    if (!mergedValues.truth_commitment_confirmed) {
      showError(t('请先勾选信息真实性承诺'));
      return;
    }

    setLoading(true);
    try {
      const fileInfo = fileList.length > 0 ? fileList[0] : null;
      const businessLicenseFile = fileInfo
        ? JSON.stringify({
            uid: fileInfo.uid,
            name: fileInfo.name,
            status: fileInfo.status,
            size: fileInfo.size,
            url: businessLicenseUrl,
          })
        : '';

      const payload = {
        business_license_url: businessLicenseUrl,
        business_license_file: businessLicenseFile,
        company_logo_url: companyLogoUrl,
        company_name: mergedValues.company_name || '',
        company_size: mergedValues.company_size || '',
        contact_mobile: mergedValues.contact_mobile || '',
        contact_name: mergedValues.contact_name || '',
        contact_wechat: mergedValues.contact_wechat || '',
        credit_code: mergedValues.credit_code || '',
        legal_representative: mergedValues.legal_representative || '',
      };

      if (hasExistingApplication && applicationId) {
        payload.id = applicationId;
      }

      const res = hasExistingApplication
        ? await API.put('/api/user/supplier/application/self', payload, {
            skipErrorHandler: true,
          })
        : await API.post('/api/user/supplier/application', payload, {
            skipErrorHandler: true,
          });

      const { success, message, data } = res.data;
      if (success) {
        const targetApplicationID = hasExistingApplication
          ? applicationId
          : data?.id;
        const capabilityPayload = {
          core_service_types: mergedValues.core_service_types || [],
          supported_models: mergedValues.supported_models || [],
          supported_model_notes: mergedValues.supported_model_notes || '',
          supported_api_endpoints: mergedValues.supported_api_endpoints || [],
          supported_api_endpoint_extra:
            mergedValues.supported_api_endpoint_extra || '',
          supported_params: mergedValues.supported_params || [],
          supported_params_extra: mergedValues.supported_params_extra || '',
          streaming_supported: mergedValues.streaming_supported === 'yes',
          streaming_notes: mergedValues.streaming_notes || '',
          structured_output_supported:
            mergedValues.structured_output_supported === 'yes',
          structured_output_notes: mergedValues.structured_output_notes || '',
          multimodal_types: mergedValues.multimodal_types || [],
          multimodal_extra: mergedValues.multimodal_extra || '',
          pricing_modes: mergedValues.pricing_modes || [],
          reference_input_price: mergedValues.reference_input_price || '',
          reference_output_price: mergedValues.reference_output_price || '',
          failure_billing_mode: mergedValues.failure_billing_mode || '',
          failure_billing_notes: mergedValues.failure_billing_notes || '',
          api_base_urls: mergedValues.api_base_urls || [],
          openai_compatible: mergedValues.openai_compatible === 'yes',
          truth_commitment_confirmed: !!mergedValues.truth_commitment_confirmed,
        };
        if (!targetApplicationID) {
          showError(t('未获取到申请ID，请重试'));
          setLoading(false);
          return;
        }
        const capabilityRes = await API.put(
          `/api/user/supplier/application/${targetApplicationID}/capability`,
          capabilityPayload,
          { skipErrorHandler: true },
        );
        if (!capabilityRes.data.success) {
          showError(t(capabilityRes.data.message || '技术能力信息保存失败'));
          setLoading(false);
          return;
        }
        showSuccess(t('申请已提交，请等待审核'));
        handleClose();
        formApiRef.current?.reset();
        setFileList([]);
        setBusinessLicenseUrl('');
        setLogoFileList([]);
        setCompanyLogoUrl('');
        setHasExistingApplication(false);
        setApplicationId(null);
        setCapabilityData(null);
        setWizardDraftValues({});
        setCommitmentChecked(false);
      } else {
        showError(t(message));
      }
    } catch (error) {
      if (error?.response?.status === 403) {
        showError(t('请先申请供应商资质'));
      } else {
        showError(error.response?.data?.message || t('提交失败'));
      }
    }
    setLoading(false);
  };

  const customRequest = async ({ file, onSuccess, onError }) => {
    const fileInstance = file.fileInstance;
    const isImage =
      fileInstance.type === 'image/jpeg' || fileInstance.type === 'image/png';
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

    setFileList([
      {
        uid: fileInstance.uid,
        name: fileInstance.name,
        status: 'uploading',
        size: fileInstance.size,
      },
    ]);

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
        setFileList([
          {
            uid: fileInstance.uid,
            name: fileInstance.name,
            status: 'success',
            size: fileInstance.size,
            url: data.url,
          },
        ]);
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
   * 上传企业 Logo 图片并回填 URL。
   */
  const customLogoRequest = async ({ file, onSuccess, onError }) => {
    const fileInstance = file.fileInstance;
    const isImage =
      fileInstance.type === 'image/jpeg' || fileInstance.type === 'image/png';
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

    setLogoFileList([
      {
        uid: fileInstance.uid,
        name: fileInstance.name,
        status: 'uploading',
        size: fileInstance.size,
      },
    ]);

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
        setCompanyLogoUrl(data.url);
        setLogoFileList([
          {
            uid: fileInstance.uid,
            name: fileInstance.name,
            status: 'success',
            size: fileInstance.size,
            url: data.url,
          },
        ]);
        onSuccess();
      } else {
        showError(message || t('上传失败'));
        setLogoFileList([]);
        onError();
      }
    } catch (error) {
      showError(error.response?.data?.message || t('上传失败'));
      setLogoFileList([]);
      onError();
    }
  };

  return (
    <Modal
      title={t('申请成为供应商')}
      visible={visible}
      onOk={null}
      onCancel={handleCancel}
      confirmLoading={loading || fetchingData}
      size={isMobile ? 'full-width' : 'large'}
      style={{ maxWidth: isMobile ? '100%' : '800px' }}
      footer={
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <Button onClick={handleCancel}>{t('取消')}</Button>
          <div style={{ display: 'flex', gap: 8 }}>
            {currentStep > 0 && (
              <Button onClick={handleStepPrev}>{t('上一步')}</Button>
            )}
            {currentStep < 2 ? (
              <Button type='primary' theme='solid' onClick={handleStepNext}>
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
                {t('提交申请')}
              </Button>
            )}
          </div>
        </div>
      }
    >
      <div style={{ marginBottom: 16 }}>
        <Progress
          percent={Math.round(((currentStep + 1) / 3) * 100)}
          showInfo={false}
          stroke={'var(--semi-color-primary)'}
        />
        <div style={{ marginTop: 8 }}>
          <Steps type='basic' current={currentStep} size='small'>
            <Steps.Step title={t('企业主体')} />
            <Steps.Step title={t('对接人信息')} />
            <Steps.Step title={t('技术能力')} />
          </Steps>
        </div>
      </div>
      <Form
        initValues={defaultFormInitValues}
        getFormApi={(api) => (formApiRef.current = api)}
        onSubmit={submit}
      >
        {/*
          三步同时挂载、用 display 切换，保证 Semi Form 注册全部 field；
          若按 currentStep 条件卸载，未挂载字段无法接收 setValues，编辑申请时对接人/技术能力不回显。
        */}
        <div style={{ display: currentStep === 0 ? 'block' : 'none' }}>
          <Divider margin='12px'>
            <Text strong style={{ fontSize: '16px' }}>
              {t('企业主体信息')}
            </Text>
          </Divider>

          <Row gutter={12}>
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
                    pattern:
                      /^[0-9A-HJ-NPQRTUWXY]{2}\d{6}[0-9A-HJ-NPQRTUWXY]{10}$/,
                    message: t('请输入有效的 18 位统一社会信用代码'),
                  },
                ]}
                showClear
                extraText={t('入驻后不可修改，务必核对准确')}
              />
            </Col>
            <Col span={24}>
              <Form.Upload
                field='license_file'
                label={
                  <Text strong>
                    {t('营业执照')}
                    <Text type='danger'>*</Text>
                  </Text>
                }
                action=''
                accept='.jpg,.jpeg,.png'
                limit={1}
                fileList={fileList}
                onChange={({ fileList }) => setFileList(fileList)}
                customRequest={customRequest}
                onRemove={() => {
                  setFileList([]);
                  setBusinessLicenseUrl('');
                }}
                extraText={t('支持 jpg/png，大小≤5M，信息完整无遮挡')}
              >
                <Button icon={<IconUpload />} theme='light'>
                  {t('上传文件')}
                </Button>
              </Form.Upload>
            </Col>
            <Col span={24}>
              <Form.Upload
                field='company_logo_file'
                label={
                  <Text strong>
                    {t('企业Logo')}
                    <Text type='danger'>*</Text>
                  </Text>
                }
                action=''
                accept='.jpg,.jpeg,.png'
                limit={1}
                fileList={logoFileList}
                onChange={({ fileList }) => setLogoFileList(fileList)}
                customRequest={customLogoRequest}
                onRemove={() => {
                  setLogoFileList([]);
                  setCompanyLogoUrl('');
                }}
                extraText={t('建议上传清晰方形Logo，支持 jpg/png，大小≤5M')}
              >
                <Button icon={<IconUpload />} theme='light'>
                  {t('上传文件')}
                </Button>
              </Form.Upload>
            </Col>
            <Col span={24}>
              <Form.Input
                field='legal_representative'
                label={<Text strong>{t('法人/经营者姓名')}</Text>}
                placeholder={t('填写营业执照上的法定代表人/经营者姓名')}
                rules={[
                  { required: true, message: t('请输入法人/经营者姓名') },
                ]}
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
                size='large'
              />
            </Col>
          </Row>
        </div>

        <div style={{ display: currentStep === 1 ? 'block' : 'none' }}>
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
                    message: t('请输入有效的手机号'),
                  },
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
                rules={[
                  { required: true, message: t('请输入对接人微信/企业微信') },
                ]}
                showClear
                extraText={t('用于拉取服务商专属沟通群')}
              />
            </Col>
          </Row>
        </div>

        <div style={{ display: currentStep === 2 ? 'block' : 'none' }}>
          <SupplierCapabilityFormFields
            t={t}
            onCommitmentChange={(checked) => setCommitmentChecked(checked)}
          />
        </div>
      </Form>
    </Modal>
  );
};

export default SupplierApplicationModal;
