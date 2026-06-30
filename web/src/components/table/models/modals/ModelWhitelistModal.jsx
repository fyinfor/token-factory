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

import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  Avatar,
  Button,
  Card,
  Col,
  Empty,
  Form,
  Input,
  Modal,
  Popconfirm,
  Radio,
  RadioGroup,
  Row,
  Select,
  SideSheet,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconEyeOpened,
  IconGlobe,
  IconInfoCircle,
  IconPlus,
  IconSave,
  IconSetting,
  IconUserGroup,
} from '@douyinfe/semi-icons';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Search } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  stringToColor,
} from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import CardTable from '../../../common/ui/CardTable';
import {
  renderDescription,
  renderLimitedItems,
} from '../../../common/ui/RenderUtils';

const { Text, Title } = Typography;

const tableSurfaceClassName =
  'w-full min-w-0 overflow-hidden rounded-lg border border-[var(--semi-color-border)] [&_.semi-spin]:!w-full [&_.semi-spin-children]:!w-full [&_.semi-table-wrapper]:!w-full [&_.semi-table-container]:!w-full [&_.semi-table]:!w-full [&_.semi-table-body]:!w-full [&_.semi-table-pagination-outer]:!min-h-0 [&_.semi-table-pagination-outer]:!px-3 [&_.semi-table-pagination-outer]:!py-2 [&_.semi-table-pagination-outer]:!mt-0 [&_.semi-table-pagination-outer]:!gap-2 [&_.semi-page]:!py-0';

const compactSearchClassName =
  'min-w-0 [&_.semi-input-prefix]:!ml-1.5 [&_.semi-input-prefix]:!mr-1.5 [&_.semi-input]:!h-8';

const uniq = (items = []) => {
  const seen = new Set();
  const out = [];
  items.forEach((item) => {
    const value = typeof item === 'number' ? item : String(item ?? '').trim();
    if (value === '' || value === 0 || seen.has(value)) return;
    seen.add(value);
    out.push(value);
  });
  return out;
};

const uniqIDs = (items = []) => {
  const seen = new Set();
  const out = [];
  items.forEach((item) => {
    const value = Number(item);
    if (!Number.isInteger(value) || value <= 0 || seen.has(value)) return;
    seen.add(value);
    out.push(value);
  });
  return out;
};

const userLabel = (user) => {
  if (!user) return '';
  const name = user.display_name || user.username || `#${user.id}`;
  return `${name} (#${user.id})`;
};

const roleLabel = (role, t) => {
  if (role >= 100) return t('超级管理员');
  if (role >= 10) return t('管理员');
  if (role === 5) return t('分销商');
  if (role === 0) return t('访客');
  return t('普通用户');
};

const roleColor = (role) => {
  if (role >= 100) return 'red';
  if (role >= 10) return 'orange';
  if (role === 5) return 'blue';
  if (role === 0) return 'grey';
  return 'green';
};

const toOption = (value) => ({ label: value, value });

const renderUserTags = (tags) => {
  const items = String(tags || '')
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean);
  if (items.length === 0) {
    return <Text type='tertiary'>-</Text>;
  }
  return renderLimitedItems({
    items,
    renderItem: (tag, idx) => (
      <Tag
        key={`${tag}-${idx}`}
        size='small'
        shape='circle'
        color={stringToColor(tag)}
      >
        {tag}
      </Tag>
    ),
    maxDisplay: 3,
  });
};

