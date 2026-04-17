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
import React, { useState, useEffect } from 'react';
import { Modal, Spin, Image, Typography, Tag, Divider, Row, Col } from '@douyinfe/semi-ui';
import { API, showError } from '../../helpers';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { useActualTheme } from '../../context/Theme';

const { Text, Paragraph } = Typography;

const SupplierDetailModal = ({ visible, handleClose }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [supplierData, setSupplierData] = useState(null);
  const isMobile = useIsMobile();
  const actualTheme = useActualTheme();
  const isDark = actualTheme === 'dark';

  useEffect(() => {
    if (visible) {
      fetchSupplierData();
    }
  }, [visible]);

  const fetchSupplierData = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/user/supplier/application/self');
      if (res.data.success && res.data.data) {
        setSupplierData(res.data.data);
      }
    } catch (error) {
      showError(error.response?.data?.message || t('获取供应商信息失败'));
    } finally {
      setLoading(false);
    }
  };

  const getStatusText = (status) => {
    switch (status) {
      case 0:
        return t('待审核');
      case 1:
        return t('审核通过');
      case 2:
        return t('审核驳回');
      case 3:
        return t('已注销');
      default:
        return '';
    }
  };

  const getStatusTag = (status) => {
    switch (status) {
      case 0:
        return <Tag color='blue'>{getStatusText(status)}</Tag>;
      case 1:
        return <Tag color='green'>{getStatusText(status)}</Tag>;
      case 2:
        return <Tag color='red'>{getStatusText(status)}</Tag>;
      case 3:
        return <Tag color='grey'>{getStatusText(status)}</Tag>;
      default:
        return null;
    }
  };

  const InfoItem = ({ label, value }) => (
    <Col span={24}>
      <div className='mb-4'>
        <Text strong style={{ fontSize: '14px', display: 'block', marginBottom: '8px' }}>
          {label}
        </Text>
        <div style={{ 
          padding: '8px 12px',
          backgroundColor: isDark ? '#2d2e33' : '#f8f8f8',
          borderRadius: '4px',
        }}>
          {value}
        </div>
      </div>
    </Col>
  );

  return (
    <Modal
      title={t('供应商详情')}
      visible={visible}
      onCancel={handleClose}
      footer={null}
      size={isMobile ? 'full-width' : 'large'}
      style={{ maxWidth: isMobile ? '100%' : '800px' }}
    >
      {loading ? (
        <div className='flex items-center justify-center py-8'>
          <Spin size='large' />
        </div>
      ) : (
        <div>
          <Divider margin='12px'>
            <Text strong style={{ fontSize: '16px' }}>
              {t('审核信息')}
            </Text>
          </Divider>

          <Row gutter={12}>
            <InfoItem 
              label={t('审核状态')} 
              value={supplierData?.status !== undefined ? getStatusTag(supplierData.status) : '-'}
            />
            {supplierData?.created_at && (
              <Col span={24}>
                <div className='mb-4'>
                  <Text strong style={{ fontSize: '14px', display: 'block', marginBottom: '8px' }}>
                    {t('申请时间')}
                  </Text>
                  <div style={{ 
                    padding: '8px 12px', 
                    backgroundColor: isDark ? '#2d2e33' : '#f8f8f8', 
                    borderRadius: '4px',
                  }}>
                    {new Date(supplierData.created_at * 1000).toLocaleString()}
                  </div>
                </div>
              </Col>
            )}
            {supplierData?.review_reason && (
              <Col span={24}>
                <div className='mb-4'>
                  <Text strong style={{ fontSize: '14px', display: 'block', marginBottom: '8px' }}>
                    {t('审核备注')}
                  </Text>
                  <div style={{ 
                    padding: '12px', 
                    backgroundColor: isDark ? '#2d2e33' : '#fff7e6', 
                    borderRadius: '4px',
                    border: '1px solid #ffd591',
                    color: '#ad6800'
                  }}>
                    {supplierData.review_reason}
                  </div>
                </div>
              </Col>
            )}
          </Row>

          <Divider margin='20px 12px'>
            <Text strong style={{ fontSize: '16px' }}>
              {t('企业主体信息')}
            </Text>
          </Divider>

          <Row gutter={12}>
            <InfoItem label={t('企业/主体名称')} value={supplierData?.company_name || '-'} />
            <InfoItem label={t('统一社会信用代码')} value={supplierData?.credit_code || '-'} />
            <InfoItem label={t('法人/经营者姓名')} value={supplierData?.legal_representative || '-'} />
            <InfoItem label={t('企业规模')} value={supplierData?.company_size || '-'} />
            {supplierData?.business_license_url && (
              <Col span={24}>
                <div className='mb-4'>
                  <Text strong style={{ fontSize: '14px', display: 'block', marginBottom: '8px' }}>
                    {t('营业执照')}
                  </Text>
                  <div style={{ 
                    padding: '12px', 
                    backgroundColor: isDark ? '#2d2e33' : '#f8f8f8', 
                    borderRadius: '4px',
                  }}>
                    <Image
                      src={supplierData.business_license_url}
                      width={200}
                      preview={{
                        src: supplierData.business_license_url,
                      }}
                      style={{ borderRadius: '4px' }}
                    />
                  </div>
                </div>
              </Col>
            )}
          </Row>

          <Divider margin='20px 12px'>
            <Text strong style={{ fontSize: '16px' }}>
              {t('对接人信息')}
            </Text>
          </Divider>

          <Row gutter={12}>
            <InfoItem label={t('对接人姓名')} value={supplierData?.contact_name || '-'} />
            <InfoItem label={t('对接人手机号')} value={supplierData?.contact_mobile || '-'} />
            <InfoItem label={t('对接人微信/企业微信')} value={supplierData?.contact_wechat || '-'} />
          </Row>
        </div>
      )}
    </Modal>
  );
};

export default SupplierDetailModal;
