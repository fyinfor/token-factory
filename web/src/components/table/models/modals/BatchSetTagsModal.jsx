import React, { useMemo, useState } from 'react';
import { Modal, Form, Typography } from '@douyinfe/semi-ui';

const DEFAULT_TAGS = ['文本', '视频', '图片'];

const normalizeTags = (tags = []) => {
  const seen = new Set();
  const list = [];
  tags.forEach((item) => {
    const name = String(item || '').trim();
    if (!name || seen.has(name)) return;
    seen.add(name);
    list.push(name);
  });
  return list;
};

const BatchSetTagsModal = ({
  visible,
  onClose,
  onSubmit,
  selectedCount = 0,
  tagOptions = [],
  t,
}) => {
  const [formApi, setFormApi] = useState(null);
  const [loading, setLoading] = useState(false);

  const mergedTagOptions = useMemo(() => {
    return normalizeTags([...DEFAULT_TAGS, ...tagOptions]).map((tag) => ({
      label: tag,
      value: tag,
    }));
  }, [tagOptions]);

  const handleOk = async () => {
    const values = formApi?.getValues() || {};
    const tags = normalizeTags(values.tags || []);
    const mode = values.mode || 'add';
    if (tags.length === 0) {
      return;
    }
    setLoading(true);
    try {
      const done = await onSubmit?.({ tags, mode });
      if (done) {
        formApi?.reset();
        onClose?.();
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={t('批量设置标签')}
      visible={visible}
      onCancel={onClose}
      onOk={handleOk}
      okButtonProps={{ loading }}
      maskClosable={false}
    >
      <Form
        initValues={{ tags: DEFAULT_TAGS, mode: 'add' }}
        getFormApi={setFormApi}
      >
        <Form.Select
          field='tags'
          label={t('标签')}
          placeholder={t('请选择或输入标签')}
          multiple
          allowCreate
          filter
          optionList={mergedTagOptions}
          rules={[{ required: true, message: t('请至少选择一个标签') }]}
          style={{ width: '100%' }}
        />
        <div style={{ marginBottom: 12 }}>
          <Typography.Text type='tertiary'>
            {t('默认标签「文本 / 视频 / 图片」会保留在选项中，可额外新增标签。')}
          </Typography.Text>
        </div>
        <div style={{ marginBottom: 12 }}>
          <Typography.Text type='tertiary'>
            {t('已选择 {{count}} 个模型', { count: selectedCount })}
          </Typography.Text>
        </div>
        <Form.RadioGroup
          field='mode'
          type='button'
          options={[
            { label: t('在原标签基础上添加（去重）'), value: 'add' },
            { label: t('直接替换现有标签'), value: 'replace' },
          ]}
        />
      </Form>
    </Modal>
  );
};

export default BatchSetTagsModal;