const ModelWhitelistModal = ({ visible, model, onClose, onChanged }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);
  const previewRequestIDRef = useRef(0);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [previewing, setPreviewing] = useState(false);
  const [sets, setSets] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [editingSet, setEditingSet] = useState(null);
  const [selectedUsers, setSelectedUsers] = useState([]);
  const [showUserPicker, setShowUserPicker] = useState(false);
  const [userPickerKeyword, setUserPickerKeyword] = useState('');
  const [userPickerUsers, setUserPickerUsers] = useState([]);
  const [userPickerLoading, setUserPickerLoading] = useState(false);
  const [userPickerPage, setUserPickerPage] = useState(1);
  const [userPickerPageSize, setUserPickerPageSize] = useState(10);
  const [userPickerTotal, setUserPickerTotal] = useState(0);
  const [groupOptions, setGroupOptions] = useState([]);
  const [tagOptions, setTagOptions] = useState([]);
  const [previewUsers, setPreviewUsers] = useState([]);
  const [previewKeyword, setPreviewKeyword] = useState('');
  const [previewPage, setPreviewPage] = useState(1);
  const [previewPageSize, setPreviewPageSize] = useState(10);
  const [previewRefreshKey, setPreviewRefreshKey] = useState(0);
  const [savingBindings, setSavingBindings] = useState(false);
  const [visibilityMode, setVisibilityMode] = useState('public');
  const [selectedSetIDs, setSelectedSetIDs] = useState([]);
  const [showSetManager, setShowSetManager] = useState(false);

  const editing = Boolean(editingSet);
  const currentModelName = model?.model_name || '-';
  const currentModelID = model?.id;
  const selectedSetCount = selectedSetIDs.length;
  const scopeSummary =
    visibilityMode === 'sets'
      ? selectedSetCount > 0
        ? t('已选择 {{count}} 个用户集', { count: selectedSetCount })
        : t('未选择用户集')
      : t('所有人可见');
  const scopeHelpText =
    visibilityMode === 'sets'
      ? t('仅命中所选用户集的登录用户可见')
      : t('所有用户和未登录访问均可见');
  const saveWhitelistText = savingBindings ? t('保存中…') : t('保存白名单');

  const loadSets = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/model_visibility/sets', {
        params: { keyword, page_size: 100 },
      });
      if (res.data?.success) {
        const data = res.data.data;
        setSets(data?.items || data || []);
      } else {
        showError(res.data?.message || t('获取用户集失败'));
      }
    } catch (error) {
      showError(error.message || t('获取用户集失败'));
    } finally {
      setLoading(false);
    }
  }, [keyword, t]);

  const loadGroupsAndTags = useCallback(async () => {
    try {
      const [groupRes, tagRes] = await Promise.all([
        API.get('/api/group/'),
        API.get('/api/user/tags'),
      ]);
      if (groupRes.data?.success) {
        setGroupOptions((groupRes.data.data || []).map(toOption));
      }
      if (tagRes.data?.success) {
        setTagOptions((tagRes.data.data || []).map(toOption));
      }
    } catch {
      // keep the form usable with allowCreate
    }
  }, []);

  const queryUsers = useCallback(async (value, page = 1, pageSize = 10) => {
    try {
      const res = await API.get('/api/model_visibility/users', {
        params: { keyword: value || '', p: page, page_size: pageSize },
      });
      if (res.data?.success) {
        const data = res.data.data;
        const items = data?.items || data || [];
        return {
          items: Array.isArray(items) ? items : [],
          total: data?.total || 0,
          page: data?.page || page,
          pageSize: data?.page_size || pageSize,
        };
      }
    } catch {
      // keep the editor open even if the backend has not exposed this helper yet
    }
    return { items: [], total: 0, page, pageSize };
  }, []);

  const searchPickerUsers = useCallback(
    async (
      value = userPickerKeyword,
      page = userPickerPage,
      pageSize = userPickerPageSize,
    ) => {
      setUserPickerLoading(true);
      try {
        const data = await queryUsers(value, page, pageSize);
        setUserPickerUsers(data.items);
        setUserPickerTotal(data.total);
        setUserPickerPage(data.page);
        setUserPickerPageSize(data.pageSize);
      } finally {
        setUserPickerLoading(false);
      }
    },
    [queryUsers, userPickerKeyword, userPickerPage, userPickerPageSize],
  );

  const openEditor = useCallback(
    async (record = null) => {
      setPreviewUsers([]);
      setSelectedUsers([]);
      if (!record?.id) {
        setEditingSet({
          id: undefined,
          name: '',
          description: '',
          user_ids: [],
          user_tags: [],
          user_groups: [],
        });
        loadGroupsAndTags();
        searchPickerUsers('');
        return;
      }
      await loadGroupsAndTags();
      await searchPickerUsers('');
      setLoading(true);
      try {
        const res = await API.get(`/api/model_visibility/sets/${record.id}`);
        if (res.data?.success) {
          const detail = res.data.data || {};
          const users = Array.isArray(detail.users) ? detail.users : [];
          setSelectedUsers(users);
          setEditingSet({
            id: detail.id,
            name: detail.name || '',
            description: detail.description || '',
            user_ids: uniqIDs(detail.user_ids || users.map((user) => user.id)),
            user_tags: uniq(detail.user_tags),
            user_groups: uniq(detail.user_groups),
          });
        } else {
          showError(res.data?.message || t('获取用户集失败'));
        }
      } catch (error) {
        showError(error.message || t('获取用户集失败'));
      } finally {
        setLoading(false);
      }
    },
    [loadGroupsAndTags, searchPickerUsers, t],
  );

  const closeEditor = useCallback(() => {
    setEditingSet(null);
    setSelectedUsers([]);
    setShowUserPicker(false);
    setPreviewUsers([]);
    formApiRef.current?.reset();
  }, []);

  const deleteSet = useCallback(
    async (id) => {
      try {
        const res = await API.delete(`/api/model_visibility/sets/${id}`);
        if (res.data?.success) {
          showSuccess(t('删除成功'));
          const nextSelected = selectedSetIDs.filter((setID) => setID !== id);
          setSelectedSetIDs(nextSelected);
          if (visibilityMode === 'sets' && nextSelected.length === 0) {
            setVisibilityMode('public');
          }
          await loadSets();
          onChanged?.();
        } else {
          showError(res.data?.message || t('删除失败'));
        }
      } catch (error) {
        showError(error.message || t('删除失败'));
      }
    },
    [loadSets, onChanged, selectedSetIDs, t, visibilityMode],
  );

  const saveSet = async (values) => {
    const payload = {
      id: editingSet?.id,
      name: String(values.name || '').trim(),
      description: String(values.description || '').trim(),
      user_ids: uniqIDs(selectedUsers.map((user) => user.id)),
      user_tags: uniq(values.user_tags || []),
      user_groups: uniq(values.user_groups || []),
    };
    if (!payload.name) {
      showError(t('请输入用户集名称'));
      return;
    }
    if (
      payload.user_ids.length === 0 &&
      payload.user_tags.length === 0 &&
      payload.user_groups.length === 0
    ) {
      showError(t('用户集至少需要选择一个显式用户、用户标签或用户分组'));
      return;
    }
    setSaving(true);
    try {
      const request = payload.id
        ? API.put('/api/model_visibility/sets', payload)
        : API.post('/api/model_visibility/sets', payload);
      const res = await request;
      if (res.data?.success) {
        const savedSet = res.data?.data;
        showSuccess(payload.id ? t('更新成功') : t('创建成功'));
        if (!payload.id && savedSet?.id && visibilityMode === 'sets') {
          setSelectedSetIDs((prev) => uniqIDs([...prev, savedSet.id]));
        }
        closeEditor();
        await loadSets();
        onChanged?.();
      } else {
        showError(res.data?.message || t('保存失败'));
      }
    } catch (error) {
      showError(error.message || t('保存失败'));
    } finally {
      setSaving(false);
    }
  };

  const saveModelBindings = async () => {
    if (!currentModelID) {
      showError(t('缺少模型 ID'));
      return;
    }
    const ids = visibilityMode === 'sets' ? uniqIDs(selectedSetIDs) : [];
    if (visibilityMode === 'sets' && ids.length === 0) {
      showError(t('请选择至少一个用户集'));
      return;
    }
    setSavingBindings(true);
    try {
      const res = await API.put(
        `/api/model_visibility/models/${currentModelID}`,
        {
          visibility_set_ids: ids,
        },
      );
      if (res.data?.success) {
        const data = res.data?.data || {};
        const nextIDs = Array.isArray(data.visibility_set_ids)
          ? data.visibility_set_ids
          : ids;
        setSelectedSetIDs(nextIDs);
        setVisibilityMode(nextIDs.length > 0 ? 'sets' : 'public');
        showSuccess(t('白名单已保存'));
        await onChanged?.();
      } else {
        showError(res.data?.message || t('保存失败'));
      }
    } catch (error) {
      showError(error.message || t('保存失败'));
    } finally {
      setSavingBindings(false);
    }
  };

  const requestPreviewRefresh = useCallback(() => {
    setPreviewRefreshKey((key) => key + 1);
  }, []);

  const previewMatchedUsers = useCallback(
    async ({ resetPage = true } = {}) => {
      const values = formApiRef.current?.getValues() || {};
      const payload = {
        user_ids: uniqIDs(selectedUsers.map((user) => user.id)),
        user_tags: uniq(values.user_tags || []),
        user_groups: uniq(values.user_groups || []),
        limit: 1000,
      };
      if (
        payload.user_ids.length === 0 &&
        payload.user_tags.length === 0 &&
        payload.user_groups.length === 0
      ) {
        setPreviewUsers([]);
        setPreviewing(false);
        if (resetPage) setPreviewPage(1);
        return;
      }

      const requestID = previewRequestIDRef.current + 1;
      previewRequestIDRef.current = requestID;
      if (resetPage) setPreviewPage(1);
      setPreviewing(true);
      try {
        const res = await API.post(
          '/api/model_visibility/users/preview',
          payload,
        );
        if (requestID !== previewRequestIDRef.current) return;
        if (res.data?.success) {
          setPreviewUsers(res.data.data?.items || []);
        } else {
          showError(res.data?.message || t('预览失败'));
        }
      } catch (error) {
        if (requestID === previewRequestIDRef.current) {
          showError(error.message || t('预览失败'));
        }
      } finally {
        if (requestID === previewRequestIDRef.current) {
          setPreviewing(false);
        }
      }
    },
    [selectedUsers, t],
  );

  const filteredPreviewUsers = useMemo(() => {
    const keyword = String(previewKeyword || '')
      .trim()
      .toLowerCase();
    if (!keyword) return previewUsers;
    return previewUsers.filter((user) => {
      const haystack = [
        user.username,
        user.display_name,
        user.phone,
        user.email,
        user.tags,
        String(user.id),
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase();
      return haystack.includes(keyword);
    });
  }, [previewKeyword, previewUsers]);

  const pagedPreviewUsers = useMemo(() => {
    const start = (previewPage - 1) * previewPageSize;
    return filteredPreviewUsers.slice(start, start + previewPageSize);
  }, [filteredPreviewUsers, previewPage, previewPageSize]);

  const selectedUserIDsKey = useMemo(
    () =>
      selectedUsers
        .map((user) => user.id)
        .sort((a, b) => a - b)
        .join(','),
    [selectedUsers],
  );

  const columns = useMemo(
    () => [
      {
        title: t('用户集名称'),
        dataIndex: 'name',
        render: (text) => <Text strong>{text}</Text>,
      },
      {
        title: t('描述'),
        dataIndex: 'description',
        render: (text) => renderDescription(text, 120),
      },
      {
        title: t('可见范围'),
        dataIndex: 'id',
        render: (_, record) => {
          const items = [
            ...(record.user_ids || []).map((id) => `#${id}`),
            ...(record.user_tags || []).map((tag) => `${t('标签')}: ${tag}`),
            ...(record.user_groups || []).map(
              (group) => `${t('分组')}: ${group}`,
            ),
          ];
          if (items.length === 0) {
            return <Text type='tertiary'>{t('暂未配置')}</Text>;
          }
          return renderLimitedItems({
            items,
            renderItem: (item, idx) => (
              <Tag
                key={`${item}-${idx}`}
                size='small'
                shape='circle'
                color={stringToColor(item)}
              >
                {item}
              </Tag>
            ),
            maxDisplay: 4,
          });
        },
      },
      {
        title: t('命中用户'),
        dataIndex: 'member_count',
        width: 100,
        render: (count) => count || 0,
      },
      {
        title: '',
        dataIndex: 'operate',
        fixed: 'right',
        width: 160,
        render: (_, record) => (
          <Space className='whitespace-nowrap'>
            <Button size='small' onClick={() => openEditor(record)}>
              {t('编辑')}
            </Button>
            <Popconfirm
              title={t('确定删除此用户集？')}
              content={t(
                '删除后会同步解除模型关联；没有剩余用户集的模型会回到所有人可见。',
              )}
              onConfirm={() => deleteSet(record.id)}
              okType='danger'
              position='left'
              style={{ maxWidth: 280 }}
            >
              <Button size='small' type='danger'>
                {t('删除')}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [deleteSet, openEditor, t],
  );

  const selectedUserColumns = useMemo(
    () => [
      {
        title: t('用户名'),
        dataIndex: 'username',
        render: (_, record) => <Text>{userLabel(record)}</Text>,
      },
      {
        title: t('手机号'),
        dataIndex: 'phone',
        render: (phone) => phone || '-',
      },
      {
        title: t('标签'),
        dataIndex: 'tags',
        render: renderUserTags,
      },
      {
        title: t('权限'),
        dataIndex: 'role',
        width: 100,
        render: (role) => (
          <Tag size='small' shape='circle' color={roleColor(role)}>
            {roleLabel(role, t)}
          </Tag>
        ),
      },
      {
        title: '',
        dataIndex: 'operate',
        width: 80,
        render: (_, record) => (
          <Button
            size='small'
            type='danger'
            onClick={() => {
              const nextUsers = selectedUsers.filter(
                (user) => user.id !== record.id,
              );
              setSelectedUsers(nextUsers);
            }}
          >
            {t('移除')}
          </Button>
        ),
      },
    ],
    [selectedUsers, t],
  );

  const pickerUserColumns = useMemo(
    () => [
      {
        title: t('用户名'),
        dataIndex: 'username',
        render: (_, record) => <Text>{userLabel(record)}</Text>,
      },
      {
        title: t('手机号'),
        dataIndex: 'phone',
        render: (phone) => phone || '-',
      },
      {
        title: t('标签'),
        dataIndex: 'tags',
        render: renderUserTags,
      },
      {
        title: t('权限'),
        dataIndex: 'role',
        width: 100,
        render: (role) => (
          <Tag size='small' shape='circle' color={roleColor(role)}>
            {roleLabel(role, t)}
          </Tag>
        ),
      },
    ],
    [t],
  );

  useEffect(() => {
    if (!visible) {
      setSelectedSetIDs([]);
      setVisibilityMode('public');
      setShowSetManager(false);
      return;
    }
    const ids = Array.isArray(model?.visibility_set_ids)
      ? uniqIDs(model.visibility_set_ids)
      : [];
    setSelectedSetIDs(ids);
    setVisibilityMode(ids.length > 0 ? 'sets' : 'public');
  }, [visible, model?.id, model?.visibility_set_ids]);

  useEffect(() => {
    if (visible) {
      loadSets();
      loadGroupsAndTags();
      searchPickerUsers('');
    }
  }, [visible, loadSets, loadGroupsAndTags, searchPickerUsers]);

  useEffect(() => {
    if (!editing) return undefined;
    const timer = setTimeout(() => {
      previewMatchedUsers();
    }, 350);
    return () => clearTimeout(timer);
  }, [editing, previewMatchedUsers, previewRefreshKey, selectedUserIDsKey]);

  return (
    <SideSheet
      placement='right'
      title={
        <Space>
          <Tag color='blue' shape='circle'>
            {t('管理')}
          </Tag>
          <Title heading={4} className='m-0'>
            {t('白名单')}
          </Title>
        </Space>
      }
      visible={visible}
      onCancel={onClose}
      width={isMobile ? '100%' : 960}
      bodyStyle={{ padding: 0, overscrollBehavior: 'contain' }}
      closeIcon={null}
      footer={
        <div className='flex flex-col gap-3 rounded-xl border-t border-[var(--semi-color-border)] bg-[var(--semi-color-bg-0)] px-4 py-3 sm:flex-row sm:items-center sm:justify-between'>
          <div className='min-w-0'>
            <div className='flex min-w-0 items-center gap-2'>
              <Text strong>{t('当前可见范围')}</Text>
              <Tag
                color={visibilityMode === 'sets' ? 'orange' : 'green'}
                shape='circle'
              >
                {scopeSummary}
              </Tag>
            </div>
            <Text type='tertiary' size='small'>
              {t('保存后立即生效')}
            </Text>
          </div>
          <Button
            type='primary'
            theme='solid'
            size='large'
            icon={<IconSave />}
            loading={savingBindings}
            disabled={!currentModelID}
            onClick={saveModelBindings}
            className='w-full sm:w-auto'
          >
            {saveWhitelistText}
          </Button>
        </div>
      }
    >
      <style>
        {`
          .model-whitelist-edit-modal .semi-modal-body {
            scrollbar-width: thin;
            scrollbar-color: var(--semi-color-fill-2) transparent;
          }

          .model-whitelist-edit-modal .semi-modal-body::-webkit-scrollbar {
            width: 6px;
            height: 6px;
          }

          .model-whitelist-edit-modal .semi-modal-body::-webkit-scrollbar-thumb {
            border-radius: 999px;
            background: var(--semi-color-fill-2);
          }

          .model-whitelist-edit-modal .semi-modal-body::-webkit-scrollbar-track {
            background: transparent;
          }
        `}
      </style>
      <Spin spinning={loading}>
        <div className='flex flex-col gap-3 p-3 pb-4'>
          <Card className='!rounded-lg border border-[var(--semi-color-border)] shadow-sm'>
            <div className='flex flex-col gap-4'>
              <div className='flex flex-col gap-3 md:flex-row md:items-start md:justify-between'>
                <div className='min-w-0 flex items-start gap-3'>
                  <Avatar size='small' color='blue' className='shrink-0'>
                    <IconSetting size={16} />
                  </Avatar>
                  <div className='min-w-0'>
                    <Title heading={5} className='!mb-1 !mt-0'>
                      {t('设置白名单')}
                    </Title>
                    <div className='flex min-w-0 flex-wrap items-center gap-2'>
                      <Text type='tertiary'>{t('当前模型')}</Text>
                      <Text
                        strong
                        ellipsis={{ showTooltip: true }}
                        className='max-w-full'
                        translate='no'
                      >
                        {currentModelName}
                      </Text>
                    </div>
                  </div>
                </div>
                <div className='flex shrink-0 items-center gap-2 rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-3 py-2'>
                  {visibilityMode === 'sets' ? (
                    <IconUserGroup
                      size={16}
                      className='text-[var(--semi-color-warning)]'
                    />
                  ) : (
                    <IconGlobe
                      size={16}
                      className='text-[var(--semi-color-success)]'
                    />
                  )}
                  <div className='min-w-0'>
                    <Text strong>{scopeSummary}</Text>
                    <div>
                      <Text type='tertiary' size='small'>
                        {scopeHelpText}
                      </Text>
                    </div>
                  </div>
                </div>
              </div>

              <div>
                <div className='mb-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between'>
                  <div className='flex items-center gap-2'>
                    <Text strong>{t('可见范围')}</Text>
                    <Tag color='blue' shape='circle' size='small'>
                      {t('保存白名单后生效')}
                    </Tag>
                  </div>
                  <Button
                    type='tertiary'
                    theme='solid'
                    size='small'
                    icon={<IconUserGroup />}
                    onClick={() => setShowSetManager(true)}
                    className='w-full sm:w-auto'
                  >
                    {t('白名单用户集')}
                  </Button>
                </div>
                <RadioGroup
                  value={visibilityMode}
                  onChange={(event) => {
                    const value = event?.target?.value ?? event;
                    setVisibilityMode(value);
                    if (value === 'public') {
                      setSelectedSetIDs([]);
                    }
                  }}
                  type='card'
                  direction={isMobile ? 'vertical' : 'horizontal'}
                  aria-label={t('模型白名单可见范围')}
                  name='model-whitelist-visibility'
                >
                  <Radio value='public' extra={t('所有用户和未登录访问均可见')}>
                    {t('所有人可见')}
                  </Radio>
                  <Radio value='sets' extra={t('登录后命中任一用户集可见')}>
                    {t('指定用户集')}
                  </Radio>
                </RadioGroup>
              </div>

              {visibilityMode === 'sets' ? (
                <div>
                  <div className='mb-2 flex flex-wrap items-center justify-between gap-2'>
                    <label
                      className='text-sm font-medium text-[var(--semi-color-text-0)]'
                      htmlFor='model-whitelist-set-select'
                    >
                      {t('选择白名单用户集')}
                    </label>
                    <Tag
                      color={selectedSetCount > 0 ? 'green' : 'orange'}
                      shape='circle'
                      size='small'
                    >
                      {selectedSetCount > 0
                        ? t('已选择 {{count}} 个用户集', {
                            count: selectedSetCount,
                          })
                        : t('未选择用户集')}
                    </Tag>
                  </div>
                  <Select
                    id='model-whitelist-set-select'
                    value={selectedSetIDs}
                    onChange={(value) =>
                      setSelectedSetIDs(
                        Array.isArray(value) ? uniqIDs(value) : [],
                      )
                    }
                    placeholder={t('请选择一个或多个用户集')}
                    optionList={sets.map((item) => ({
                      label: item.name,
                      value: item.id,
                    }))}
                    multiple
                    filter
                    showClear
                    style={{ width: '100%' }}
                    aria-label={t('选择白名单用户集')}
                  />
                  <Text type='tertiary' size='small' className='mt-1 block'>
                    {t(
                      '通过上方“白名单用户集”按钮维护用户集，这里只选择应用到当前模型的范围。',
                    )}
                  </Text>
                </div>
              ) : (
                <div className='flex items-start gap-2 rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-3 py-2'>
                  <IconInfoCircle
                    size={16}
                    className='mt-0.5 shrink-0 text-[var(--semi-color-text-2)]'
                  />
                  <Text type='tertiary'>
                    {t('未选择用户集时，模型对所有用户和未登录访问保持可见。')}
                  </Text>
                </div>
              )}
            </div>
          </Card>
        </div>
      </Spin>

      <Modal
        title={
          <Space>
            <Avatar size='small' color='blue' className='shrink-0'>
              <IconEyeOpened size={16} />
            </Avatar>
            <Text strong>{t('白名单用户集')}</Text>
          </Space>
        }
        visible={showSetManager}
        onCancel={() => setShowSetManager(false)}
        footer={null}
        width={isMobile ? '96%' : 860}
        closeIcon={null}
        bodyStyle={{
          padding: isMobile ? 12 : 16,
          maxHeight: isMobile ? '72vh' : '68vh',
          overflowY: 'auto',
        }}
      >
        <div className='mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
          <Text type='tertiary' size='small'>
            {t('维护可复用的用户范围，保存白名单时才会应用到当前模型')}
          </Text>
          <Button
            type='primary'
            theme='solid'
            size='small'
            icon={<IconPlus />}
            onClick={() => openEditor()}
            className='w-full sm:w-auto'
          >
            {t('新建用户集')}
          </Button>
        </div>
        <div className='flex gap-2 mb-3 items-stretch'>
          <Input
            prefix={<Search size={14} />}
            placeholder={t('搜索用户集')}
            value={keyword}
            onChange={(value) => setKeyword(value || '')}
            onEnterPress={loadSets}
            size='default'
            className={`${compactSearchClassName} flex-1`}
          />
          <Button
            size='default'
            theme='solid'
            type='tertiary'
            onClick={loadSets}
            className='h-8'
          >
            {t('查询')}
          </Button>
        </div>
        {sets.length > 0 ? (
          <div className={tableSurfaceClassName}>
            <CardTable
              columns={columns}
              dataSource={sets}
              rowKey='id'
              hidePagination
              size='small'
              scroll={{ x: 'max-content' }}
            />
          </div>
        ) : (
          <Empty
            image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
            darkModeImage={
              <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
            }
            description={t('暂无用户集')}
            style={{ padding: 30 }}
          />
        )}
      </Modal>

      <Modal
        title={editingSet?.id ? t('编辑用户集') : t('新建用户集')}
        visible={editing}
        onCancel={closeEditor}
        width={isMobile ? '96%' : 720}
        closeIcon={null}
        className='model-whitelist-edit-modal'
        footer={
          <div className='flex justify-end'>
            <Space>
              <Button onClick={closeEditor}>{t('取消')}</Button>
              <Button
                type='primary'
                theme='solid'
                loading={saving}
                onClick={() => formApiRef.current?.submitForm()}
              >
                {t('保存')}
              </Button>
            </Space>
          </div>
        }
        bodyStyle={{
          padding: isMobile ? 12 : 20,
          maxHeight: isMobile ? '72vh' : '68vh',
          overflowY: 'auto',
        }}
      >
        <Form
          key={`visibility-set-${editingSet?.id ?? 'new'}`}
          initValues={editingSet || {}}
          getFormApi={(api) => (formApiRef.current = api)}
          onSubmit={saveSet}
        >
          <Row>
            <Col span={24}>
              <Form.Input
                field='name'
                label={t('用户集名称')}
                placeholder={t('请输入用户集名称')}
                rules={[{ required: true, message: t('请输入用户集名称') }]}
                showClear
              />
            </Col>
            <Col span={24}>
              <Form.TextArea
                field='description'
                label={t('描述')}
                rows={2}
                showClear
              />
            </Col>
            <Col span={24}>
              <div className='mb-2 flex items-center justify-between'>
                <Text strong>{t('显式用户')}</Text>
                <Button
                  size='small'
                  type='primary'
                  theme='solid'
                  onClick={() => {
                    setShowUserPicker(true);
                    searchPickerUsers(userPickerKeyword);
                  }}
                >
                  {t('选择用户')}
                </Button>
              </div>
              {selectedUsers.length > 0 ? (
                <div className={tableSurfaceClassName}>
                  <CardTable
                    columns={selectedUserColumns}
                    dataSource={selectedUsers}
                    rowKey='id'
                    size='small'
                    scroll={{ x: '100%' }}
                    pagination={{
                      pageSize: 5,
                      total: selectedUsers.length,
                      showSizeChanger: false,
                      size: 'small',
                    }}
                  />
                </div>
              ) : (
                <Empty description={t('暂未选择用户')} />
              )}
            </Col>
            <Col span={24}>
              <Form.Select
                field='user_tags'
                label={t('用户标签')}
                placeholder={t('选择或输入用户标签')}
                multiple
                filter
                allowCreate
                showClear
                optionList={tagOptions}
                style={{ width: '100%' }}
                onChange={requestPreviewRefresh}
              />
            </Col>
            <Col span={24}>
              <Form.Select
                field='user_groups'
                label={t('用户分组')}
                placeholder={t('选择或输入用户分组')}
                multiple
                filter
                allowCreate
                showClear
                optionList={groupOptions}
                style={{ width: '100%' }}
                onChange={requestPreviewRefresh}
              />
            </Col>
          </Row>
          <div className='mt-4 mb-2 flex items-center justify-between gap-2'>
            <Text strong>{t('命中用户预览')}</Text>
            <div className='flex items-stretch gap-2'>
              <Input
                prefix={<Search size={14} />}
                placeholder={t('查询命中用户')}
                value={previewKeyword}
                onChange={(value) => {
                  setPreviewKeyword(value || '');
                  setPreviewPage(1);
                }}
                size='default'
                className={compactSearchClassName}
                style={{ width: isMobile ? '100%' : 200 }}
              />
              <Button
                size='default'
                theme='solid'
                type='tertiary'
                loading={previewing}
                onClick={previewMatchedUsers}
                className='h-8'
              >
                {t('预览')}
              </Button>
            </div>
          </div>
          {previewUsers.length > 0 ? (
            <div className={tableSurfaceClassName}>
              <CardTable
                columns={pickerUserColumns}
                dataSource={pagedPreviewUsers}
                rowKey='id'
                size='small'
                scroll={{ x: '100%' }}
                pagination={{
                  currentPage: previewPage,
                  pageSize: previewPageSize,
                  total: filteredPreviewUsers.length,
                  showSizeChanger: true,
                  pageSizeOptions: [5, 10, 20, 50],
                  size: 'small',
                  onPageChange: setPreviewPage,
                  onPageSizeChange: (size) => {
                    setPreviewPageSize(size);
                    setPreviewPage(1);
                  },
                }}
                empty={<Empty description={t('暂无预览结果')} />}
              />
            </div>
          ) : (
            <Text type='tertiary'>{t('暂无预览结果')}</Text>
          )}
        </Form>
      </Modal>

      <Modal
        title={t('选择用户')}
        visible={showUserPicker}
        onCancel={() => setShowUserPicker(false)}
        onOk={() => {
          setShowUserPicker(false);
        }}
        width={isMobile ? '96%' : 840}
        closeIcon={null}
        bodyStyle={{ padding: isMobile ? '12px' : '16px 20px 12px' }}
      >
        <div className='mb-3 flex flex-col gap-2 sm:flex-row sm:items-stretch'>
          <Input
            prefix={<Search size={14} />}
            placeholder={t('搜索用户名、手机号、邮箱或 ID')}
            value={userPickerKeyword}
            onChange={(value) => setUserPickerKeyword(value || '')}
            onEnterPress={() => searchPickerUsers(userPickerKeyword)}
            size='default'
            className={`${compactSearchClassName} w-full sm:w-[320px]`}
          />
          <Button
            size='default'
            theme='solid'
            type='tertiary'
            onClick={() => searchPickerUsers(userPickerKeyword)}
            className='h-8 w-full sm:w-auto'
          >
            {t('查询')}
          </Button>
        </div>
        <div className={tableSurfaceClassName}>
          <CardTable
            columns={pickerUserColumns}
            dataSource={userPickerUsers}
            rowKey='id'
            loading={userPickerLoading}
            size='small'
            scroll={{ x: '100%' }}
            pagination={{
              currentPage: userPickerPage,
              pageSize: userPickerPageSize,
              total: userPickerTotal,
              showSizeChanger: true,
              pageSizeOptions: [10, 20, 50, 100],
              size: 'small',
              onPageChange: (page) =>
                searchPickerUsers(userPickerKeyword, page),
              onPageSizeChange: (size) => {
                searchPickerUsers(userPickerKeyword, 1, size);
              },
            }}
            rowSelection={{
              selectedRowKeys: selectedUsers.map((user) => user.id),
              onChange: (_, selectedRows) => {
                const visibleIDs = new Set(
                  userPickerUsers.map((user) => user.id),
                );
                const selectedVisibleIDs = new Set(
                  selectedRows.map((user) => user.id),
                );
                const keptUsers = selectedUsers.filter(
                  (user) =>
                    !visibleIDs.has(user.id) || selectedVisibleIDs.has(user.id),
                );
                const keptIDs = new Set(keptUsers.map((user) => user.id));
                const nextUsers = [
                  ...keptUsers,
                  ...selectedRows.filter((user) => !keptIDs.has(user.id)),
                ];
                setSelectedUsers(nextUsers);
              },
            }}
            empty={<Empty description={t('暂无用户')} />}
          />
        </div>
      </Modal>
    </SideSheet>
  );
};

export default ModelWhitelistModal;
