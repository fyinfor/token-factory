import React, { useRef, useState } from 'react';
import {
  Button,
  Input,
  Modal,
  Table,
  TextArea,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import { API, showError } from '../../helpers';
import { useTranslation } from 'react-i18next';

export const parseUserIDs = (value) => {
  const ids = new Set();
  String(value || '')
    .split(/[\r\n,]+/)
    .forEach((item) => {
      const normalized = item.trim();
      if (!/^\+?\d+$/.test(normalized)) return;
      const id = Number(normalized);
      if (Number.isSafeInteger(id) && id > 0) ids.add(id);
    });
  return [...ids];
};

export default function UserIDsInput({
  value,
  onChange,
  label,
  extraText,
  disabled = false,
}) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [results, setResults] = useState([]);
  const [selected, setSelected] = useState([]);
  const [loading, setLoading] = useState(false);
  const requestID = useRef(0);

  const searchUsers = async () => {
    const query = keyword.trim();
    if (!query) return;

    const id = ++requestID.current;
    setLoading(true);
    setResults([]);
    try {
      const response = await API.get('/api/user/search', {
        params: { keyword: query, p: 1, page_size: 50 },
      });
      if (id !== requestID.current) return;
      if (!response.data?.success) {
        showError(response.data?.message || t('搜索用户失败'));
        return;
      }
      setResults(
        Array.isArray(response.data?.data?.items)
          ? response.data.data.items
          : [],
      );
    } catch {
      if (id === requestID.current) showError(t('搜索用户失败'));
    } finally {
      if (id === requestID.current) setLoading(false);
    }
  };

  const openPicker = () => {
    setSelected(parseUserIDs(value));
    setKeyword('');
    setResults([]);
    setLoading(false);
    setVisible(true);
  };

  const closePicker = () => {
    requestID.current += 1;
    setLoading(false);
    setVisible(false);
  };

  const appendSelected = () => {
    const existing = parseUserIDs(value);
    const existingSet = new Set(existing);
    const additional = selected.filter((id) => !existingSet.has(id));
    if (additional.length > 0) {
      const current = String(value || '').trim();
      onChange?.([current, ...additional].filter(Boolean).join(', '));
    }
    setVisible(false);
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 88 },
    { title: t('用户名'), dataIndex: 'username', width: 160 },
    {
      title: t('显示名称'),
      dataIndex: 'display_name',
      width: 160,
      render: (displayName) => displayName || '-',
    },
    {
      title: t('邮箱'),
      dataIndex: 'email',
      render: (email) => email || '-',
    },
  ];

  return (
    <>
      <div style={{ maxWidth: 420 }}>
        <Typography.Text strong>{label}</Typography.Text>
        <TextArea
          value={String(value || '')}
          onChange={(nextValue) => onChange?.(nextValue || '')}
          placeholder={t('用户 ID 以逗号分隔，例如：1,2,3')}
          autosize={{ minRows: 3, maxRows: 8 }}
          disabled={disabled}
          style={{
            display: 'block',
            width: '100%',
            marginTop: 6,
            fontFamily: 'JetBrains Mono, Consolas',
          }}
        />
        {extraText && (
          <Typography.Text
            type='tertiary'
            size='small'
            style={{ display: 'block', marginTop: 4 }}
          >
            {extraText}
          </Typography.Text>
        )}
        <Button
          size='small'
          icon={<IconSearch />}
          onClick={openPicker}
          disabled={disabled}
          style={{ marginTop: 8 }}
        >
          {t('选择用户')}
        </Button>
      </div>

      <Modal
        title={t('选择用户')}
        visible={visible}
        width={760}
        onCancel={closePicker}
        onOk={appendSelected}
        okText={t('确定')}
        bodyStyle={{ maxHeight: '65vh', overflowY: 'auto' }}
      >
        <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
          <Input
            prefix={<IconSearch />}
            value={keyword}
            onChange={(nextValue) => setKeyword(nextValue || '')}
            onEnterPress={searchUsers}
            placeholder={t('支持搜索用户的 ID、用户名、显示名称和邮箱地址')}
            style={{ flex: 1 }}
          />
          <Button
            type='primary'
            theme='solid'
            loading={loading}
            disabled={!keyword.trim()}
            onClick={searchUsers}
          >
            {t('查询')}
          </Button>
        </div>
        <Table
          columns={columns}
          dataSource={results}
          rowKey='id'
          loading={loading}
          pagination={false}
          size='small'
          scroll={{ x: '100%' }}
          rowSelection={{
            selectedRowKeys: selected,
            onChange: (selectedRowKeys) => {
              const visibleIDs = new Set(results.map((user) => user.id));
              const nextVisibleIDs = selectedRowKeys
                .map(Number)
                .filter((id) => Number.isSafeInteger(id) && id > 0);
              setSelected((previousIDs) => [
                ...previousIDs.filter((id) => !visibleIDs.has(id)),
                ...nextVisibleIDs,
              ]);
            },
          }}
        />
      </Modal>
    </>
  );
}
