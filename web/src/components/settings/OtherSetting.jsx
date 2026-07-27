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

import React, { useContext, useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Row,
  Modal,
  Popconfirm,
  Space,
  Card,
  Tabs,
  Table,
  Upload,
} from '@douyinfe/semi-ui';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  timestamp2string,
  getSystemName,
} from '../../helpers';
import { marked } from 'marked';
import { useTranslation } from 'react-i18next';
import { languageSelectOptions } from '../../i18n/language';
import { StatusContext } from '../../context/Status';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import MarkdownRenderer from '../common/markdown/MarkdownRenderer';
import SettingsSeedance from './SettingsSeedance';
import HomeFooterCertificatesSetting from './HomeFooterCertificatesSetting';
import LegalContentEditor from './LegalContentEditor';
import defaultUserAgreementHtml from '../../content/legal/user-agreement.html?raw';
import defaultPrivacyPolicyHtml from '../../content/legal/privacy-policy.html?raw';

const LEGAL_USER_AGREEMENT_KEY = 'legal.user_agreement';
const LEGAL_USER_AGREEMENT_FORMAT_KEY = 'legal.user_agreement_format';
const LEGAL_PRIVACY_POLICY_KEY = 'legal.privacy_policy';
const LEGAL_PRIVACY_POLICY_FORMAT_KEY = 'legal.privacy_policy_format';
const DEFAULT_LEGAL_FORMAT = 'html';
const DOCS_CONFIG_KEYS = [
  'DocsBrandName',
  'DocsSiteNameEn',
  'DocsSiteNameZh',
  'DocsSiteNameJa',
  'DocsLogoUrl',
  'DocsHomeUrl',
  'DocsGithubUrl',
  'DocsMetaKeywords',
  'DocsBusinessPhone',
  'DocsBusinessPhoneHref',
  'DocsBusinessWorkTimeZh',
  'DocsBusinessWorkTimeEn',
  'DocsBusinessWorkTimeJa',
  'DocsBusinessWechatQrUrl',
];

const docsImagePreviewStyle = {
  width: 96,
  height: 96,
  border: '1px solid var(--semi-color-border)',
  borderRadius: 8,
  objectFit: 'contain',
  background: 'var(--semi-color-fill-0)',
  cursor: 'zoom-in',
};

const docsImageEmptyStyle = {
  ...docsImagePreviewStyle,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  color: 'var(--semi-color-text-2)',
  fontSize: 12,
  cursor: 'default',
};

