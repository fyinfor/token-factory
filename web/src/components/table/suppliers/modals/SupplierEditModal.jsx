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
import { Modal, Form, Row, Col, Button, Typography, Upload, Divider, Input } from '@douyinfe/semi-ui';
import { IconUpload } from '@douyinfe/semi-icons';
import { debounce } from 'lodash-es';
import { API, showError, showSuccess, isAdmin } from '../../../../helpers';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const { Text } = Typography;

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
  const [selectedUser, setSelectedUser] = useState(null);
  const [supplierData, setSupplierData] = useState(null);

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
      const res = await API.get(apiPath);
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
      formApiRef.current?.reset();
      setFileList([]);
      setBusinessLicenseUrl('');
      setSelectedUser(null);
      loadInitialUsers();
    }
  }, [visible, supplier, fetchSupplierData, loadInitialUsers]);

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
      });
      
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
        setSelectedUser(supplierData.user_id);
        setUserOptions([{
          value: supplierData.user_id,
          label: supplierData.applicant_username || `用户 ${supplierData.user_id}`,
          username: supplierData.applicant_username || `用户${supplierData.user_id}`,
          display_name: supplierData.display_name,
        }]);
      }
    }
  }, [supplierData, t]);

  const handleCancel = () => {
    handleClose();
    formApiRef.current?.reset();
    setFileList([]);
    setBusinessLicenseUrl('');
    setSelectedUser(null);
    setUserOptions([]);
    setSupplierData(null);
  };

  const submit = async (values) => {
    if (!businessLicenseUrl) {
      showError(t('请上传营业执照'));
      return;
    }

    if (!supplier && !values.user_id) {
      showError(t('请选择用户'));
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
      if (isAdmin()) {
        payload.supplier_alias = (values.supplier_alias || '').trim();
      }

      if (isAdmin() && !payload.supplier_alias) {
        showError(t('请填写供应商别名'));
        setLoading(false);
        return;
      }

      if (!supplier) {
        payload.user_id = values.user_id;
      }

      let res;
      if (supplier) {
        const apiPath = isAdmin()
          ? `/api/user/supplier/application/${supplier.id}`
          : '/api/user/supplier/application/self';
        res = await API.put(apiPath, payload);
      } else {
        res = await API.post('/api/user/supplier', payload);
      }
      
      const { success, message } = res.data;
      if (success) {
        showSuccess(supplier ? t('修改成功') : t('添加成功'));
        handleClose();
        formApiRef.current?.reset();
        setFileList([]);
        setBusinessLicenseUrl('');
        setSelectedUser(null);
        setUserOptions([]);
        setSupplierData(null);
        if (onSuccess) {
          onSuccess();
        }
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(error.response?.data?.message || t('操作失败'));
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
      title={supplier ? t('修改供应商') : t('新增供应商')}
      visible={visible}
      onOk={() => formApiRef.current?.submitForm()}
      onCancel={handleCancel}
      confirmLoading={loading || fetchingData}
      size={isMobile ? 'full-width' : 'large'}
      style={{ maxWidth: isMobile ? '100%' : '800px' }}
    >
      <Form
        getFormApi={(api) => (formApiRef.current = api)}
        onSubmit={submit}
      >
        {!supplier && (
          <Form.Select
            style={{width: '100%'}}
            filter
            field='user_id'
            label={<Text strong>{t('选择用户')}<Text type='danger'>*</Text></Text>}
            placeholder={t('输入用户名或ID搜索')}
            rules={[{ required: true, message: t('请选择用户') }]}
            remote
            onSearch={searchUsers}
            optionList={userOptions}
            loading={loadingUsers}
            showClear
            extraText={t('关联平台用户账号')}
            emptyContent={null}
            onChangeWithObject
            onChange={(value) => {
              if (value && value.username) {
                setSelectedUser(value);
              }
            }}
          />
        )}

        {supplier && supplierData && (
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
        )}

        <Divider margin='12px'>
          <Text strong style={{ fontSize: '16px' }}>
            {t('企业主体信息')}
          </Text>
        </Divider>

        <Row gutter={12}>
          {isAdmin() && (
            <Col span={24}>
              <Form.Input
                field='supplier_alias'
                label={<Text strong>{t('供应商别名')}<Text type='danger'>*</Text></Text>}
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

export default SupplierEditModal;
