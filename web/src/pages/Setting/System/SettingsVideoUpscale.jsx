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

import React, { useEffect, useRef, useState } from 'react';
import { Banner, Button, Form, Input } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, compareObjects, showError, showSuccess, showWarning } from '../../../helpers';

const BASE_INPUTS = {
  'video_upscale_setting.secret_id': '',
  'video_upscale_setting.secret_key': '',
  'video_upscale_setting.output_path': '',
  'video_upscale_setting.templates': '[]',
};

function parseTemplates(raw) {
  if (Array.isArray(raw)) return raw;
  if (typeof raw !== 'string' || !raw.trim()) return [];
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : [];
  } catch (e) {
    return [];
  }
}

export default function SettingsVideoUpscale(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState(BASE_INPUTS);
  const [inputsRow, setInputsRow] = useState(BASE_INPUTS);
  const [templates, setTemplates] = useState([]);
  const refForm = useRef();

  useEffect(() => {
    const next = { ...BASE_INPUTS };
    Object.keys(BASE_INPUTS).forEach((key) => {
      if (props.options?.[key] !== undefined && props.options?.[key] !== null) {
        next[key] = props.options[key];
      }
    });
    setInputs(next);
    setInputsRow(next);
    setTemplates(parseTemplates(next['video_upscale_setting.templates']));
    refForm.current?.setValues(next, { isOverride: true });
  }, [props.options]);

  const save = async () => {
    const snapshot = {
      ...inputs,
      'video_upscale_setting.templates': JSON.stringify(
        templates
          .map((item) => ({
            id: Number(item?.id) || 0,
            name: String(item?.name || '').trim(),
          }))
          .filter((item) => item.id > 0 && item.name),
      ),
    };
    const changes = compareObjects(snapshot, inputsRow);
    if (!changes.length) {
      showWarning(t('当前配置没有修改'));
      return;
    }
    setLoading(true);
    try {
      const res = await Promise.all(
        changes.map((item) =>
          API.put('/api/option/', {
            key: item.key,
            value: snapshot[item.key],
          }),
        ),
      );
      const failed = res.find((item) => item?.data?.success === false);
      if (failed) {
        showError(failed?.data?.message || t('保存失败，请重试'));
        return;
      }
      showSuccess(t('保存成功'));
      if (typeof props.refresh === 'function') {
        await props.refresh();
      }
    } catch (e) {
      showError(t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Banner
        type='info'
        description={t(
          '配置腾讯云 MPS 视频超分。SecretId/SecretKey/输出路径任一为空时，全站不启用超分，视频任务走原有流程。',
        )}
        style={{ marginBottom: 16 }}
      />
      <Form
        getFormApi={(formAPI) => (refForm.current = formAPI)}
        onValueChange={(values) => setInputs((prev) => ({ ...prev, ...values }))}
      >
        <Form.Input
          field="['video_upscale_setting.secret_id']"
          label='SecretId'
          placeholder={t('腾讯云 SecretId')}
          showClear
        />
        <Form.Input
          field="['video_upscale_setting.secret_key']"
          label='SecretKey'
          mode='password'
          placeholder={t('腾讯云 SecretKey')}
          showClear
        />
        <Form.Input
          field="['video_upscale_setting.output_path']"
          label={t('输出路径')}
          placeholder='https://bucket.cos.ap-guangzhou.myqcloud.com/upscale/'
          extraText={t(
            'COS 输出路径，格式：https://<bucket>.cos.<region>.myqcloud.com/<prefix>/',
          )}
          showClear
        />
      </Form>
      <div className='mt-4 mb-2 font-medium'>{t('超分模版')}</div>
      <div className='text-xs text-gray-500 mb-3'>
        {t('渠道超分规则将从此列表选择模版。ID 为腾讯云 MPS 转码模版 Definition。')}
      </div>
      {templates.map((item, index) => (
        <div
          key={`upscale-tpl-${index}`}
          className='flex flex-wrap items-center gap-2 mb-2'
        >
          <Input
            value={item.id ? String(item.id) : ''}
            placeholder={t('模版 ID')}
            style={{ width: 180 }}
            onChange={(value) => {
              const next = [...templates];
              next[index] = { ...next[index], id: value };
              setTemplates(next);
            }}
          />
          <Input
            value={item.name || ''}
            placeholder={t('模版名称')}
            style={{ width: 220 }}
            onChange={(value) => {
              const next = [...templates];
              next[index] = { ...next[index], name: value };
              setTemplates(next);
            }}
          />
          <Button
            type='danger'
            theme='borderless'
            onClick={() =>
              setTemplates(templates.filter((_, i) => i !== index))
            }
          >
            {t('删除')}
          </Button>
        </div>
      ))}
      <Button
        theme='light'
        onClick={() => setTemplates([...templates, { id: '', name: '' }])}
      >
        {t('添加模版')}
      </Button>
      <div className='mt-4'>
        <Button type='primary' loading={loading} onClick={save}>
          {t('保存视频超分配置')}
        </Button>
      </div>
    </div>
  );
}
