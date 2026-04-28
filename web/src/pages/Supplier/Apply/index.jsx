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
import { Button, Empty, Spin, Tag } from '@douyinfe/semi-ui';
import {
  IllustrationNoAccess,
  IllustrationNoAccessDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';
import { API, isSupplier } from '../../../helpers';
import SupplierApplicationModal from '../../../components/supplier/SupplierApplicationModal';
import SupplierDetailModal from '../../../components/supplier/SupplierDetailModal';

const SupplierApplyPage = () => {
  const { t } = useTranslation();
  const [showApplicationModal, setShowApplicationModal] = useState(false);
  const [showDetailModal, setShowDetailModal] = useState(false);
  const [applicationData, setApplicationData] = useState(null);
  const [loading, setLoading] = useState(true);

  const fetchApplicationData = async () => {
    try {
      const res = await API.get('/api/user/supplier/application/self');
      if (res.data.success && res.data.data) {
        setApplicationData(res.data.data);
      }
    } catch (error) {
      console.error('Failed to fetch application data:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchApplicationData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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

  const getButtonText = () => {
    if (!applicationData) {
      return t('申请成为供应商');
    }
    switch (applicationData.status) {
      case 0:
        return t('修改申请（待审核）');
      case 2:
        return t('重新申请（已驳回）');
      case 3:
        return t('重新申请（已注销）');
      default:
        return t('申请成为供应商');
    }
  };

  const canApplyOrModify =
    !applicationData ||
    applicationData.status === 0 ||
    applicationData.status === 2 ||
    applicationData.status === 3;

  const handleModalClose = () => {
    setShowApplicationModal(false);
    fetchApplicationData();
  };

  if (loading) {
    return (
      <div
        className='mt-[60px] px-2 flex items-center justify-center'
        style={{ minHeight: 'calc(100vh - 360px)' }}
      >
        <Spin size='large' />
      </div>
    );
  }

  return (
    <div className='mt-[60px] px-2'>
      <SupplierApplicationModal
        visible={showApplicationModal}
        handleClose={handleModalClose}
      />
      <SupplierDetailModal
        visible={showDetailModal}
        handleClose={() => setShowDetailModal(false)}
      />
      {!isSupplier() ? (
        <div
          className='flex items-center justify-center'
          style={{ minHeight: 'calc(100vh - 360px)' }}
        >
          <Empty
            image={<IllustrationNoAccess style={{ width: 200, height: 200 }} />}
            darkModeImage={
              <IllustrationNoAccessDark style={{ width: 200, height: 200 }} />
            }
            layout='horizontal'
            title={
              <div className='flex items-center gap-2'>
                {t('成为供应商')}
                {applicationData && getStatusTag(applicationData.status)}
              </div>
            }
            description={
              applicationData && applicationData.status === 0 ? (
                t('您的申请正在审核中，请耐心等待。')
              ) : applicationData && applicationData.status === 2 ? (
                <div>
                  <div>{t('您的申请已被驳回，可以修改后重新提交。')}</div>
                  {applicationData.review_reason && (
                    <div className='mt-2 p-3 bg-red-50 border border-red-200 rounded-md text-sm text-red-700'>
                      <strong>{t('驳回原因：')}</strong>
                      {applicationData.review_reason}
                    </div>
                  )}
                </div>
              ) : applicationData && applicationData.status === 3 ? (
                <div>
                  <div>{t('您的供应商资格已被注销，可以重新申请。')}</div>
                  {applicationData.review_reason && (
                    <div className='mt-2 p-3 bg-gray-50 border border-gray-200 rounded-md text-sm text-gray-700'>
                      <strong>{t('注销原因：')}</strong>
                      {applicationData.review_reason}
                    </div>
                  )}
                </div>
              ) : (
                t(
                  '您当前还不是供应商，申请成为供应商后即可管理您的供应商信息。',
                )
              )
            }
          >
            {canApplyOrModify && (
              <Button
                theme='solid'
                type='primary'
                size='large'
                className='!rounded-md mt-4'
                style={{ fontWeight: 500 }}
                onClick={() => setShowApplicationModal(true)}
              >
                {getButtonText()}
              </Button>
            )}
          </Empty>
        </div>
      ) : (
        <div
          className='flex items-center justify-center'
          style={{ minHeight: 'calc(100vh - 360px)' }}
        >
          <Empty
            image={<IllustrationNoAccess style={{ width: 200, height: 200 }} />}
            darkModeImage={
              <IllustrationNoAccessDark style={{ width: 200, height: 200 }} />
            }
            layout='horizontal'
            title={t('已是供应商')}
            description={t(
              '您已经是供应商，可以在渠道管理和模型管理中管理您的供应商信息。',
            )}
          >
            <Button
              theme='solid'
              type='primary'
              size='large'
              className='!rounded-md mt-4'
              style={{ fontWeight: 500 }}
              onClick={() => setShowDetailModal(true)}
            >
              {t('查看供应商详情')}
            </Button>
          </Empty>
        </div>
      )}
    </div>
  );
};

export default SupplierApplyPage;
