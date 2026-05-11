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
  Input,
  Modal,
  Select,
  Space,
  Table,
  TextArea,
  Typography,
  Upload,
} from '@douyinfe/semi-ui';
import { Plus, Trash2, Edit2 } from 'lucide-react';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const OPTION_KEY = 'HomeBannerSlides';

const emptySlide = () => ({
  image_url: '',
  badge: 'NEW',
  title: '',
  subtitle: '',
  button_text: '',
  target_model: '',
});

function parseSlides(raw) {
  if (!raw || typeof raw !== 'string') return [];
  try {
    const v = JSON.parse(raw);
    return Array.isArray(v) ? v : [];
  } catch {
    return [];
  }
}

const { Text } = Typography;

const SettingsHomeBanner = ({ options, refresh }) => {
  const { t } = useTranslation();
  const [slides, setSlides] = useState([]);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [modelOptions, setModelOptions] = useState([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editIndex, setEditIndex] = useState(null);
  const [form, setForm] = useState(emptySlide());

  const raw = options?.[OPTION_KEY] ?? '';

  useEffect(() => {
    setSlides(parseSlides(raw));
  }, [raw]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await API.get('/api/pricing');
        const { success, data } = res.data || {};
        if (!success || !Array.isArray(data) || cancelled) return;
        const opts = data
          .map((m) => m?.model_name)
          .filter(Boolean)
          .sort((a, b) => String(a).localeCompare(String(b)))
          .map((name) => ({ label: name, value: name }));
        if (!cancelled) setModelOptions(opts);
      } catch {
        /* ignore */
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const persist = useCallback(
    async (nextSlides) => {
      setSaving(true);
      try {
        const value = JSON.stringify(nextSlides);
        const res = await API.put('/api/option/', {
          key: OPTION_KEY,
          value,
        });
        const { success, message } = res.data || {};
        if (!success) {
          showError(message || t('保存失败'));
          return;
        }
        showSuccess(t('保存成功'));
        setSlides(nextSlides);
        if (refresh) await refresh();
      } catch (e) {
        showError(e?.response?.data?.message || t('保存失败'));
      } finally {
        setSaving(false);
      }
    },
    [refresh, t],
  );

  const openAdd = useCallback(() => {
    setEditIndex(null);
    setForm(emptySlide());
    setModalOpen(true);
  }, []);

  const openEdit = useCallback((idx) => {
    setEditIndex(idx);
    setForm({ ...emptySlide(), ...slides[idx] });
    setModalOpen(true);
  }, [slides]);

  const removeAt = useCallback(
    async (idx) => {
      const next = slides.filter((_, i) => i !== idx);
      await persist(next);
    },
    [slides, persist],
  );

  const handleModalOk = async () => {
    if (!form.title?.trim()) {
      showError(t('首页广告标题必填'));
      return;
    }
    let next;
    if (editIndex == null) {
      next = [...slides, { ...form }];
    } else {
      next = slides.map((s, i) => (i === editIndex ? { ...form } : s));
    }
    await persist(next);
    setModalOpen(false);
  };

  const handleUpload = useCallback(
    async ({ file, onSuccess, onError }) => {
      const inst = file.fileInstance || file;
      if (!inst) {
        onError(new Error('no file'));
        return;
      }
      setUploading(true);
      const fd = new FormData();
      fd.append('file', inst);
      try {
        const res = await API.post('/api/oss/upload', fd, {
          skipErrorHandler: true,
        });
        const { success, message, data } = res.data || {};
        const url = data?.url;
        if (!success || !url) {
          showError(message || t('上传失败'));
          onError(new Error(message));
          return;
        }
        setForm((f) => ({ ...f, image_url: url }));
        onSuccess(data);
        showSuccess(t('上传成功'));
      } catch (e) {
        const msg =
          e?.response?.data?.message ||
          e?.message ||
          t('上传失败，请确认已启用 OSS 并完成配置');
        showError(msg);
        onError(e);
      } finally {
        setUploading(false);
      }
    },
    [t],
  );

  const columns = [
    {
      title: t('广告缩略图'),
      width: 110,
      render: (_, record) =>
        record.image_url ? (
          <img
            src={record.image_url}
            alt=''
            className='h-12 w-20 rounded-md object-cover border border-semi-color-border'
          />
        ) : (
          <Text type='tertiary'>{t('无图')}</Text>
        ),
    },
    { title: t('广告标题列'), dataIndex: 'title' },
    { title: t('跳转模型'), dataIndex: 'target_model' },
    {
      title: t('操作'),
      width: 180,
      render: (_, record) => (
        <Space>
          <Button
            type='tertiary'
            theme='borderless'
            icon={<Edit2 size={16} />}
            onClick={() => openEdit(record._idx)}
          >
            {t('编辑')}
          </Button>
          <Button
            type='danger'
            theme='borderless'
            icon={<Trash2 size={16} />}
            onClick={() => removeAt(record._idx)}
          >
            {t('删除')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Typography.Title heading={6} className='!mb-2'>
        {t('首页广告轮播说明')}
      </Typography.Title>
      <Text type='tertiary' className='!block !mb-4'>
        {t('首页广告轮播提示')}
      </Text>
      <Button icon={<Plus size={16} />} type='primary' onClick={openAdd}>
        {t('添加广告幻灯片')}
      </Button>

      <Table
        className='mt-4'
        columns={columns}
        dataSource={slides.map((s, i) => ({
          ...s,
          _idx: i,
          key: String(i),
        }))}
        pagination={false}
      />

      <Modal
        title={
          editIndex == null ? t('添加广告幻灯片') : t('编辑广告幻灯片')
        }
        visible={modalOpen}
        onOk={handleModalOk}
        onCancel={() => setModalOpen(false)}
        confirmLoading={saving}
        width={560}
      >
        <div className='space-y-3'>
          <div>
            <Text strong className='!block !mb-2'>
              {t('广告图片')}
            </Text>
            <Upload
              listType='picture'
              maxCount={1}
              customRequest={handleUpload}
              showUploadList={{ showRemoveIcon: true }}
              onRemove={() => {
                setForm((f) => ({ ...f, image_url: '' }));
                return true;
              }}
              fileList={
                form.image_url
                  ? [
                      {
                        uid: '1',
                        name: 'banner',
                        status: 'success',
                        url: form.image_url,
                      },
                    ]
                  : []
              }
              accept='image/*'
              disabled={uploading}
            />
            <Text type='tertiary' size='small' className='!mt-1 !block'>
              {t('首页广告图片说明')}
            </Text>
          </div>
          <div>
            <Text strong className='!block !mb-1'>
              {t('角标文案')}
            </Text>
            <Input
              value={form.badge}
              onChange={(v) => setForm((f) => ({ ...f, badge: v }))}
            />
          </div>
          <div>
            <Text strong className='!block !mb-1'>
              {t('广告标题列')}
            </Text>
            <Input
              value={form.title}
              onChange={(v) => setForm((f) => ({ ...f, title: v }))}
            />
          </div>
          <div>
            <Text strong className='!block !mb-1'>
              {t('广告副标题')}
            </Text>
            <TextArea
              value={form.subtitle}
              onChange={(v) => setForm((f) => ({ ...f, subtitle: v }))}
              rows={2}
            />
          </div>
          <div>
            <Text strong className='!block !mb-1'>
              {t('广告按钮文案')}
            </Text>
            <Input
              value={form.button_text}
              onChange={(v) => setForm((f) => ({ ...f, button_text: v }))}
              placeholder={t('立即体验')}
            />
          </div>
          <div>
            <Text strong className='!block !mb-1'>
              {t('跳转模型')}
            </Text>
            <Select
              value={form.target_model || undefined}
              onChange={(v) =>
                setForm((f) => ({ ...f, target_model: v || '' }))
              }
              optionList={modelOptions}
              filter
              showClear
              placeholder={t('首页广告模型占位')}
              className='w-full'
            />
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default SettingsHomeBanner;
