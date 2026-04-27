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
import { Modal, Form, Row, Col, Button, Typography, Divider, Image } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import { timestamp2string } from '../../../../helpers';

const { Text } = Typography;
const supplierTypeOptions = [
  { label: '公有云', value: '公有云' },
  { label: 'AIDC', value: 'AIDC' },
  { label: '企业中转站', value: '企业中转站' },
  { label: '个人中转站', value: '个人中转站' },
];

const ReviewApplicationModal = ({ visible, application, handleClose, handleReview }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const formApiRef = useRef(null);
  const isMobile = useIsMobile();

  useEffect(() => {
    if (!visible) {
      formApiRef.current?.reset();
      return;
    }
    if (application) {
      formApiRef.current?.setValues({
        reason: '',
        supplier_alias: application.supplier_alias || '',
        supplier_type: application.supplier_type || '',
      });
    }
  }, [visible, application]);

  const handleApprove = async () => {
    if (!application) return;
    const values = formApiRef.current?.getValues();
    if (!values.supplier_alias || values.supplier_alias.trim() === '') {
      formApiRef.current?.setError('supplier_alias', t('审批通过时必须填写供应商别名'));
      return;
    }
    if (!values.supplier_type || values.supplier_type.trim() === '') {
      formApiRef.current?.setError('supplier_type', t('审批通过时必须选择供应商类型'));
      return;
    }
    setLoading(true);
    await handleReview(application.id, {
      status: 1,
      reason: values.reason || '',
      supplier_alias: values.supplier_alias.trim(),
      supplier_type: values.supplier_type.trim(),
    });
    setLoading(false);
  };

  const handleReject = async () => {
    if (!application) return;
    const values = formApiRef.current?.getValues();
    if (!values.reason || values.reason.trim() === '') {
      formApiRef.current?.setError('reason', t('驳回时必须填写审批意见'));
      return;
    }
    setLoading(true);
    await handleReview(application.id, {
      status: 2,
      reason: values.reason,
      supplier_alias: '',
      supplier_type: '',
    });
    setLoading(false);
  };

  const getStatusText = (status) => {
    const statusMap = {
      0: t('待审核'),
      1: t('审核通过'),
      2: t('审核驳回'),
    };
    return statusMap[status] || t('未知');
  };

  const isReviewed = application?.status !== 0;
  const capability = application?.supplier_capability || null;
  const renderListValue = (value) => {
    if (!Array.isArray(value) || value.length === 0) {
      return t('未填写');
    }
    return value.join('、');
  };

  return (
    <Modal
      title={t('供应商申请审批')}
      visible={visible && !!application}
      onCancel={handleClose}
      size={isMobile ? 'full-width' : 'large'}
      style={{ maxWidth: isMobile ? '100%' : '800px' }}
      footer={
        <div style={{ display: 'flex', gap: '8px', justifyContent: 'flex-end', marginTop: '16px' }}>
          <Button
            type='primary'
            theme='solid'
            onClick={handleApprove}
            loading={loading}
            disabled={isReviewed}
          >
            {t('审批通过')}
          </Button>
          <Button
            type='danger'
            onClick={handleReject}
            loading={loading}
            disabled={isReviewed}
          >
            {t('审批不通过')}
          </Button>
        </div>
      }
    >
      <div style={{ maxHeight: '70vh', overflowY: 'auto', paddingRight: '8px' }}>
        {!application ? null : (<>
        <Divider margin='12px'>
          <Text strong style={{ fontSize: '16px' }}>
            {t('企业主体信息')}
          </Text>
        </Divider>

        <Row gutter={12}>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('企业/主体名称')}</Text>
              <div style={{ marginTop: '4px' }}>
                <Text strong>{application.company_name}</Text>
              </div>
            </div>
          </Col>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('统一社会信用代码')}</Text>
              <div style={{ marginTop: '4px' }}>
                <Text strong>{application.credit_code}</Text>
              </div>
            </div>
          </Col>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('企业Logo')}</Text>
              <div style={{ marginTop: '4px' }}>
                {application.company_logo_url ? (
                  <Image
                    src={application.company_logo_url}
                    alt={t('企业Logo')}
                    width={120}
                    height={120}
                    preview={{
                      src: application.company_logo_url,
                    }}
                    style={{ borderRadius: '8px', objectFit: 'contain' }}
                  />
                ) : (
                  <Text type='tertiary'>{t('未上传')}</Text>
                )}
              </div>
            </div>
          </Col>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('营业执照')}</Text>
              <div style={{ marginTop: '4px' }}>
                {application.business_license_url ? (
                  <Image
                    src={application.business_license_url}
                    alt={t('营业执照')}
                    width={200}
                    height={150}
                    preview={{
                      src: application.business_license_url,
                    }}
                  />
                ) : (
                  <Text type='tertiary'>{t('未上传')}</Text>
                )}
              </div>
            </div>
          </Col>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('法人/经营者姓名')}</Text>
              <div style={{ marginTop: '4px' }}>
                <Text strong>{application.legal_representative}</Text>
              </div>
            </div>
          </Col>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('企业规模')}</Text>
              <div style={{ marginTop: '4px' }}>
                <Text>{application.company_size || t('未填写')}</Text>
              </div>
            </div>
          </Col>
        </Row>

        <Divider margin='20px 12px'>
          <Text strong style={{ fontSize: '16px' }}>
            {t('对接人信息')}
          </Text>
        </Divider>

        <Row gutter={12}>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('对接人姓名')}</Text>
              <div style={{ marginTop: '4px' }}>
                <Text strong>{application.contact_name}</Text>
              </div>
            </div>
          </Col>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('对接人手机号')}</Text>
              <div style={{ marginTop: '4px' }}>
                <Text strong>{application.contact_mobile}</Text>
              </div>
            </div>
          </Col>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('对接人微信/企业微信')}</Text>
              <div style={{ marginTop: '4px' }}>
                <Text>{application.contact_wechat}</Text>
              </div>
            </div>
          </Col>
        </Row>

        <Divider margin='20px 12px'>
          <Text strong style={{ fontSize: '16px' }}>
            {t('申请信息')}
          </Text>
        </Divider>

        <Row gutter={12}>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('申请状态')}</Text>
              <div style={{ marginTop: '4px' }}>
                <Text>{getStatusText(application.status)}</Text>
              </div>
            </div>
          </Col>
          <Col span={24}>
            <div style={{ marginBottom: '16px' }}>
              <Text type='secondary'>{t('申请时间')}</Text>
              <div style={{ marginTop: '4px' }}>
                <Text>{timestamp2string(application.created_at)}</Text>
              </div>
            </div>
          </Col>
          {application.reviewed_at > 0 && (
            <Col span={24}>
              <div style={{ marginBottom: '16px' }}>
                <Text type='secondary'>{t('审批时间')}</Text>
                <div style={{ marginTop: '4px' }}>
                  <Text>{timestamp2string(application.reviewed_at)}</Text>
                </div>
              </div>
            </Col>
          )}
          {application.review_reason ? (
            <Col span={24}>
              <div style={{ marginBottom: '16px' }}>
                <Text type='secondary'>{t('审批意见')}</Text>
                <div style={{ marginTop: '4px' }}>
                  <Text>{application.review_reason}</Text>
                </div>
              </div>
            </Col>
          ) : null}
        </Row>

        <Divider margin='20px 12px'>
          <Text strong style={{ fontSize: '16px' }}>
            {t('技术能力信息')}
          </Text>
        </Divider>
        {!capability ? (
          <Text type='danger'>{t('申请方尚未填写技术能力信息')}</Text>
        ) : (
          <Row gutter={12}>
            <Col span={24}>
              <Text type='secondary'>{t('核心服务类型')}</Text>
              <div style={{ marginTop: '4px' }}><Text>{renderListValue(capability.core_service_types)}</Text></div>
            </Col>
            <Col span={24}>
              <Text type='secondary'>{t('支持的模型')}</Text>
              <div style={{ marginTop: '4px' }}><Text>{renderListValue(capability.supported_models)}</Text></div>
            </Col>
            <Col span={24}>
              <Text type='secondary'>{t('支持的API接口')}</Text>
              <div style={{ marginTop: '4px' }}><Text>{renderListValue(capability.supported_api_endpoints)}</Text></div>
            </Col>
            <Col span={24}>
              <Text type='secondary'>{t('支持的参数配置')}</Text>
              <div style={{ marginTop: '4px' }}><Text>{renderListValue(capability.supported_params)}</Text></div>
            </Col>
            <Col span={24}>
              <Text type='secondary'>{t('定价模式')}</Text>
              <div style={{ marginTop: '4px' }}><Text>{renderListValue(capability.pricing_modes)}</Text></div>
            </Col>
            <Col span={24}>
              <Text type='secondary'>{t('API接口地址')}</Text>
              <div style={{ marginTop: '4px' }}><Text>{renderListValue(capability.api_base_urls)}</Text></div>
            </Col>
          </Row>
        )}

        <Divider margin='20px 12px'>
          <Text strong style={{ fontSize: '16px' }}>
            {t('审批操作')}
          </Text>
        </Divider>

        <Form
          getFormApi={(api) => (formApiRef.current = api)}
        >
          <Form.Input
            field='supplier_alias'
            label={<Text strong>{t('供应商别名')}</Text>}
            placeholder={t('请输入供应商别名（管理员填写且全局唯一）')}
            maxLength={128}
            showClear
            disabled={isReviewed}
          />
          <Form.Select
            field='supplier_type'
            label={<Text strong>{t('供应商类型')}</Text>}
            placeholder={t('请选择供应商类型')}
            optionList={supplierTypeOptions}
            disabled={isReviewed}
            rules={[{ required: true, message: t('请选择供应商类型') }]}
          />
          <Form.TextArea
            field='reason'
            label={<Text strong>{t('审批意见')}</Text>}
            placeholder={t('请输入审批意见（驳回时必填）')}
            rows={4}
            maxLength={500}
            showClear
          />
        </Form>
        </>
        )}
      </div>
    </Modal>
  );
};

export default ReviewApplicationModal;
