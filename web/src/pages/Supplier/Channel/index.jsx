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

import React from 'react';
import { Banner, Button, Empty } from '@douyinfe/semi-ui';
import { IconAlertTriangle } from '@douyinfe/semi-icons';
import {
  IllustrationNoAccess,
  IllustrationNoAccessDark,
} from '@douyinfe/semi-illustrations';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import CardPro from '../../../components/common/ui/CardPro';
import ChannelsTable from '../../../components/table/channels/ChannelsTable';
import ChannelsActions from '../../../components/table/channels/ChannelsActions';
import ChannelsFilters from '../../../components/table/channels/ChannelsFilters';
import ChannelsTabs from '../../../components/table/channels/ChannelsTabs';
import { useChannelsData } from '../../../hooks/channels/useChannelsData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import BatchTagModal from '../../../components/table/channels/modals/BatchTagModal';
import ModelTestModal from '../../../components/table/channels/modals/ModelTestModal';
import ColumnSelectorModal from '../../../components/table/channels/modals/ColumnSelectorModal';
import EditChannelModal from '../../../components/table/channels/modals/EditChannelModal';
import EditTagModal from '../../../components/table/channels/modals/EditTagModal';
import MultiKeyManageModal from '../../../components/table/channels/modals/MultiKeyManageModal';
import ChannelUpstreamUpdateModal from '../../../components/table/channels/modals/ChannelUpstreamUpdateModal';
import { createCardProPagination, isSupplier } from '../../../helpers/utils';

const SupplierChannelContent = () => {
  const channelsData = useChannelsData('/api/user/supplier/channels');
  const isMobile = useIsMobile();
  const { t } = useTranslation();

  return (
    <div className='mt-[60px] px-2'>
      {/* Modals */}
      <ColumnSelectorModal {...channelsData} />
      <EditTagModal
        visible={channelsData.showEditTag}
        tag={channelsData.editingTag}
        handleClose={() => channelsData.setShowEditTag(false)}
        refresh={channelsData.refresh}
      />
      <EditChannelModal
        refresh={channelsData.refresh}
        visible={channelsData.showEdit}
        handleClose={channelsData.closeEdit}
        editingChannel={channelsData.editingChannel}
      />
      <BatchTagModal {...channelsData} />
      <ModelTestModal {...channelsData} />
      <MultiKeyManageModal
        visible={channelsData.showMultiKeyManageModal}
        onCancel={() => channelsData.setShowMultiKeyManageModal(false)}
        channel={channelsData.currentMultiKeyChannel}
        onRefresh={channelsData.refresh}
      />
      <ChannelUpstreamUpdateModal
        visible={channelsData.showUpstreamUpdateModal}
        addModels={channelsData.upstreamUpdateAddModels}
        removeModels={channelsData.upstreamUpdateRemoveModels}
        preferredTab={channelsData.upstreamUpdatePreferredTab}
        confirmLoading={channelsData.upstreamApplyLoading}
        onConfirm={channelsData.applyUpstreamUpdates}
        onCancel={channelsData.closeUpstreamUpdateModal}
      />

      {/* Main Content */}
      {channelsData.globalPassThroughEnabled ? (
        <Banner
          type='warning'
          closeIcon={null}
          icon={
            <IconAlertTriangle
              size='large'
              style={{ color: 'var(--semi-color-warning)' }}
            />
          }
          description={channelsData.t(
            '已开启全局请求透传：参数覆写、模型重定向、渠道适配等 NewAPI 内置功能将失效，非最佳实践；如因此产生问题，请勿提交 issue 反馈。',
          )}
          style={{ marginBottom: 12 }}
        />
      ) : null}
      <CardPro
        type='type3'
        tabsArea={<ChannelsTabs {...channelsData} />}
        actionsArea={<ChannelsActions {...channelsData} />}
        searchArea={<ChannelsFilters {...channelsData} />}
        paginationArea={createCardProPagination({
          currentPage: channelsData.activePage,
          pageSize: channelsData.pageSize,
          total: channelsData.channelCount,
          onPageChange: channelsData.handlePageChange,
          onPageSizeChange: channelsData.handlePageSizeChange,
          isMobile: isMobile,
          t: channelsData.t,
        })}
        t={channelsData.t}
      >
        <ChannelsTable {...channelsData} />
      </CardPro>
    </div>
  );
};

const SupplierChannelPage = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  if (!isSupplier()) {
    return (
      <div className='mt-[60px] px-2'>
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
            title={t('需要供应商权限')}
            description={t('您需要先成为供应商才能访问此页面。')}
          >
            <Button
              theme='solid'
              type='primary'
              size='large'
              className='!rounded-md mt-4'
              style={{ fontWeight: 500 }}
              onClick={() => navigate('/console/supplier/apply')}
            >
              {t('前往申请')}
            </Button>
          </Empty>
        </div>
      </div>
    );
  }

  return <SupplierChannelContent />;
};

export default SupplierChannelPage;
