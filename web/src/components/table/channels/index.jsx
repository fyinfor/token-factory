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
import { Banner } from '@douyinfe/semi-ui';
import { IconAlertTriangle } from '@douyinfe/semi-icons';
import CardPro from '../../common/ui/CardPro';
import ChannelsTable from './ChannelsTable';
import ChannelsActions from './ChannelsActions';
import ChannelsFilters from './ChannelsFilters';
import ChannelsTabs from './ChannelsTabs';
import { useChannelsData } from '../../../hooks/channels/useChannelsData';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
import BatchTagModal from './modals/BatchTagModal';
import ModelTestModal from './modals/ModelTestModal';
import ColumnSelectorModal from './modals/ColumnSelectorModal';
import EditChannelModal from './modals/EditChannelModal';
import EditTagModal from './modals/EditTagModal';
import MultiKeyManageModal from './modals/MultiKeyManageModal';
import ChannelUpstreamUpdateModal from './modals/ChannelUpstreamUpdateModal';
import ChannelOnboardModal from './modals/ChannelOnboardModal';
import { createCardProPagination } from '../../../helpers/utils';

const ChannelsPage = () => {
  const channelsData = useChannelsData();
  const isMobile = useIsMobile();
  const { channelBalanceAlertConfig, channelBalanceAlerts } = channelsData;

  const renderAlertNames = (items = []) =>
    items
      .slice(0, 6)
      .map((item) => item.name)
      .join('、');

  return (
    <>
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
      <ChannelOnboardModal
        visible={channelsData.showOnboardModal}
        channel={channelsData.currentOnboardChannel}
        onClose={channelsData.closeOnboardModal}
        onRefresh={channelsData.refresh}
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
      {channelBalanceAlertConfig.enabled &&
      channelBalanceAlerts.softChannels.length > 0 ? (
        <Banner
          type='info'
          closeIcon={null}
          description={channelsData
            .t(
              '渠道柔和提示：${count} 个渠道余额低于 ${threshold}，包括：${names}',
            )
            .replace('${count}', channelBalanceAlerts.softChannels.length)
            .replace(
              '${threshold}',
              String(channelBalanceAlertConfig.softThreshold),
            )
            .replace('${names}', renderAlertNames(channelBalanceAlerts.softChannels))}
          style={{ marginBottom: 12 }}
        />
      ) : null}
      {channelBalanceAlertConfig.enabled &&
      channelBalanceAlerts.riskChannels.length > 0 ? (
        <Banner
          type='warning'
          closeIcon={null}
          icon={
            <IconAlertTriangle
              size='large'
              style={{ color: 'var(--semi-color-warning)' }}
            />
          }
          description={channelsData
            .t(
              '渠道风险警告：${count} 个渠道余额低于 ${threshold}，包括：${names}',
            )
            .replace('${count}', channelBalanceAlerts.riskChannels.length)
            .replace(
              '${threshold}',
              String(channelBalanceAlertConfig.riskThreshold),
            )
            .replace('${names}', renderAlertNames(channelBalanceAlerts.riskChannels))}
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
    </>
  );
};

export default ChannelsPage;
