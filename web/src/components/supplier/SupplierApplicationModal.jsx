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
import { Modal, Form, Row, Col, Button, Typography, Upload, Divider } from '@douyinfe/semi-ui';
import { IconUpload } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../helpers';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../hooks/common/useIsMobile';

const { Text } = Typography;

const SupplierApplicationModal = ({ visible, handleClose }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);
  const isMobile = useIsMobile();

  const [fileList, setFileList] = useState([]);
  const [businessLicenseUrl, setBusinessLicenseUrl] = useState('');
  const [fetchingData, setFetchingData] = useState(false);
  const [hasExistingApplication, setHasExistingApplication] = useState(false);
  const [applicationId, setApplicationId] = useState(null);

  useEffect(() => {
    if (visible) {
      fetchApplicationData();
    } else {
      formApiRef.current?.reset();
      setFileList([]);
      setBusinessLicenseUrl('');
      setHasExistingApplication(false);
      setApplicationId(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible]);

  const fetchApplicationData = async () => {
    setFetchingData(true);
    try {
      const res = await API.get('/api/user/supplier/application/self');
      if (res.data.success && res.data.data) {
        const data = res.data.data;
        setHasExistingApplication(true);
        setApplicationId(data.id);
        formApiRef.current?.setValues({
          company_name: data.company_name || '',
          credit_code: data.credit_code || '',
          legal_representative: data.legal_representative || '',
          company_size: data.company_size || '',
          contact_name: data.contact_name || '',
          contact_mobile: data.contact_mobile || '',
          contact_wechat: data.contact_wechat || '',
        });
        
        if (data.business_license_url) {
          setBusinessLicenseUrl(data.business_license_url);
        }

        if (data.business_license_file) {
          try {
            const fileInfo = JSON.parse(data.business_license_file);
            console.log('fileInfo', fileInfo);
            setFileList([fileInfo]);
            
          } catch (e) {
            console.error('Failed to parse business_license_file:', e);
            if (data.business_license_url) {
              setFileList([{
                uid: 'existing',
                name: t('已上传的营业执照'),
                status: 'success',
                url: data.business_license_url,
              }]);
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
    } finally {
      setFetchingData(false);
    }
  };

  const getInitValues = () => ({
    company_name: '',
    credit_code: '',
    legal_representative: '',
    company_size: '',
    contact_name: '',
    contact_mobile: '',
    contact_wechat: '',
  });

  const handleCancel = () => {
    handleClose();
    formApiRef.current?.reset();
    setFileList([]);
    setBusinessLicenseUrl('');
    setHasExistingApplication(false);
    setApplicationId(null);
  };

  const submit = async (values) => {
    if (!businessLicenseUrl) {
      showError(t('请上传营业执照'));
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

      const payload = {
        business_license_url: businessLicenseUrl,
        business_license_file: businessLicenseFile,
        company_name: values.company_name || '',
        company_size: values.company_size || '',
        contact_mobile: values.contact_mobile || '',
        contact_name: values.contact_name || '',
        contact_wechat: values.contact_wechat || '',
        credit_code: values.credit_code || '',
        legal_representative: values.legal_representative || '',
      };

      if (hasExistingApplication && applicationId) {
        payload.id = applicationId;
      }

      const res = hasExistingApplication 
        ? await API.put('/api/user/supplier/application/self', payload)
        : await API.post('/api/user/supplier/application', payload);
      
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('申请已提交，请等待审核'));
        handleClose();
        formApiRef.current?.reset();
        setFileList([]);
        setBusinessLicenseUrl('');
        setHasExistingApplication(false);
        setApplicationId(null);
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(error.response?.data?.message || t('提交失败'));
    }
    setLoading(false);
  };

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

  return (
    <Modal
      title={t('申请成为供应商')}
      visible={visible}
      onOk={() => formApiRef.current?.submitForm()}
      onCancel={handleCancel}
      confirmLoading={loading || fetchingData}
      size={isMobile ? 'full-width' : 'large'}
      style={{ maxWidth: isMobile ? '100%' : '800px' }}
    >
      <Form
        initValues={getInitValues()}
        getFormApi={(api) => (formApiRef.current = api)}
        onSubmit={submit}
      >
        <Divider margin='12px'>
          <Text strong style={{ fontSize: '16px' }}>
            {t('企业主体信息')}
          </Text>
        </Divider>

        <Row gutter={12}>
          <Col span={24}>
            <Form.Input
              field='company_name'
              label={<Text strong>{t('企业/主体名称')}<Text type='danger'>*</Text></Text>}
              placeholder={t('填写与营业执照完全一致的全称')}
              rules={[{ required: true, message: t('请输入企业/主体名称') }]}
              showClear
              extraText={t('需与上传执照 100% 匹配')}
            />
          </Col>
          <Col span={24}>
            <Form.Input
              field='credit_code'
              label={<Text strong>{t('统一社会信用代码')}<Text type='danger'>*</Text></Text>}
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
              onChange={({ fileList }) => setFileList(fileList)}
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
              label={<Text strong>{t('法人/经营者姓名')}<Text type='danger'>*</Text></Text>}
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

        <Divider margin='20px 12px'>
          <Text strong style={{ fontSize: '16px' }}>
            {t('对接人信息')}
          </Text>
        </Divider>

        <Row gutter={12}>
          <Col span={24}>
            <Form.Input
              field='contact_name'
              label={<Text strong>{t('对接人姓名')}<Text type='danger'>*</Text></Text>}
              placeholder={t('填写实际对接的负责人姓名')}
              rules={[{ required: true, message: t('请输入对接人姓名') }]}
              showClear
              extraText={t('平台日常沟通、问题对接')}
            />
          </Col>
          <Col span={24}>
            <Form.Input
              field='contact_mobile'
              label={<Text strong>{t('对接人手机号')}<Text type='danger'>*</Text></Text>}
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
              label={<Text strong>{t('对接人微信/企业微信')}<Text type='danger'>*</Text></Text>}
              placeholder={t('填写微信号/企业微信 ID')}
              rules={[{ required: true, message: t('请输入对接人微信/企业微信') }]}
              showClear
              extraText={t('用于拉取服务商专属沟通群')}
            />
          </Col>
        </Row>
      </Form>
    </Modal>
  );
};

export default SupplierApplicationModal;
