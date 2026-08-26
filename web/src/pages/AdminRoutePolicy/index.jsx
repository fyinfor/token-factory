import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Select, Tabs, TabPane, Typography } from '@douyinfe/semi-ui';
import RoutePolicyCard from '../../components/settings/personal/cards/RoutePolicyCard';
import { API, showError } from '../../helpers';

const isDeletedUser = (u) =>
  Boolean(u?.deleted_at || u?.DeletedAt || u?.deletedAt);

const toUserOption = (u) => ({
  value: u.id,
  label: `${u.username || `#${u.id}`}${
    u.display_name ? ` (${u.display_name})` : ''
  } #${u.id}`,
  username: u.username || String(u.id),
});

const mapUsersToOptions = (users) =>
  (users || []).filter((u) => u?.id && !isDeletedUser(u)).map(toUserOption);

const AdminRoutePolicyPage = () => {
  const { t } = useTranslation();
  const [userOptions, setUserOptions] = useState([]);
  const [selectedUserId, setSelectedUserId] = useState();
  const [resolvedUser, setResolvedUser] = useState(null);
  const [searchLoading, setSearchLoading] = useState(false);
  const searchTimerRef = useRef(null);
  const selectedOptRef = useRef(null);

  const searchUsers = useCallback(
    async (keyword = '') => {
      setSearchLoading(true);
      try {
        const res = await API.get('/api/user/search', {
          params: {
            keyword: keyword || '',
            p: 1,
            page_size: 50,
          },
        });
        if (res.data?.success === false) {
          return;
        }
        const payload = res.data?.data;
        const items = Array.isArray(payload?.items)
          ? payload.items
          : Array.isArray(payload)
            ? payload
            : [];
        const next = mapUsersToOptions(items);
        const keep = selectedOptRef.current;
        if (keep && !next.some((o) => o.value === keep.value)) {
          setUserOptions([keep, ...next]);
        } else {
          setUserOptions(next);
        }
      } catch (err) {
        if (keyword) {
          showError(
            err.response?.data?.error ||
              err.message ||
              t('route_policy.admin_load_users_failed'),
          );
        }
      } finally {
        setSearchLoading(false);
      }
    },
    [t],
  );

  const debouncedSearch = useCallback(
    (keyword) => {
      if (searchTimerRef.current) {
        clearTimeout(searchTimerRef.current);
      }
      searchTimerRef.current = setTimeout(() => {
        searchUsers(keyword);
      }, 300);
    },
    [searchUsers],
  );

  useEffect(() => {
    searchUsers('');
    return () => {
      if (searchTimerRef.current) {
        clearTimeout(searchTimerRef.current);
      }
    };
  }, [searchUsers]);

  const handleUserChange = useCallback(
    (userId) => {
      setSelectedUserId(userId);
      if (!userId) {
        selectedOptRef.current = null;
        setResolvedUser(null);
        return;
      }
      const opt =
        userOptions.find((o) => o.value === userId) ||
        selectedOptRef.current ||
        {
          value: userId,
          label: `#${userId}`,
          username: String(userId),
        };
      selectedOptRef.current = opt;
      setResolvedUser({
        id: userId,
        username: opt.username || String(userId),
      });
    },
    [userOptions],
  );

  const selectedLabel = useMemo(() => {
    if (!resolvedUser) return '';
    return (
      selectedOptRef.current?.label ||
      userOptions.find((o) => o.value === resolvedUser.id)?.label ||
      `#${resolvedUser.id} ${resolvedUser.username}`
    );
  }, [resolvedUser, userOptions]);

  return (
    <div className='mt-[64px]'>
      <div className='px-2 md:px-8 py-4 md:py-6 space-y-4'>
        <Card className='!rounded-2xl shadow-sm border-0'>
          <Typography.Title heading={4} className='!mb-1'>
            {t('route_policy.admin_page_title')}
          </Typography.Title>
          <Typography.Text type='tertiary'>
            {t('route_policy.admin_page_description')}
          </Typography.Text>
        </Card>

        <Tabs type='line' keepDOM={false}>
          <TabPane tab={t('route_policy.admin_tab_template')} itemKey='template'>
            <div className='mt-4'>
              <RoutePolicyCard t={t} variant='admin-template' />
            </div>
          </TabPane>
          <TabPane tab={t('route_policy.admin_tab_user')} itemKey='user'>
            <div className='mt-4 space-y-4'>
              <Card className='!rounded-2xl shadow-sm border-0'>
                <Typography.Text strong className='block mb-2'>
                  {t('route_policy.admin_select_user')}
                </Typography.Text>
                <Select
                  style={{ width: '100%', maxWidth: 480 }}
                  filter
                  remote
                  showClear
                  placeholder={t('route_policy.admin_search_user_placeholder')}
                  optionList={userOptions}
                  value={selectedUserId}
                  loading={searchLoading}
                  onSearch={debouncedSearch}
                  onChange={handleUserChange}
                  onClear={() => handleUserChange(undefined)}
                  emptyContent={t('route_policy.admin_no_matching_user')}
                />
                {resolvedUser ? (
                  <div className='mt-3 text-sm text-gray-600 dark:text-gray-300'>
                    {t('route_policy.admin_currently_editing')}：{selectedLabel}
                  </div>
                ) : null}
              </Card>
              {resolvedUser ? (
                <RoutePolicyCard
                  t={t}
                  variant='admin-user'
                  managedUserId={resolvedUser.id}
                />
              ) : (
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <Typography.Text type='tertiary'>
                    {t('route_policy.admin_select_user_first')}
                  </Typography.Text>
                </Card>
              )}
            </div>
          </TabPane>
        </Tabs>
      </div>
    </div>
  );
};

export default AdminRoutePolicyPage;
