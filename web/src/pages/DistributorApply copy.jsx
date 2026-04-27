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

import React, { useContext, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Typography,
  Banner,
  Spin,
  Upload,
  Modal,
} from '@douyinfe/semi-ui';
import { IconFile, IconUserGroup } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess, userIsDistributorUser } from '../helpers';
import { StatusContext } from '../context/Status';
import { UserContext } from '../context/User';
import DOMPurify from 'dompurify';

const { Text } = Typography;

const statusText = (s) => {
  if (s === 1) return '审核中';
  if (s === 2) return '已通过';
  if (s === 3) return '已驳回';
  return '—';
};

export default function DistributorApply() {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [userState] = useContext(UserContext);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [app, setApp] = useState(null);
  const [urls, setUrls] = useState([]);
  const [previewUrl, setPreviewUrl] = useState(null);
  const formApi = React.useRef(null);

  const csImage = (
    statusState?.status?.distributor_apply_cs_image_url || ''
  ).trim();
  const showCsColumn = Boolean(csImage);

  const applyIntroHtmlRaw = (
    statusState?.status?.distributor_apply_intro_html || ''
  ).trim();
  const applyIntroHtml = applyIntroHtmlRaw
    ? DOMPurify.sanitize(applyIntroHtmlRaw, { USE_PROFILES: { html: true } })
    : '';

  const isPdfUrl = (u) => /\.pdf(\?|$)/i.test(u || '');

  const load = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/distributor/my_application');
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setApp(data || null);
    } catch {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const isDist = useMemo(() => {
    const u = userState?.user;
    if (u) return userIsDistributorUser(u);
    try {
      const raw = localStorage.getItem('user');
      if (raw) return userIsDistributorUser(JSON.parse(raw));
    } catch {
      // ignore
    }
    return false;
  }, [userState?.user]);

  /** 申请记录为已通过，且当前账号仍是代理：显示「去分销中心」 */
  const showApprovedForActiveDistributor = app?.status === 2 && isDist;

  /** 记录曾为已通过，但账号已非代理（如被降级）：应允许重新提交 */
  const showReapplyAfterRevoked = app?.status === 2 && !isDist;

  useEffect(() => {
    if (!showReapplyAfterRevoked || !app || loading) return;
    const tid = window.setTimeout(() => {
      try {
        formApi.current?.setValues({
          real_name: app.real_name || '',
          id_card_no: app.id_card_no || '',
          contact: app.contact || '',
        });
        const raw = app.qualification_urls;
        if (raw) {
          const j = typeof raw === 'string' ? JSON.parse(raw) : raw;
          if (Array.isArray(j) && j.length) setUrls(j.filter(Boolean));
        }
      } catch {
        // ignore
      }
    }, 0);
    return () => window.clearTimeout(tid);
  }, [showReapplyAfterRevoked, app, loading]);

  const onSubmit = async () => {
    const api = formApi.current;
    if (!api) return;
    try {
      await api.validate();
    } catch {
      return;
    }
    const values = api.getValues();
    if (!urls.length) {
      showError(t('请上传资格证书'));
      return;
    }
    setSubmitting(true);
    try {
      const res = await API.post('/api/distributor/application', {
        real_name: values.real_name,
        id_card_no: values.id_card_no,
        contact: values.contact,
        qualification_urls: urls,
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('提交成功'));
        await load();
      } else {
        showError(message);
      }
    } catch {
      showError(t('提交失败'));
    } finally {
      setSubmitting(false);
    }
  };

  if (isDist) {
    return (
      <div className='mt-16 px-4 max-w-3xl mx-auto'>
        <Banner
          type='success'
          description={t('您已是代理，无需再次申请')}
        />
      </div>
    );
  }

  return (
    <div className='mt-14 px-4 pb-16'>
      <div
        className={`mx-auto flex flex-col gap-8 ${showCsColumn ? 'max-w-5xl lg:flex-row' : 'max-w-3xl'}`}
      >
        <Card className='flex-1 !rounded-2xl min-w-0 !overflow-hidden'>
          <header className='distributor-apply-hero relative mb-6 rounded-xl p-[12px]'>
            <span
              className='distributor-apply-hero-orb distributor-apply-hero-orb--a'
              aria-hidden
            />
            <span
              className='distributor-apply-hero-orb distributor-apply-hero-orb--b'
              aria-hidden
            />
            <div className='relative z-[1] flex flex-col gap-4'>
              <div
                className='distributor-apply-hero-icon flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl border border-[var(--semi-color-border)] bg-[var(--semi-color-bg-0)] text-[var(--semi-color-primary)] shadow-sm'
                aria-hidden
              >
                <IconUserGroup size={28} />
              </div>
              <div className='min-w-0 flex-1 space-y-2'>
                <Typography.Title
                  heading={3}
                  className='distributor-apply-hero-title !mb-0 !mt-0 !font-semibold !tracking-tight'
                >
                  {t('分销伙伴招募')}
                </Typography.Title>
                {applyIntroHtml ? (
                  <div
                    className='distributor-apply-intro-html max-w-none text-[15px] leading-relaxed text-[var(--semi-color-text-0)] [&_p]:mb-2 [&_p]:last:mb-0 [&_img]:max-w-full [&_img]:h-auto [&_ul]:mb-2 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:mb-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_a]:text-[var(--semi-color-primary)] [&_blockquote]:border-l-2 [&_blockquote]:border-[var(--semi-color-border)] [&_blockquote]:pl-3'
                    dangerouslySetInnerHTML={{ __html: applyIntroHtml }}
                  />
                ) : null}
                <Text
                  type='tertiary'
                  className='distributor-apply-hero-desc !block !text-[15px] !leading-relaxed'
                >
                  {t('请如实填写以下信息，审核通过后即可获得代理资格与邀请分成。')}
                </Text>
              </div>
            </div>
          </header>

          {/* {app && (
            <Banner
              className='mb-4'
              type={
                app.status === 3 ? 'danger' : app.status === 2 ? 'success' : 'info'
              }
              description={
                <span>
                  {t('当前状态')}：{statusText(app.status)}
                  {app.status === 3 && app.reject_reason ? (
                    <span className='block mt-1'>
                      {t('驳回原因')}：{app.reject_reason}
                    </span>
                  ) : null}
                </span>
              }
            />
          )} */}

          {app?.status === 1 ? (
            <div className='py-10 px-4 text-center'>
              <Text className='!text-lg md:!text-xl !leading-relaxed !font-medium text-[var(--semi-color-text-0)]'>
                {t('您的申请正在审核中，请耐心等待。')}
              </Text>
            </div>
          ) : showApprovedForActiveDistributor ? (
            <div className='py-10 px-4 text-center'>
              <Text className='!text-lg md:!text-xl !leading-relaxed !font-medium text-[var(--semi-color-text-0)]'>
                {t('申请已通过，请从侧栏进入「分销中心」。')}
              </Text>
            </div>
          ) : (
            <Spin spinning={loading}>
              {showReapplyAfterRevoked ? (
                <Banner
                  type='info'
                  className='mb-4'
                  description={t(
                    '您的代理资格已失效（例如已被管理员调整），可修改资料后重新提交审核。',
                  )}
                />
              ) : null}
              <Form
                getFormApi={(f) => (formApi.current = f)}
                layout='vertical'
              >
                <Form.Input
                  field='real_name'
                  label={t('姓名')}
                  rules={[{ required: true, message: t('必填') }]}
                />
                <Form.Input
                  field='id_card_no'
                  label={t('身份证')}
                  rules={[{ required: true, message: t('必填') }]}
                />
                <Form.Input
                  field='contact'
                  label={t('联系方式')}
                  rules={[{ required: true, message: t('必填') }]}
                />
                <div className='mb-4'>
                  <Text strong className='block mb-2'>
                    {t('资格证书')}
                  </Text>
                  <Upload
                    action=''
                    accept='image/*,.pdf'
                    showUploadList={false}
                    customRequest={async ({ file, onSuccess, onError }) => {
                      const fd = new FormData();
                      const inst = file.fileInstance || file;
                      fd.append('file', inst);
                      try {
                        const res = await API.post('/api/oss/upload', fd, {
                          skipErrorHandler: true,
                        });
                        const { success, message, data } = res.data || {};
                        if (!success || !data?.url) {
                          onError(new Error(message || 'upload'));
                          showError(message || t('上传失败'));
                          return;
                        }
                        setUrls((prev) =>
                          prev.length >= 5 ? prev : [...prev, data.url],
                        );
                        onSuccess(data);
                        showSuccess(t('已上传'));
                      } catch (e) {
                        onError(e);
                        showError(
                          e?.response?.data?.message || t('上传失败'),
                        );
                      }
                    }}
                    limit={5}
                    multiple
                    disabled={urls.length >= 5}
                  >
                    <Button disabled={urls.length >= 5}>
                      {t('上传文件')}
                    </Button>
                  </Upload>
                  <Text type='tertiary' size='small' className='block mt-1'>
                    {t('支持图片或 PDF，最多 5 个；点击图片可大图预览')}
                  </Text>
                  {urls.length > 0 && (
                    <div className='mt-3 flex flex-wrap gap-3'>
                      {urls.map((u, idx) =>
                        isPdfUrl(u) ? (
                          <div
                            key={`${u}-${idx}`}
                            className='relative flex h-24 w-24 flex-col items-center justify-center rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)]'
                          >
                            <IconFile size='large' />
                            <span className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
                              PDF
                            </span>
                            <button
                              type='button'
                              className='absolute inset-0 rounded-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-primary'
                              title={t('在新窗口打开')}
                              onClick={() =>
                                window.open(u, '_blank', 'noopener,noreferrer')
                              }
                            />
                            <Button
                              size='small'
                              type='danger'
                              theme='borderless'
                              className='!absolute -right-1 -top-1 !min-w-0 z-10'
                              onClick={(e) => {
                                e.stopPropagation();
                                setUrls((prev) =>
                                  prev.filter((_, i) => i !== idx),
                                );
                              }}
                            >
                              ×
                            </Button>
                          </div>
                        ) : (
                          <div
                            key={`${u}-${idx}`}
                            className='relative h-24 w-24 overflow-hidden rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)]'
                          >
                            <button
                              type='button'
                              className='block h-full w-full cursor-zoom-in p-0 border-0 bg-transparent'
                              onClick={() => setPreviewUrl(u)}
                            >
                              <img
                                src={u}
                                alt=''
                                className='h-full w-full object-cover'
                              />
                            </button>
                            <Button
                              size='small'
                              type='danger'
                              theme='borderless'
                              className='!absolute -right-1 -top-1 !min-w-0'
                              onClick={(e) => {
                                e.stopPropagation();
                                setUrls((prev) =>
                                  prev.filter((_, i) => i !== idx),
                                );
                              }}
                            >
                              ×
                            </Button>
                          </div>
                        ),
                      )}
                    </div>
                  )}
                </div>
                <Button
                  theme='solid'
                  type='primary'
                  loading={submitting}
                  onClick={onSubmit}
                >
                  {showReapplyAfterRevoked
                    ? t('重新提交审核')
                    : t('提交申请')}
                </Button>
              </Form>
            </Spin>
          )}
        </Card>

        {showCsColumn ? (
          <div className='w-full lg:w-80 flex-shrink-0'>
            <Text strong className='block mb-2 text-center'>
              {t('联系客服')}
            </Text>
            <button
              type='button'
              className='w-full cursor-zoom-in rounded-xl border border-[var(--semi-color-border)] bg-transparent p-0'
              onClick={() => setPreviewUrl(csImage)}
            >
              <img
                src={csImage}
                alt=''
                className='w-full rounded-xl object-contain'
              />
            </button>
          </div>
        ) : null}
      </div>

      <Modal
        title={t('预览')}
        visible={Boolean(previewUrl)}
        onCancel={() => setPreviewUrl(null)}
        footer={null}
        width={Math.min(960, typeof window !== 'undefined' ? window.innerWidth - 48 : 960)}
      >
        {previewUrl &&
          (isPdfUrl(previewUrl) ? (
            <div className='py-6 text-center'>
              <Button
                type='primary'
                onClick={() =>
                  window.open(previewUrl, '_blank', 'noopener,noreferrer')
                }
              >
                {t('在新窗口打开 PDF')}
              </Button>
            </div>
          ) : (
            <div className='flex max-h-[85vh] justify-center overflow-auto p-2'>
              <img
                src={previewUrl}
                alt=''
                className='max-h-[85vh] max-w-full object-contain'
              />
            </div>
          ))}
      </Modal>
    </div>
  );
}
