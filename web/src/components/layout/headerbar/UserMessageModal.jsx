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

import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Empty,
  Input,
  Modal,
  Pagination,
  Skeleton,
  TabPane,
  Tabs,
  Tag,
} from '@douyinfe/semi-ui';
import { API, showError, timestamp2string } from '../../../helpers';

const PAGE_SIZE = 10;

// UserMessageModal 展示用户站内消息并支持逐条已读。
const UserMessageModal = ({ visible, onClose, onReadStateChanged, t, isMobile }) => {
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [markingID, setMarkingID] = useState(0);
  const [markingAll, setMarkingAll] = useState(false);
  const [readTab, setReadTab] = useState('all');
  const [titleKeyword, setTitleKeyword] = useState('');

  // normalizePagedMessageData 将接口响应规范化为列表与总数。
  const normalizePagedMessageData = useCallback((data) => {
    const messageItems = Array.isArray(data?.items) ? data.items : [];
    const messageTotal = Number(data?.total || messageItems.length || 0);
    return { messageItems, messageTotal };
  }, []);

  // loadMessages 加载指定页站内消息（支持标题与已读状态筛选）。
  const loadMessages = useCallback(
    async (targetPage, overrideReadStatus, overrideTitleKeyword) => {
      setLoading(true);
      try {
        const effectiveReadStatus = overrideReadStatus || readTab;
        const effectiveTitleKeyword =
          overrideTitleKeyword !== undefined ? overrideTitleKeyword : titleKeyword;
        const res = await API.get('/api/user/messages/self', {
          params: {
            p: targetPage,
            page_size: PAGE_SIZE,
            title: effectiveTitleKeyword.trim(),
            read_status: effectiveReadStatus,
          },
          skipErrorHandler: true,
        });
        const { success, message, data } = res.data || {};
        if (!success) {
          showError(message || t('加载站内消息失败'));
          return;
        }
        const { messageItems, messageTotal } = normalizePagedMessageData(data);
        setItems(messageItems);
        setTotal(messageTotal);
      } catch (error) {
        showError(error?.message || t('加载站内消息失败'));
      } finally {
        setLoading(false);
      }
    },
    [normalizePagedMessageData, readTab, t, titleKeyword],
  );

  // handleMarkRead 将单条消息标记为已读。
  const handleMarkRead = useCallback(
    async (messageItem) => {
      if (!messageItem?.id || messageItem?.is_read || markingID > 0) {
        return;
      }
      setMarkingID(messageItem.id);
      try {
        const res = await API.post(`/api/user/messages/${messageItem.id}/read`, null, {
          skipErrorHandler: true,
        });
        const { success, message } = res.data || {};
        if (!success) {
          showError(message || t('标记已读失败'));
          return;
        }
        setItems((prev) =>
          prev.map((item) =>
            item.id === messageItem.id
              ? {
                  ...item,
                  is_read: true,
                  read_at: Math.floor(Date.now() / 1000),
                }
              : item,
          ),
        );
        if (onReadStateChanged) {
          await onReadStateChanged();
        }
      } catch (error) {
        showError(error?.message || t('标记已读失败'));
      } finally {
        setMarkingID(0);
      }
    },
    [markingID, onReadStateChanged, t],
  );

  // handlePageChange 处理分页切换。
  const handlePageChange = useCallback(
    (nextPage) => {
      setPage(nextPage);
      loadMessages(nextPage);
    },
    [loadMessages],
  );

  // handleMarkAllRead 将当前页可见未读消息全部标记为已读。
  const handleMarkAllRead = useCallback(async () => {
    if (markingAll) {
      return;
    }
    setMarkingAll(true);
    try {
      const res = await API.post('/api/user/messages/read_all', null, {
        skipErrorHandler: true,
      });
      const { success, message } = res.data || {};
      if (!success) {
        showError(message || t('全部标记已读失败'));
        return;
      }
      setItems((prev) =>
        prev.map((item) => ({
          ...item,
          is_read: true,
          read_at: item.read_at || Math.floor(Date.now() / 1000),
        })),
      );
      if (onReadStateChanged) {
        await onReadStateChanged();
      }
      loadMessages(page);
    } catch (error) {
      // 兼容后端尚未升级 read_all 路由时的兜底逻辑：逐条标记当前页未读消息。
      if (error?.response?.status === 404) {
        const unreadItems = items.filter((item) => !item.is_read);
        try {
          await Promise.all(
            unreadItems.map((item) =>
              API.post(`/api/user/messages/${item.id}/read`, null, {
                skipErrorHandler: true,
              }),
            ),
          );
          setItems((prev) =>
            prev.map((item) => ({
              ...item,
              is_read: true,
              read_at: item.read_at || Math.floor(Date.now() / 1000),
            })),
          );
          if (onReadStateChanged) {
            await onReadStateChanged();
          }
          loadMessages(page);
          return;
        } catch (fallbackError) {
          showError(fallbackError?.message || t('全部标记已读失败'));
          return;
        }
      }
      showError(error?.message || t('全部标记已读失败'));
    } finally {
      setMarkingAll(false);
    }
  }, [items, loadMessages, markingAll, onReadStateChanged, page, t]);

  useEffect(() => {
    if (!visible) {
      return;
    }
    // 仅在弹窗打开时初始化一次，避免输入时被副作用重置。
    const initialTab = 'all';
    const initialKeyword = '';
    setPage(1);
    setReadTab(initialTab);
    setTitleKeyword(initialKeyword);
    loadMessages(1, initialTab, initialKeyword);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible]);
  const unreadVisibleCount = items.filter((item) => !item.is_read).length;

  // handleTabChange 切换查看全部/查看已读/查看未读。
  const handleTabChange = useCallback(
    (tabKey) => {
      setReadTab(tabKey);
      setPage(1);
      loadMessages(1, tabKey, titleKeyword);
    },
    [loadMessages, titleKeyword],
  );

  // handleSearch 按标题关键词搜索消息。
  const handleSearch = useCallback(() => {
    setPage(1);
    loadMessages(1, readTab, titleKeyword);
  }, [loadMessages, readTab, titleKeyword]);

  return (
    <Modal
      title={t('站内消息')}
      visible={visible}
      onCancel={onClose}
      onOk={onClose}
      okText={t('关闭')}
      cancelButtonProps={{ style: { display: 'none' } }}
      size={isMobile ? 'full-width' : 'large'}
    >
      <div className='mb-3 flex items-center justify-between gap-2'>
        <Tabs type='button' activeKey={readTab} onChange={handleTabChange}>
          <TabPane itemKey='all' tab={t('查看全部')} />
          <TabPane itemKey='read' tab={t('查看已读')} />
          <TabPane itemKey='unread' tab={t('查看未读')} />
        </Tabs>
      </div>
      <div className='mb-3 flex items-center gap-2'>
        <Input
          value={titleKeyword}
          size='small'
          style={{ maxWidth: isMobile ? '100%' : 260 }}
          onChange={(value) => setTitleKeyword(value)}
          onEnterPress={handleSearch}
          placeholder={t('按标题搜索')}
        />
        <Button
          size='small'
          theme='light'
          type='primary'
          onClick={handleSearch}
        >
          {t('查询')}
        </Button>
        <Button
          size='small'
          theme='light'
          type='tertiary'
          onClick={handleMarkAllRead}
          loading={markingAll}
          disabled={unreadVisibleCount === 0}
        >
          {t('全部标记已读')}
        </Button>
      </div>
      <div className='max-h-[60vh] overflow-y-auto pr-1'>
        {loading ? (
          <div className='space-y-3'>
            <Skeleton placeholder={<Skeleton.Title style={{ width: '80%' }} />} loading />
            <Skeleton placeholder={<Skeleton.Paragraph rows={2} />} loading />
            <Skeleton placeholder={<Skeleton.Paragraph rows={2} />} loading />
          </div>
        ) : items.length === 0 ? (
          <Empty description={t('暂无站内消息')} />
        ) : (
          <div className='space-y-3'>
            {items.map((item) => (
              <div
                key={item.id}
                className='border border-semi-color-border rounded-lg p-3 bg-semi-color-bg-0'
              >
                <div className='flex items-center justify-between gap-2 mb-2'>
                  <div className='text-sm font-medium text-semi-color-text-0'>
                    {item.title || t('消息')}
                  </div>
                  {item.is_read ? (
                    <Tag color='green' size='small'>
                      {t('已读')}
                    </Tag>
                  ) : (
                    <Tag color='red' size='small'>
                      {t('未读')}
                    </Tag>
                  )}
                </div>
                <div className='text-sm text-semi-color-text-1 whitespace-pre-wrap break-words'>
                  {item.content || ''}
                </div>
                <div className='mt-2 flex items-center justify-between'>
                  <span className='text-xs text-semi-color-text-2'>
                    {timestamp2string(item.created_at || 0)}
                  </span>
                  {!item.is_read && (
                    <Button
                      size='small'
                      theme='light'
                      type='primary'
                      loading={markingID === item.id}
                      onClick={() => handleMarkRead(item)}
                    >
                      {t('标记已读')}
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
      {total > PAGE_SIZE && (
        <div className='mt-3 flex justify-end'>
          <Pagination
            size='small'
            pageSize={PAGE_SIZE}
            currentPage={page}
            total={total}
            onPageChange={handlePageChange}
            showTotal
          />
        </div>
      )}
    </Modal>
  );
};

export default UserMessageModal;