const formatChangelogDate = (value) => {
  if (!value) {
    return '';
  }
  if (value instanceof Date) {
    const year = value.getFullYear();
    const month = String(value.getMonth() + 1).padStart(2, '0');
    const day = String(value.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }
  const str = String(value).trim();
  if (/^\d{4}-\d{2}-\d{2}/.test(str)) {
    return str.slice(0, 10);
  }
  return str;
};

const OtherSetting = ({ activeSection = 'system-info' }) => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
    Notice: '',
    'general_setting.default_site_language': 'zh-CN',
    [LEGAL_USER_AGREEMENT_KEY]: defaultUserAgreementHtml,
    [LEGAL_USER_AGREEMENT_FORMAT_KEY]: DEFAULT_LEGAL_FORMAT,
    [LEGAL_PRIVACY_POLICY_KEY]: defaultPrivacyPolicyHtml,
    [LEGAL_PRIVACY_POLICY_FORMAT_KEY]: DEFAULT_LEGAL_FORMAT,
    SystemName: '',
    SystemNameEn: '',
    Logo: '',
    DocsBrandName: '',
    DocsSiteNameEn: '',
    DocsSiteNameZh: '',
    DocsSiteNameJa: '',
    DocsLogoUrl: '',
    DocsHomeUrl: '',
    DocsGithubUrl: '',
    DocsMetaKeywords: '',
    DocsBusinessPhone: '',
    DocsBusinessPhoneHref: '',
    DocsBusinessWorkTimeZh: '',
    DocsBusinessWorkTimeEn: '',
    DocsBusinessWorkTimeJa: '',
    DocsBusinessWechatQrUrl: '',
    Footer: '',
    About: '',
    AboutEn: '',
    HomePageContent: '',
  });
  let [loading, setLoading] = useState(false);
  const [showUpdateModal, setShowUpdateModal] = useState(false);
  const [statusState] = useContext(StatusContext);
  const [changelogEntries, setChangelogEntries] = useState([]);
  const [changelogTotal, setChangelogTotal] = useState(0);
  const [changelogPage, setChangelogPage] = useState(1);
  const [changelogPageSize, setChangelogPageSize] = useState(10);
  const [changelogLoading, setChangelogLoading] = useState(false);
  const [changelogModalVisible, setChangelogModalVisible] = useState(false);
  const [editingChangelog, setEditingChangelog] = useState(null);
  const [changelogForm, setChangelogForm] = useState({
    date: '',
    content: '',
  });
  const [updateData, setUpdateData] = useState({
    tag_name: '',
    content: '',
  });
  const [docsImagePreview, setDocsImagePreview] = useState({
    visible: false,
    url: '',
    title: '',
  });

  const updateOption = async (key, value) => {
    setLoading(true);
    const res = await API.put('/api/option/', {
      key,
      value,
    });
    const { success, message } = res.data;
    if (success) {
      setInputs((inputs) => ({ ...inputs, [key]: value }));
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const updateOptions = async (entries) => {
    setLoading(true);
    try {
      const results = await Promise.all(
        entries.map(([key, value]) =>
          API.put('/api/option/', {
            key,
            value,
          }),
        ),
      );
      const failed = results.find((res) => !res.data?.success);
      if (failed) {
        showError(failed.data?.message || t('设置保存失败'));
        return false;
      }
      setInputs((inputs) => {
        const next = { ...inputs };
        entries.forEach(([key, value]) => {
          next[key] = value;
        });
        return next;
      });
      return true;
    } finally {
      setLoading(false);
    }
  };

  const [loadingInput, setLoadingInput] = useState({
    Notice: false,
    [LEGAL_USER_AGREEMENT_KEY]: false,
    [LEGAL_PRIVACY_POLICY_KEY]: false,
    SystemName: false,
    Logo: false,
    DocsConfig: false,
    DocsLogoUrl: false,
    DocsBusinessWechatQrUrl: false,
    HomePageContent: false,
    About: false,
    AboutEn: false,
    Footer: false,
    CheckUpdate: false,
    Changelog: false,
    DefaultSiteLanguage: false,
  });
  const handleInputChange = async (value, e) => {
    const name = e.target.id;
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };
  const loadChangelogs = async (
    targetPage = changelogPage,
    targetPageSize = changelogPageSize,
  ) => {
    setChangelogLoading(true);
    try {
      const res = await API.get('/api/changelog/admin/', {
        params: {
          p: targetPage,
          page_size: targetPageSize,
        },
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载更新日志失败'));
        return;
      }
      const items = Array.isArray(data?.items) ? data.items : [];
      setChangelogEntries(items);
      setChangelogTotal(Number(data?.total || items.length || 0));
    } catch (error) {
      showError(error?.message || t('加载更新日志失败'));
    } finally {
      setChangelogLoading(false);
    }
  };
  const openCreateChangelogModal = () => {
    setEditingChangelog(null);
    setChangelogForm({ date: '', content: '' });
    setChangelogModalVisible(true);
  };
  const openEditChangelogModal = (record) => {
    setEditingChangelog(record);
    setChangelogForm({
      date: record.date || '',
      content: record.content || '',
    });
    setChangelogModalVisible(true);
  };
  const submitChangelogModal = async () => {
    const payload = {
      date: formatChangelogDate(changelogForm.date),
      content: String(changelogForm.content || '').trim(),
    };
    if (!payload.date || !payload.content) {
      showError(t('请完整填写更新日志日期和内容'));
      return;
    }
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Changelog: true }));
      const res = editingChangelog?.id
        ? await API.put(`/api/changelog/admin/${editingChangelog.id}`, payload)
        : await API.post('/api/changelog/admin/', payload);
      const { success, message } = res.data || {};
      if (!success) {
        showError(message || t('更新日志保存失败'));
        return;
      }
      showSuccess(t('更新日志已保存'));
      setChangelogModalVisible(false);
      const nextPage = editingChangelog?.id ? changelogPage : 1;
      setChangelogPage(nextPage);
      await loadChangelogs(nextPage, changelogPageSize);
    } catch (error) {
      showError(error?.message || t('更新日志保存失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        Changelog: false,
      }));
    }
  };
  const deleteChangelog = async (id) => {
    try {
      const res = await API.delete(`/api/changelog/admin/${id}`);
      const { success, message } = res.data || {};
      if (!success) {
        showError(message || t('删除更新日志失败'));
        return;
      }
      showSuccess(t('更新日志已删除'));
      const nextPage =
        changelogEntries.length === 1 && changelogPage > 1
          ? changelogPage - 1
          : changelogPage;
      setChangelogPage(nextPage);
      await loadChangelogs(nextPage, changelogPageSize);
    } catch (error) {
      showError(error?.message || t('删除更新日志失败'));
    }
  };
  const changelogColumns = [
    {
      title: t('日期'),
      dataIndex: 'date',
      width: 180,
    },
    {
      title: t('内容（Markdown）'),
      dataIndex: 'content',
      render: (content) => (
        <div
          style={{
            maxWidth: 560,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
          title={content}
        >
          {content}
        </div>
      ),
    },
    {
      title: t('操作'),
      dataIndex: 'action',
      width: 100,
      render: (_, record) => (
        <Space>
          <Button
            theme='borderless'
            onClick={() => openEditChangelogModal(record)}
          >
            {t('编辑')}
          </Button>
          <Popconfirm
            title={t('确定删除这条更新日志？')}
            onConfirm={() => deleteChangelog(record.id)}
            position='left'
          >
            <Button type='danger' theme='borderless'>
              {t('删除')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 通用设置
  const formAPISettingGeneral = useRef();
  // 通用设置 - Notice
  const submitNotice = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Notice: true }));
      await updateOption('Notice', inputs.Notice);
      showSuccess(t('公告已更新'));
    } catch (error) {
      console.error(t('公告更新失败'), error);
      showError(t('公告更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Notice: false }));
    }
  };
  const submitDefaultSiteLanguage = async () => {
    try {
      setLoadingInput((s) => ({ ...s, DefaultSiteLanguage: true }));
      await updateOption(
        'general_setting.default_site_language',
        inputs['general_setting.default_site_language'] || 'zh-CN',
      );
      showSuccess(t('默认界面语言已保存'));
    } catch (error) {
      console.error(error);
      showError(t('设置保存失败'));
    } finally {
      setLoadingInput((s) => ({ ...s, DefaultSiteLanguage: false }));
    }
  };
  // 通用设置 - UserAgreement
  const submitUserAgreement = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_USER_AGREEMENT_KEY]: true,
      }));
      const saved = await updateOptions([
        [LEGAL_USER_AGREEMENT_KEY, inputs[LEGAL_USER_AGREEMENT_KEY] || ''],
        [
          LEGAL_USER_AGREEMENT_FORMAT_KEY,
          inputs[LEGAL_USER_AGREEMENT_FORMAT_KEY] || 'markdown',
        ],
      ]);
      if (saved) {
        showSuccess(t('用户协议已更新'));
      }
    } catch (error) {
      console.error(t('用户协议更新失败'), error);
      showError(t('用户协议更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_USER_AGREEMENT_KEY]: false,
      }));
    }
  };
  // 通用设置 - PrivacyPolicy
  const submitPrivacyPolicy = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_PRIVACY_POLICY_KEY]: true,
      }));
      const saved = await updateOptions([
        [LEGAL_PRIVACY_POLICY_KEY, inputs[LEGAL_PRIVACY_POLICY_KEY] || ''],
        [
          LEGAL_PRIVACY_POLICY_FORMAT_KEY,
          inputs[LEGAL_PRIVACY_POLICY_FORMAT_KEY] || 'markdown',
        ],
      ]);
      if (saved) {
        showSuccess(t('隐私政策已更新'));
      }
    } catch (error) {
      console.error(t('隐私政策更新失败'), error);
      showError(t('隐私政策更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        [LEGAL_PRIVACY_POLICY_KEY]: false,
      }));
    }
  };
  // 个性化设置
  const formAPIPersonalization = useRef();
  const formAPIDocsConfig = useRef();
  //  个性化设置 - SystemName
  const submitSystemName = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        SystemName: true,
      }));
      await updateOption('SystemName', inputs.SystemName);
      await updateOption('SystemNameEn', inputs.SystemNameEn);
      const zhName = String(inputs.SystemName ?? '').trim();
      const enName = String(inputs.SystemNameEn ?? '').trim();
      localStorage.setItem('system_name', zhName);
      localStorage.setItem('system_name_en', enName);
      const displayName = getSystemName();
      if (displayName) {
        document.title = displayName;
      }
      showSuccess(t('系统名称已更新'));
    } catch (error) {
      console.error(t('系统名称更新失败'), error);
      showError(t('系统名称更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        SystemName: false,
      }));
    }
  };

  // 个性化设置 - Logo
  const submitLogo = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Logo: true }));
      await updateOption('Logo', inputs.Logo);
      localStorage.setItem('logo', inputs.Logo || '');
      let iconLink = document.querySelector("link[rel~='icon']");
      if (!iconLink) {
        iconLink = document.createElement('link');
        iconLink.rel = 'icon';
        document.head.appendChild(iconLink);
      }
      iconLink.href = inputs.Logo || '/logo.png';
      showSuccess('Logo 已更新');
    } catch (error) {
      console.error('Logo 更新失败', error);
      showError('Logo 更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Logo: false }));
    }
  };
  const uploadLogo = async ({ file, onSuccess, onError }) => {
    const inst = file?.fileInstance || file;
    if (!inst) {
      onError(new Error('no file'));
      return;
    }
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Logo: true }));
      const fd = new FormData();
      fd.append('file', inst);
      fd.append('purpose', 'icons');
      const res = await API.post('/api/oss/upload', fd, {
        skipErrorHandler: true,
      });
      const { success, message, data } = res.data || {};
      const url = data?.url;
      if (!success || !url) {
        const err = new Error(message || t('上传失败'));
        onError(err);
        showError(err.message);
        return;
      }
      setInputs((prev) => ({ ...prev, Logo: url }));
      formAPIPersonalization.current?.setValue('Logo', url);
      onSuccess(data);
      showSuccess(t('Logo 上传成功，请点击「设置 Logo」保存'));
    } catch (error) {
      onError(error);
      showError(
        error?.response?.data?.message ||
          error?.message ||
          t('上传失败，请确认已启用 OSS 并完成配置'),
      );
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Logo: false }));
    }
  };
  // 个性化设置 - 首页内容
  const submitOption = async (key) => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        HomePageContent: true,
      }));
      await updateOption(key, inputs[key]);
      showSuccess('首页内容已更新');
    } catch (error) {
      console.error('首页内容更新失败', error);
      showError('首页内容更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        HomePageContent: false,
      }));
    }
  };
  const submitDocsConfig = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        DocsConfig: true,
      }));
      const results = await Promise.all(
        DOCS_CONFIG_KEYS.map((key) =>
          API.put('/api/option/', {
            key,
            value: inputs[key] || '',
          }),
        ),
      );
      const failed = results.find((res) => !res.data?.success);
      if (failed) {
        throw new Error(failed.data?.message || t('文档配置更新失败'));
      }
      showSuccess(t('文档配置已更新'));
    } catch (error) {
      console.error(t('文档配置更新失败'), error);
      showError(t('文档配置更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        DocsConfig: false,
      }));
    }
  };
  const uploadDocsImage =
    (field, label) =>
    async ({ file, onSuccess, onError }) => {
      const inst = file?.fileInstance || file;
      if (!inst) {
        onError(new Error('no file'));
        return;
      }
      try {
        setLoadingInput((loadingInput) => ({ ...loadingInput, [field]: true }));
        const fd = new FormData();
        fd.append('file', inst);
        const res = await API.post('/api/oss/upload', fd, {
          skipErrorHandler: true,
        });
        const { success, message, data } = res.data || {};
        const url = data?.url;
        if (!success || !url) {
          const err = new Error(message || t('上传失败'));
          onError(err);
          showError(err.message);
          return;
        }
        setInputs((prev) => ({ ...prev, [field]: url }));
        formAPIDocsConfig.current?.setValue(field, url);
        onSuccess(data);
        showSuccess(
          t('{{label}}上传成功，请点击「保存文档配置」保存', { label }),
        );
      } catch (error) {
        onError(error);
        showError(
          error?.response?.data?.message ||
            error?.message ||
            t('上传失败，请确认已启用 OSS 并完成配置'),
        );
      } finally {
        setLoadingInput((loadingInput) => ({
          ...loadingInput,
          [field]: false,
        }));
      }
    };
  // 个性化设置 - 关于
  const submitAbout = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, About: true }));
      await updateOption('About', inputs.About);
      await updateOption('AboutEn', inputs.AboutEn);
      showSuccess(t('关于内容已更新'));
    } catch (error) {
      console.error('关于内容更新失败', error);
      showError(t('关于内容更新失败'));
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, About: false }));
    }
  };
  // 个性化设置 - 页脚
  const submitFooter = async () => {
    try {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Footer: true }));
      await updateOption('Footer', inputs.Footer);
      showSuccess('页脚内容已更新');
    } catch (error) {
      console.error('页脚内容更新失败', error);
      showError('页脚内容更新失败');
    } finally {
      setLoadingInput((loadingInput) => ({ ...loadingInput, Footer: false }));
    }
  };

  const checkUpdate = async () => {
    try {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        CheckUpdate: true,
      }));
      // Use a CORS proxy to avoid direct cross-origin requests to GitHub API
      // Option 1: Use a public CORS proxy service
      // const proxyUrl = 'https://cors-anywhere.herokuapp.com/';
      // const res = await API.get(
      //   `${proxyUrl}https://api.github.com/repos/Calcium-Ion/new-api/releases/latest`,
      // );

      // Option 2: Use the JSON proxy approach which often works better with GitHub API
      // const res = await fetch(
      //   'https://api.github.com/repos/Calcium-Ion/new-api/releases/latest',
      //   {
      //     headers: {
      //       Accept: 'application/json',
      //       'Content-Type': 'application/json',
      //       // Adding User-Agent which is often required by GitHub API
      //       'User-Agent': 'new-api-update-checker',
      //     },
      //   },
      // ).then((response) => response.json());

      // Option 3: Use a local proxy endpoint
      // Create a cached version of the response to avoid frequent GitHub API calls
      // const res = await API.get('/api/status/github-latest-release');

      // const { tag_name, body } = res;
      // if (tag_name === statusState?.status?.version) {
      //   showSuccess(`已是最新版本：${tag_name}`);
      // } else {
      //   setUpdateData({
      //     tag_name: tag_name,
      //     content: marked.parse(body),
      //   });
      //   setShowUpdateModal(true);
      // }
      showInfo('更新检查功能已禁用');
    } catch (error) {
      console.error('Failed to check for updates:', error);
      showError('检查更新失败，请稍后再试');
    } finally {
      setLoadingInput((loadingInput) => ({
        ...loadingInput,
        CheckUpdate: false,
      }));
    }
  };
  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        if (item.key in inputs) {
          newInputs[item.key] = item.value;
        }
      });
      const mergedInputs = {
        ...inputs,
        ...newInputs,
      };
      if (!String(mergedInputs[LEGAL_USER_AGREEMENT_KEY] || '').trim()) {
        mergedInputs[LEGAL_USER_AGREEMENT_KEY] = defaultUserAgreementHtml;
        mergedInputs[LEGAL_USER_AGREEMENT_FORMAT_KEY] = DEFAULT_LEGAL_FORMAT;
      }
      if (!String(mergedInputs[LEGAL_PRIVACY_POLICY_KEY] || '').trim()) {
        mergedInputs[LEGAL_PRIVACY_POLICY_KEY] = defaultPrivacyPolicyHtml;
        mergedInputs[LEGAL_PRIVACY_POLICY_FORMAT_KEY] = DEFAULT_LEGAL_FORMAT;
      }
      setInputs(mergedInputs);
      formAPISettingGeneral.current?.setValues(mergedInputs);
      formAPIPersonalization.current?.setValues(mergedInputs);
      formAPIDocsConfig.current?.setValues(mergedInputs);
    } else {
      showError(message);
    }
  };

  useEffect(() => {
    getOptions();
    loadChangelogs(1, changelogPageSize);
  }, []);

  // Function to open GitHub release page
  // const openGitHubRelease = () => {
  //   window.open(
  //     `https://github.com/Calcium-Ion/new-api/releases/tag/${updateData.tag_name}`,
  //     '_blank',
  //   );
  // };

  const getStartTimeString = () => {
    const timestamp = statusState?.status?.start_time;
    return statusState.status ? timestamp2string(timestamp) : '';
  };

  return (
    <Row>
      <Col
        span={24}
        style={{
          marginTop: '10px',
          display: 'flex',
          flexDirection: 'column',
          gap: '10px',
        }}
      >
        {/* 版本信息 */}
        <Form>
          <Card
            id='setting-section-system-info'
            style={{
              display: activeSection === 'system-info' ? undefined : 'none',
            }}
          >
            <Form.Section text={t('系统信息')}>
              <Row>
                <Col span={16}>
                  <Space>
                    <Text>
                      {t('当前版本')}：
                      {statusState?.status?.version || t('未知')}
                    </Text>
                    <Button
                      type='primary'
                      onClick={checkUpdate}
                      loading={loadingInput['CheckUpdate']}
                    >
                      {t('检查更新')}
                    </Button>
                  </Space>
                </Col>
              </Row>
              <Row>
                <Col span={16}>
                  <Text>
                    {t('启动时间')}：{getStartTimeString()}
                  </Text>
                </Col>
              </Row>
            </Form.Section>
          </Card>
        </Form>
        {/* 通用设置 */}
        <Form
          values={inputs}
          getFormApi={(formAPI) => (formAPISettingGeneral.current = formAPI)}
        >
          <Card
            id='setting-section-general'
            style={{
              display: activeSection === 'general' ? undefined : 'none',
            }}
          >
            <Form.Section text={t('通用设置')}>
              <Form.TextArea
                label={t('公告')}
                placeholder={t(
                  '在此输入新的公告内容，支持 Markdown & HTML 代码',
                )}
                field={'Notice'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button onClick={submitNotice} loading={loadingInput['Notice']}>
                {t('设置公告')}
              </Button>
              <Form.Select
                label={t('站点默认界面语言')}
                field='general_setting.default_site_language'
                optionList={languageSelectOptions}
                helpText={t('站点默认界面语言说明')}
                onChange={(v) =>
                  setInputs((prev) => ({
                    ...prev,
                    'general_setting.default_site_language': v,
                  }))
                }
                style={{ maxWidth: 400 }}
              />
              <Button
                onClick={submitDefaultSiteLanguage}
                loading={loadingInput.DefaultSiteLanguage}
              >
                {t('保存默认界面语言')}
              </Button>
              <LegalContentEditor
                title={t('用户协议')}
                value={inputs[LEGAL_USER_AGREEMENT_KEY]}
                format={inputs[LEGAL_USER_AGREEMENT_FORMAT_KEY]}
                styleId='legal-editor-user-agreement-styles'
                placeholder={t(
                  '在此输入用户协议内容，支持 Markdown & HTML 代码',
                )}
                helpText={t(
                  '填写用户协议内容后，用户注册时将被要求勾选已阅读用户协议',
                )}
                saveText={t('设置用户协议')}
                loading={loadingInput[LEGAL_USER_AGREEMENT_KEY]}
                onChange={(value) =>
                  setInputs((prev) => ({
                    ...prev,
                    [LEGAL_USER_AGREEMENT_KEY]: value,
                  }))
                }
                onFormatChange={(format) =>
                  setInputs((prev) => ({
                    ...prev,
                    [LEGAL_USER_AGREEMENT_FORMAT_KEY]: format,
                  }))
                }
                onSave={submitUserAgreement}
              />
              <LegalContentEditor
                title={t('隐私政策')}
                value={inputs[LEGAL_PRIVACY_POLICY_KEY]}
                format={inputs[LEGAL_PRIVACY_POLICY_FORMAT_KEY]}
                styleId='legal-editor-privacy-policy-styles'
                placeholder={t(
                  '在此输入隐私政策内容，支持 Markdown & HTML 代码',
                )}
                helpText={t(
                  '填写隐私政策内容后，用户注册时将被要求勾选已阅读隐私政策',
                )}
                saveText={t('设置隐私政策')}
                loading={loadingInput[LEGAL_PRIVACY_POLICY_KEY]}
                onChange={(value) =>
                  setInputs((prev) => ({
                    ...prev,
                    [LEGAL_PRIVACY_POLICY_KEY]: value,
                  }))
                }
                onFormatChange={(format) =>
                  setInputs((prev) => ({
                    ...prev,
                    [LEGAL_PRIVACY_POLICY_FORMAT_KEY]: format,
                  }))
                }
                onSave={submitPrivacyPolicy}
              />
            </Form.Section>
          </Card>
        </Form>
        {/* 更新日志 */}
        <Form>
          <Card
            id='setting-section-changelog'
            style={{
              display: activeSection === 'changelog' ? undefined : 'none',
            }}
          >
            <Form.Section text={t('更新日志')}>
              <Space vertical align='start' style={{ width: '100%' }}>
                <Space wrap>
                  <Button type='primary' onClick={openCreateChangelogModal}>
                    {t('新增更新日志')}
                  </Button>
                </Space>
                <Table
                  rowKey='id'
                  columns={changelogColumns}
                  dataSource={changelogEntries}
                  loading={changelogLoading}
                  scroll={{ y: 360, x: 'max-content' }}
                  pagination={{
                    currentPage: changelogPage,
                    pageSize: changelogPageSize,
                    total: changelogTotal,
                    hideOnSinglePage: false,
                    showSizeChanger: true,
                    showQuickJumper: true,
                    showTotal: true,
                    pageSizeOptions: ['5', '10', '20', '50'],
                    onChange: (page, size) => {
                      setChangelogPage(page);
                      setChangelogPageSize(size);
                      loadChangelogs(page, size);
                    },
                    onShowSizeChange: (current, size) => {
                      setChangelogPage(1);
                      setChangelogPageSize(size);
                      loadChangelogs(1, size);
                    },
                  }}
                  empty={t('暂无更新日志')}
                  style={{ width: '100%' }}
                />
              </Space>
            </Form.Section>
          </Card>
        </Form>
        {/* 个性化设置 */}
        <Form
          values={inputs}
          getFormApi={(formAPI) => (formAPIPersonalization.current = formAPI)}
        >
          <Card
            id='setting-section-appearance'
            style={{
              display: activeSection === 'appearance' ? undefined : 'none',
            }}
          >
            <Form.Section text={t('个性化设置')}>
              <Form.Input
                label={t('中文系统名称')}
                placeholder={t('在此输入系统名称')}
                field={'SystemName'}
                onChange={handleInputChange}
              />
              <Form.Input
                label={t('英文系统名称')}
                placeholder={t('在此输入英文系统名称')}
                field={'SystemNameEn'}
                onChange={handleInputChange}
              />
              <Button
                onClick={submitSystemName}
                loading={loadingInput['SystemName']}
              >
                {t('设置系统名称')}
              </Button>
              <Form.Input
                label={t('Logo 图片地址')}
                placeholder={t('在此输入 Logo 图片地址')}
                field={'Logo'}
                onChange={handleInputChange}
              />
              <Space
                vertical
                align='start'
                spacing='tight'
                style={{ marginBottom: 12 }}
              >
                <Space align='center' spacing='tight' wrap>
                  <Upload
                    action=''
                    accept='image/*'
                    showUploadList={false}
                    customRequest={uploadLogo}
                  >
                    <Button loading={loadingInput['Logo']}>
                      {t('上传 Logo 图片')}
                    </Button>
                  </Upload>
                  <Button onClick={submitLogo} loading={loadingInput['Logo']}>
                    {t('设置 Logo')}
                  </Button>
                </Space>
                <Text type='tertiary' size='small'>
                  {t(
                    '可直接上传图片自动填写地址，或手动输入 URL；上传需先配置并启用 OSS',
                  )}
                </Text>
              </Space>
              <Form.TextArea
                label={t('首页内容')}
                placeholder={t(
                  '在此输入首页内容，支持 Markdown & HTML 代码，设置后首页的状态信息将不再显示。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为首页',
                )}
                field={'HomePageContent'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button
                onClick={() => submitOption('HomePageContent')}
                loading={loadingInput['HomePageContent']}
              >
                {t('设置首页内容')}
              </Button>
              <HomeFooterCertificatesSetting />
              <Form.TextArea
                label={t('关于')}
                placeholder={t(
                  '在此输入新的关于内容，支持 Markdown & HTML 代码。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性，这允许你设置任意网页作为关于页面',
                )}
                field={'About'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Form.TextArea
                label={t('关于（英文）')}
                placeholder={t(
                  '在此输入关于内容的英文版，支持 Markdown & HTML 代码。如果输入的是一个链接，则会使用该链接作为 iframe 的 src 属性',
                )}
                field={'AboutEn'}
                onChange={handleInputChange}
                style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                autosize={{ minRows: 6, maxRows: 12 }}
              />
              <Button onClick={submitAbout} loading={loadingInput['About']}>
                {t('设置关于')}
              </Button>
              {/*  */}
              <Banner
                fullMode={false}
                type='info'
                description={t(
                  '移除 One API 的版权标识必须首先获得授权，项目维护需要花费大量精力，如果本项目对你有意义，请主动支持本项目',
                )}
                closeIcon={null}
                style={{ marginTop: 15 }}
              />
              <Form.Input
                label={t('页脚')}
                placeholder={t(
                  '在此输入新的页脚，留空则使用默认页脚，支持 HTML 代码',
                )}
                field={'Footer'}
                onChange={handleInputChange}
              />
              <Button onClick={submitFooter} loading={loadingInput['Footer']}>
                {t('设置页脚')}
              </Button>
            </Form.Section>
          </Card>
        </Form>
        {/* 文档配置 */}
        <Form
          values={inputs}
          getFormApi={(formAPI) => (formAPIDocsConfig.current = formAPI)}
        >
          <Card
            id='setting-section-docs'
            style={{ display: activeSection === 'docs' ? undefined : 'none' }}
          >
            <Form.Section text={t('文档配置')}>
              <Card title={t('基础信息')} style={{ marginBottom: 12 }}>
                <Form.Input
                  label={t('文档品牌名称')}
                  placeholder='TokenFactory'
                  field='DocsBrandName'
                  onChange={handleInputChange}
                  helpText={t('用于替换文档站标题、SEO 和页面中的品牌关键词')}
                />
                <Row gutter={16}>
                  <Col span={8}>
                    <Form.Input
                      label={t('英文站点名称')}
                      placeholder='TokenFactory'
                      field='DocsSiteNameEn'
                      onChange={handleInputChange}
                    />
                  </Col>
                  <Col span={8}>
                    <Form.Input
                      label={t('中文站点名称')}
                      placeholder='开放词元工厂'
                      field='DocsSiteNameZh'
                      onChange={handleInputChange}
                    />
                  </Col>
                  <Col span={8}>
                    <Form.Input
                      label={t('日文站点名称')}
                      placeholder='TokenFactory'
                      field='DocsSiteNameJa'
                      onChange={handleInputChange}
                    />
                  </Col>
                </Row>
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Input
                      label={t('文档首页地址')}
                      placeholder='https://tokenfactoryopen.com/'
                      field='DocsHomeUrl'
                      onChange={handleInputChange}
                    />
                  </Col>
                  <Col span={12}>
                    <Form.Input
                      label={t('GitHub 链接')}
                      placeholder='https://github.com/fyinfor/token-factory'
                      field='DocsGithubUrl'
                      onChange={handleInputChange}
                    />
                  </Col>
                </Row>
                <Form.TextArea
                  label={t('SEO 关键字')}
                  placeholder={t('多个关键字用英文逗号分隔')}
                  field='DocsMetaKeywords'
                  onChange={handleInputChange}
                  autosize={{ minRows: 2, maxRows: 6 }}
                />
              </Card>

              <Card title={t('图片资源')} style={{ marginBottom: 12 }}>
                <Row gutter={16}>
                  <Col span={18}>
                    <Form.Input
                      label={t('文档 Logo 地址')}
                      placeholder='/assets/logo.png'
                      field='DocsLogoUrl'
                      onChange={handleInputChange}
                    />
                    <Upload
                      action=''
                      accept='image/*'
                      showUploadList={false}
                      customRequest={uploadDocsImage(
                        'DocsLogoUrl',
                        t('文档 Logo'),
                      )}
                    >
                      <Button loading={loadingInput['DocsLogoUrl']}>
                        {t('上传文档 Logo')}
                      </Button>
                    </Upload>
                    <div style={{ marginTop: 6 }}>
                      <Text type='tertiary' size='small'>
                        {t(
                          '支持绝对 URL，或直接上传图片自动填写地址；上传需先配置并启用 OSS',
                        )}
                      </Text>
                    </div>
                  </Col>
                  <Col span={6}>
                    <Text type='tertiary' size='small'>
                      {t('预览')}
                    </Text>
                    <div style={{ marginTop: 8 }}>
                      {inputs.DocsLogoUrl ? (
                        <img
                          src={inputs.DocsLogoUrl}
                          alt={t('文档 Logo 预览')}
                          style={docsImagePreviewStyle}
                          onClick={() =>
                            setDocsImagePreview({
                              visible: true,
                              url: inputs.DocsLogoUrl,
                              title: t('文档 Logo 预览'),
                            })
                          }
                        />
                      ) : (
                        <div style={docsImageEmptyStyle}>{t('暂无图片')}</div>
                      )}
                    </div>
                  </Col>
                </Row>
                <Row gutter={16}>
                  <Col span={18}>
                    <Form.Input
                      label={t('企业微信二维码地址')}
                      placeholder='/assets/wechat.png'
                      field='DocsBusinessWechatQrUrl'
                      onChange={handleInputChange}
                    />
                    <Upload
                      action=''
                      accept='image/*'
                      showUploadList={false}
                      customRequest={uploadDocsImage(
                        'DocsBusinessWechatQrUrl',
                        t('企业微信二维码'),
                      )}
                    >
                      <Button loading={loadingInput['DocsBusinessWechatQrUrl']}>
                        {t('上传企业微信二维码')}
                      </Button>
                    </Upload>
                    <div style={{ marginTop: 6 }}>
                      <Text type='tertiary' size='small'>
                        {t(
                          '支持绝对 URL，或直接上传图片自动填写地址；上传需先配置并启用 OSS',
                        )}
                      </Text>
                    </div>
                  </Col>
                  <Col span={6}>
                    <Text type='tertiary' size='small'>
                      {t('预览')}
                    </Text>
                    <div style={{ marginTop: 8 }}>
                      {inputs.DocsBusinessWechatQrUrl ? (
                        <img
                          src={inputs.DocsBusinessWechatQrUrl}
                          alt={t('企业微信二维码预览')}
                          style={docsImagePreviewStyle}
                          onClick={() =>
                            setDocsImagePreview({
                              visible: true,
                              url: inputs.DocsBusinessWechatQrUrl,
                              title: t('企业微信二维码预览'),
                            })
                          }
                        />
                      ) : (
                        <div style={docsImageEmptyStyle}>{t('暂无图片')}</div>
                      )}
                    </div>
                  </Col>
                </Row>
              </Card>

              <Card title={t('商务信息')} style={{ marginBottom: 12 }}>
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Input
                      label={t('商务联系电话')}
                      placeholder='156 2568 9773'
                      field='DocsBusinessPhone'
                      onChange={handleInputChange}
                    />
                  </Col>
                  <Col span={12}>
                    <Form.Input
                      label={t('商务电话拨号号码')}
                      placeholder='15625689773'
                      field='DocsBusinessPhoneHref'
                      onChange={handleInputChange}
                      helpText={t('用于 tel 链接，建议只填写数字和区号')}
                    />
                  </Col>
                </Row>
                <Row gutter={16}>
                  <Col span={8}>
                    <Form.Input
                      label={t('工作时间说明（中文）')}
                      placeholder='工作日 9:30 - 12:00 13:30 - 19:00'
                      field='DocsBusinessWorkTimeZh'
                      onChange={handleInputChange}
                    />
                  </Col>
                  <Col span={8}>
                    <Form.Input
                      label={t('工作时间说明（英文）')}
                      placeholder='Weekdays 9:30 - 12:00, 13:30 - 19:00'
                      field='DocsBusinessWorkTimeEn'
                      onChange={handleInputChange}
                    />
                  </Col>
                  <Col span={8}>
                    <Form.Input
                      label={t('工作时间说明（日文）')}
                      placeholder='平日 9:30 - 12:00、13:30 - 19:00'
                      field='DocsBusinessWorkTimeJa'
                      onChange={handleInputChange}
                    />
                  </Col>
                </Row>
              </Card>
              <Button
                onClick={submitDocsConfig}
                loading={loadingInput['DocsConfig']}
              >
                {t('保存文档配置')}
              </Button>
            </Form.Section>
          </Card>
        </Form>

        {/* 素材设置（Seedance2.0 合规素材库） */}
        <div
          id='setting-section-material'
          style={{ display: activeSection === 'material' ? undefined : 'none' }}
        >
          <SettingsSeedance />
        </div>
      </Col>
      <Modal
        title={t('新版本') + '：' + updateData.tag_name}
        visible={showUpdateModal}
        onCancel={() => setShowUpdateModal(false)}
        footer={[
          <Button
            key='details'
            type='primary'
            onClick={() => {
              setShowUpdateModal(false);
              openGitHubRelease();
            }}
          >
            {t('详情')}
          </Button>,
        ]}
      >
        <div dangerouslySetInnerHTML={{ __html: updateData.content }}></div>
      </Modal>
      <Modal
        title={docsImagePreview.title}
        visible={docsImagePreview.visible}
        onCancel={() =>
          setDocsImagePreview({ visible: false, url: '', title: '' })
        }
        footer={null}
        style={{ maxWidth: '80vw' }}
      >
        {docsImagePreview.url && (
          <div style={{ textAlign: 'center', padding: '12px 0 20px' }}>
            <img
              src={docsImagePreview.url}
              alt={docsImagePreview.title}
              style={{
                maxWidth: '100%',
                maxHeight: '64vh',
                objectFit: 'contain',
              }}
            />
          </div>
        )}
      </Modal>
      <Modal
        title={editingChangelog ? t('编辑更新日志') : t('新增更新日志')}
        visible={changelogModalVisible}
        onOk={submitChangelogModal}
        onCancel={() => setChangelogModalVisible(false)}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={loadingInput['Changelog']}
        width={720}
      >
        <Form
          layout='vertical'
          initValues={changelogForm}
          key={
            editingChangelog?.id
              ? `edit-changelog-${editingChangelog.id}`
              : `new-changelog-${changelogModalVisible ? 'open' : 'closed'}`
          }
        >
          <Form.DatePicker
            field='date'
            label={t('日期')}
            type='date'
            placeholder={t('请选择日期')}
            style={{ width: '100%' }}
            rules={[{ required: true, message: t('请输入日期') }]}
            onChange={(value) =>
              setChangelogForm((prev) => ({
                ...prev,
                date: formatChangelogDate(value),
              }))
            }
          />
          <Form.Slot label={t('内容（Markdown）')}>
            <Tabs type='line'>
              <Tabs.TabPane tab={t('编辑')} itemKey='edit'>
                <Form.TextArea
                  field='content'
                  label={t('内容（Markdown）')}
                  placeholder={t('请输入更新内容，支持 Markdown')}
                  autosize={{ minRows: 8, maxRows: 16 }}
                  rules={[{ required: true, message: t('请输入更新内容') }]}
                  style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                  onChange={(value) =>
                    setChangelogForm((prev) => ({
                      ...prev,
                      content: value,
                    }))
                  }
                />
              </Tabs.TabPane>
              <Tabs.TabPane tab={t('预览')} itemKey='preview'>
                <div
                  style={{
                    minHeight: 220,
                    maxHeight: 420,
                    overflowY: 'auto',
                    padding: 12,
                    border: '1px solid var(--semi-color-border)',
                    borderRadius: 6,
                    background: 'var(--semi-color-bg-0)',
                  }}
                >
                  {String(changelogForm.content || '').trim() ? (
                    <MarkdownRenderer content={changelogForm.content} />
                  ) : (
                    <Text type='tertiary'>{t('暂无内容')}</Text>
                  )}
                </div>
              </Tabs.TabPane>
            </Tabs>
          </Form.Slot>
        </Form>
      </Modal>
    </Row>
  );
};

export default OtherSetting;
