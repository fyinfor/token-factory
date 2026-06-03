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

import React, { useMemo, useState } from 'react';
import {
  Button,
  Modal,
  Space,
  Table,
  Typography,
  Upload,
} from '@douyinfe/semi-ui';
import { IconDownload, IconUpload } from '@douyinfe/semi-icons';
import { API, showError, showSuccess } from '../../../../helpers';

const { Text } = Typography;

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

const ImportUsersModal = ({ visible, onCancel, refresh, t, language }) => {
  const [file, setFile] = useState(null);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);

  const failureColumns = useMemo(
    () => [
      { title: t('行号'), dataIndex: 'row', width: 70 },
      { title: t('用户名'), dataIndex: 'username', width: 130 },
      { title: t('邮箱'), dataIndex: 'email', width: 180 },
      { title: t('手机号'), dataIndex: 'phone', width: 130 },
      { title: t('标签'), dataIndex: 'tags', width: 160 },
      { title: t('是否为学员'), dataIndex: 'is_student', width: 110 },
      { title: t('学员奖励金额'), dataIndex: 'student_reward_amount', width: 130 },
      { title: t('是否为代理'), dataIndex: 'is_agent', width: 110 },
      { title: t('失败原因'), dataIndex: 'reason', width: 260 },
    ],
    [t],
  );

  const handleClose = () => {
    if (loading) return;
    setFile(null);
    setResult(null);
    onCancel();
  };

  const downloadTemplate = async () => {
    try {
      const res = await API.get('/api/user/import/template', {
        responseType: 'blob',
        disableDuplicate: true,
        params: { lang: language },
      });
      downloadBlob(res.data, 'user_import_template.xlsx');
    } catch (error) {
      showError(t('下载模板失败'));
    }
  };

  const submitImport = async () => {
    if (!file) {
      showError(t('请选择 Excel 文件'));
      return;
    }
    const formData = new FormData();
    formData.append('file', file);
    setLoading(true);
    try {
      const res = await API.post('/api/user/import', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        params: { lang: language },
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setResult(data);
      if ((data?.created || 0) > 0) {
        refresh?.();
      }
      showSuccess(
        t('导入完成：成功 {{created}} 个，失败 {{failed}} 个', {
          created: data?.created || 0,
          failed: data?.failed || 0,
        }),
      );
    } catch (error) {
      showError(t('导入失败，请检查 Excel 文件'));
    } finally {
      setLoading(false);
    }
  };

  const downloadFailures = async () => {
    if (!result?.failure_id) return;
    try {
      const res = await API.get(
        `/api/user/import/failures/${result.failure_id}`,
        {
          responseType: 'blob',
          disableDuplicate: true,
        },
      );
      downloadBlob(res.data, 'user_import_failures.xlsx');
    } catch (error) {
      showError(t('下载失败列表失败，请重新导入'));
    }
  };

  const failures = result?.failures || [];

  return (
    <Modal
      title={t('批量导入用户')}
      visible={visible}
      onCancel={handleClose}
      footer={
        <Space>
          <Button
            icon={<IconDownload />}
            onClick={downloadTemplate}
            disabled={loading}
          >
            {t('下载模板')}
          </Button>
          <Button onClick={handleClose} disabled={loading}>
            {t('关闭')}
          </Button>
          <Button
            theme='solid'
            type='primary'
            icon={<IconUpload />}
            loading={loading}
            onClick={submitImport}
          >
            {t('开始导入')}
          </Button>
        </Space>
      }
      width={820}
      maskClosable={false}
    >
      <div className='space-y-4'>
        <div className='flex flex-col gap-2'>
          <Text type='tertiary'>
            {t(
              '请使用模板填写用户信息。用户名和密码必填；标签可用逗号、分号或竖线分隔。',
            )}
          </Text>
          <Upload
            accept='.xlsx,.xlsm'
            draggable
            dragIcon={<IconUpload />}
            dragMainText={t('点击上传文件或拖拽 Excel 文件到这里')}
            dragSubText={t('仅支持 .xlsx / .xlsm 文件，单次上传 1 个文件')}
            beforeUpload={() => false}
            limit={1}
            fileList={file ? [{ uid: 'user-import-file', name: file.name }] : []}
            onChange={({ fileList }) => {
              const nextFile = fileList?.[0]?.fileInstance || null;
              setFile(nextFile);
              setResult(null);
            }}
            onRemove={() => {
              setFile(null);
              setResult(null);
            }}
          />
        </div>

        {result && (
          <div className='space-y-3'>
            <Space>
              <Text strong>
                {t('总行数')}: {result.total || 0}
              </Text>
              <Text type='success'>
                {t('成功')}: {result.created || 0}
              </Text>
              <Text type={result.failed > 0 ? 'danger' : 'tertiary'}>
                {t('失败')}: {result.failed || 0}
              </Text>
              {result.failed > 0 && (
                <Button
                  size='small'
                  icon={<IconDownload />}
                  onClick={downloadFailures}
                >
                  {t('下载失败列表')}
                </Button>
              )}
            </Space>
            {failures.length > 0 && (
              <Table
                size='small'
                columns={failureColumns}
                dataSource={failures.map((item) => ({
                  ...item,
                  key: item.row,
                }))}
                pagination={false}
                scroll={{ y: 260 }}
              />
            )}
          </div>
        )}
      </div>
    </Modal>
  );
};

export default ImportUsersModal;
