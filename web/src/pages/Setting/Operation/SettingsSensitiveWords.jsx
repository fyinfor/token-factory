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

import React, { useContext, useEffect, useState, useRef } from 'react';
import {
  Button,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Spin,
  Table,
} from '@douyinfe/semi-ui';
import { IconSearch } from '@douyinfe/semi-icons';
import {
  compareObjects,
  API,
  patchStatusData,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../../context/Status';

const defaultInputs = {
  CheckSensitiveEnabled: false,
  CheckSensitiveOnPromptEnabled: false,
  SensitiveWords: '',
  AliyunGuardrailEnabled: false,
  AliyunGuardrailInputEnabled: true,
  AliyunGuardrailOutputEnabled: true,
  AliyunGuardrailVideoEnabled: false,
  AliyunGuardrailHidePlaygroundMediaTabs: false,
  AliyunGuardrailUserIDs: '',
  AliyunGuardrailAccessKeyID: '',
  AliyunGuardrailAccessKeySecret: '',
  AliyunGuardrailRegionID: 'cn-shanghai',
};

const parseGuardrailUserIDs = (value) => {
  const ids = new Set();
  for (const line of String(value || '').split(/[\r\n,]+/)) {
    const normalized = line.trim();
    if (!/^\+?\d+$/.test(normalized)) continue;
    const id = Number(normalized);
    if (Number.isSafeInteger(id) && id > 0) {
      ids.add(id);
    }
  }
  return [...ids];
};

export default function SettingsSensitiveWords(props) {
  const { t } = useTranslation();
  const [statusState, statusDispatch] = useContext(StatusContext);
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState(defaultInputs);
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);
  const [userPickerVisible, setUserPickerVisible] = useState(false);
  const [userSearchKeyword, setUserSearchKeyword] = useState('');
  const [userSearchResults, setUserSearchResults] = useState([]);
  const [userSearchLoading, setUserSearchLoading] = useState(false);
  const [selectedUserIDs, setSelectedUserIDs] = useState([]);
  const userSearchRequestID = useRef(0);

  const searchUsers = async () => {
    const keyword = userSearchKeyword.trim();
    if (!keyword) return;

    const requestID = ++userSearchRequestID.current;
    setUserSearchLoading(true);
    setUserSearchResults([]);
    try {
      const res = await API.get('/api/user/search', {
        params: { keyword, p: 1, page_size: 50 },
      });
      if (requestID !== userSearchRequestID.current) return;
      if (!res.data?.success) {
        showError(res.data?.message || t('搜索用户失败'));
        return;
      }
      setUserSearchResults(res.data.data?.items || []);
    } catch {
      if (requestID === userSearchRequestID.current) {
        showError(t('搜索用户失败'));
      }
    } finally {
      if (requestID === userSearchRequestID.current) {
        setUserSearchLoading(false);
      }
    }
  };

  const openUserPicker = () => {
    setSelectedUserIDs(parseGuardrailUserIDs(inputs.AliyunGuardrailUserIDs));
    setUserSearchKeyword('');
    setUserSearchResults([]);
    setUserSearchLoading(false);
    setUserPickerVisible(true);
  };

  const appendSelectedUserIDs = () => {
    const existingUserIDs = parseGuardrailUserIDs(
      inputs.AliyunGuardrailUserIDs,
    );
    const existingIDSet = new Set(existingUserIDs);
    const additionalUserIDs = selectedUserIDs.filter(
      (id) => !existingIDSet.has(id),
    );
    if (additionalUserIDs.length > 0) {
      const existingValue = String(inputs.AliyunGuardrailUserIDs || '').trim();
      const nextUserIDs = [existingValue, ...additionalUserIDs]
        .filter(Boolean)
        .join(', ');
      setInputs((currentInputs) => ({
        ...currentInputs,
        AliyunGuardrailUserIDs: nextUserIDs,
      }));
      refForm.current?.setValue('AliyunGuardrailUserIDs', nextUserIDs);
    }
    setUserPickerVisible(false);
  };

  const userPickerColumns = [
    { title: 'ID', dataIndex: 'id', width: 88 },
    { title: t('用户名'), dataIndex: 'username', width: 160 },
    {
      title: t('显示名称'),
      dataIndex: 'display_name',
      width: 160,
      render: (value) => value || '-',
    },
    {
      title: t('邮箱'),
      dataIndex: 'email',
      render: (value) => value || '-',
    },
  ];

  function onSubmit() {
    const normalizedInputs = {
      ...inputs,
      AliyunGuardrailAccessKeyID:
        inputs.AliyunGuardrailAccessKeyID?.trim() || '',
      AliyunGuardrailAccessKeySecret:
        inputs.AliyunGuardrailAccessKeySecret?.trim() || '',
      AliyunGuardrailRegionID: inputs.AliyunGuardrailRegionID?.trim() || '',
    };
    if (
      normalizedInputs.AliyunGuardrailAccessKeyID &&
      normalizedInputs.AliyunGuardrailAccessKeyID ===
        normalizedInputs.AliyunGuardrailAccessKeySecret
    ) {
      return showError(t('AccessKey ID 不能与 AccessKey Secret 相同'));
    }
    const updateArray = compareObjects(normalizedInputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof normalizedInputs[item.key] === 'boolean') {
        value = String(normalizedInputs[item.key]);
      } else {
        value = normalizedInputs[item.key];
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        const statusPatch = {
          aliyun_guardrail_hide_playground_media_tabs:
            normalizedInputs.AliyunGuardrailHidePlaygroundMediaTabs,
          aliyun_guardrail_hide_playground_reasoning:
            !!normalizedInputs.AliyunGuardrailEnabled &&
            !!normalizedInputs.AliyunGuardrailOutputEnabled &&
            !normalizedInputs.AliyunGuardrailUserIDs?.trim(),
        };
        const cachedStatus = patchStatusData(statusPatch);
        statusDispatch({
          type: 'set',
          payload: {
            ...(statusState?.status || {}),
            ...cachedStatus,
            ...statusPatch,
          },
        });
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const currentInputs = { ...defaultInputs };
    for (const key of Object.keys(defaultInputs)) {
      if (props.options[key] !== undefined) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current?.setValues(currentInputs);
  }, [props.options]);
  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('屏蔽词过滤设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'CheckSensitiveEnabled'}
                  label={t('启用屏蔽词过滤功能')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) => {
                    setInputs({
                      ...inputs,
                      CheckSensitiveEnabled: value,
                    });
                  }}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'CheckSensitiveOnPromptEnabled'}
                  label={t('启用 Prompt 检查')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      CheckSensitiveOnPromptEnabled: value,
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.TextArea
                  label={t('屏蔽词列表')}
                  extraText={t('一行一个屏蔽词，不需要符号分割')}
                  placeholder={t('一行一个屏蔽词，不需要符号分割')}
                  field={'SensitiveWords'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      SensitiveWords: value,
                    })
                  }
                  style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                  autosize={{ minRows: 6, maxRows: 12 }}
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存屏蔽词过滤设置')}
              </Button>
            </Row>
            <Form.Section text={t('阿里云 AI 安全护栏')}>
              <Row gutter={16}>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Switch
                    field='AliyunGuardrailEnabled'
                    label={t('启用阿里云安全护栏')}
                    onChange={(value) =>
                      setInputs({ ...inputs, AliyunGuardrailEnabled: value })
                    }
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Switch
                    field='AliyunGuardrailInputEnabled'
                    label={t('审核用户输入')}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        AliyunGuardrailInputEnabled: value,
                      })
                    }
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Switch
                    field='AliyunGuardrailOutputEnabled'
                    label={t('审核模型非流式输出')}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        AliyunGuardrailOutputEnabled: value,
                      })
                    }
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Switch
                    field='AliyunGuardrailVideoEnabled'
                    label={t('审核视频输出（异步）')}
                    extraText={t('审核通过后才会返回视频链接')}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        AliyunGuardrailVideoEnabled: value,
                      })
                    }
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Switch
                    field='AliyunGuardrailHidePlaygroundMediaTabs'
                    label={t('操练场隐藏图片和视频模型')}
                    extraText={t('开启后操练场将不显示图片模型和视频模型 Tab')}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        AliyunGuardrailHidePlaygroundMediaTabs: value,
                      })
                    }
                  />
                </Col>
              </Row>
              <Row>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.TextArea
                    field='AliyunGuardrailUserIDs'
                    label={t('安全护栏适用用户 ID')}
                    extraText={t(
                      '留空则对所有用户启用；填写后仅名单内用户启用，多个用户 ID 使用逗号分隔',
                    )}
                    placeholder={t('用户 ID 以逗号分隔，例如：1,2,3')}
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        AliyunGuardrailUserIDs: value,
                      })
                    }
                    style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                    autosize={{ minRows: 3, maxRows: 8 }}
                  />
                  <Button
                    size='small'
                    icon={<IconSearch />}
                    onClick={openUserPicker}
                    style={{ marginTop: 8 }}
                  >
                    {t('选择用户')}
                  </Button>
                </Col>
              </Row>
              <Row gutter={16}>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Input
                    field='AliyunGuardrailAccessKeyID'
                    label='AccessKey ID'
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        AliyunGuardrailAccessKeyID: value,
                      })
                    }
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Input
                    field='AliyunGuardrailAccessKeySecret'
                    mode='password'
                    label='AccessKey Secret'
                    onChange={(value) =>
                      setInputs({
                        ...inputs,
                        AliyunGuardrailAccessKeySecret: value,
                      })
                    }
                  />
                </Col>
                <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                  <Form.Input
                    field='AliyunGuardrailRegionID'
                    label={t('地域')}
                    extraText={t('默认 cn-shanghai')}
                    onChange={(value) =>
                      setInputs({ ...inputs, AliyunGuardrailRegionID: value })
                    }
                  />
                </Col>
              </Row>
              <Row>
                <Button size='default' onClick={onSubmit}>
                  {t('保存阿里云安全护栏设置')}
                </Button>
              </Row>
            </Form.Section>
          </Form.Section>
        </Form>
        <Modal
          title={t('选择用户')}
          visible={userPickerVisible}
          onCancel={() => {
            userSearchRequestID.current += 1;
            setUserSearchLoading(false);
            setUserPickerVisible(false);
          }}
          onOk={appendSelectedUserIDs}
          okText={t('确定')}
          width={760}
          bodyStyle={{ maxHeight: '65vh', overflowY: 'auto' }}
        >
          <div className='flex gap-2 mb-3'>
            <Input
              prefix={<IconSearch />}
              placeholder={t('支持搜索用户的 ID、用户名、显示名称和邮箱地址')}
              value={userSearchKeyword}
              onChange={(value) => setUserSearchKeyword(value || '')}
              onEnterPress={searchUsers}
              style={{ flex: 1 }}
            />
            <Button
              type='primary'
              theme='solid'
              onClick={searchUsers}
              loading={userSearchLoading}
              disabled={!userSearchKeyword.trim()}
            >
              {t('查询')}
            </Button>
          </div>
          <Table
            columns={userPickerColumns}
            dataSource={userSearchResults}
            rowKey='id'
            loading={userSearchLoading}
            pagination={false}
            size='small'
            scroll={{ x: '100%' }}
            rowSelection={{
              selectedRowKeys: selectedUserIDs,
              onChange: (selectedRowKeys) => {
                const visibleIDs = new Set(
                  userSearchResults.map((user) => user.id),
                );
                const nextVisibleIDs = selectedRowKeys
                  .map((id) => Number(id))
                  .filter((id) => Number.isSafeInteger(id) && id > 0);
                setSelectedUserIDs((previousIDs) => [
                  ...previousIDs.filter((id) => !visibleIDs.has(id)),
                  ...nextVisibleIDs,
                ]);
              },
            }}
          />
        </Modal>
      </Spin>
    </>
  );
}
