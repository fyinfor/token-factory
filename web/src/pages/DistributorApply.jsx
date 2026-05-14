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
  Spin,
  Upload,
  Modal,
  Popover,
  Progress,
  Radio,
} from '@douyinfe/semi-ui';
import {
  IconFile,
  IconUserGroup,
  IconAlertTriangle,
  IconInfoCircle,
  IconTick,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess, userIsDistributorUser } from '../helpers';
import { StatusContext } from '../context/Status';
import { UserContext } from '../context/User';
import DOMPurify from 'dompurify';

const { Text } = Typography;

/** 代理申请：必填项标题（红 * + 加粗），与资格证书区块一致 */
function ApplyRequiredLabel({ children, className = '' }) {
  return (
    <Text
      strong
      className={`text-[var(--semi-color-text-0)] ${className}`.trim()}
    >
      <span
        className='text-[var(--semi-color-danger)] mr-1 font-normal'
        aria-hidden
      >
        *
      </span>
      {children}
    </Text>
  );
}

const applyTrimmedRequiredRule = (t) => ({
  validator: (rule, value) => {
    const v = value == null ? '' : String(value).trim();
    if (!v) return Promise.reject(t('必填'));
    return Promise.resolve();
  },
});

const iconLg = 22;

/**
 * 自研圆角提示条（与主卡片 `rounded-2xl` 一致，替代 Semi Banner 直角/风格差异问题）
 * @param {'danger' | 'info' | 'success'} variant
 */
function ApplyStatusNotice({
  variant = 'info',
  title,
  className = '',
  children,
}) {
  const tone =
    variant === 'danger'
      ? {
          frame:
            'border-[var(--semi-color-danger-light-hover)] bg-[var(--semi-color-danger-light-default)]',
          icon: (
            <IconAlertTriangle
              size={iconLg}
              style={{ color: 'var(--semi-color-danger)' }}
            />
          ),
        }
      : variant === 'success'
        ? {
            frame:
              'border-[var(--semi-color-success-light-hover)] bg-[var(--semi-color-success-light-default)]',
            icon: (
              <IconTick
                size={iconLg}
                style={{ color: 'var(--semi-color-success)' }}
              />
            ),
          }
        : {
            frame:
              'border-[var(--semi-color-info-light-hover)] bg-[var(--semi-color-info-light-default)]',
            icon: (
              <IconInfoCircle
                size={iconLg}
                style={{ color: 'var(--semi-color-primary)' }}
              />
            ),
          };

  return (
    <div
      role={variant === 'danger' ? 'alert' : 'status'}
      className={[
        'distributor-apply-notice flex gap-3 rounded-2xl border border-solid p-3.5 shadow-sm md:p-4',
        tone.frame,
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <span className='shrink-0 pt-0.5' aria-hidden>
        {tone.icon}
      </span>
      <div className='min-w-0 flex-1 text-[15px] leading-relaxed [overflow-wrap:anywhere] text-[var(--semi-color-text-0)]'>
        {title ? (
          <div className='mb-1.5 font-semibold text-[var(--semi-color-text-0)]'>
            {title}
          </div>
        ) : null}
        {children}
      </div>
    </div>
  );
}

export default function DistributorApply() {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [userState] = useContext(UserContext);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [app, setApp] = useState(null);
  const [urls, setUrls] = useState([]);
  const [previewUrl, setPreviewUrl] = useState(null);
  const [applyType, setApplyType] = useState(1);
  /** 上传真实进度（axios），未确认 url 前不超过 99 */
  const [uploadPct, setUploadPct] = useState(null);
  /** 展示用平滑进度，避免单次回调 0→99 无过渡感 */
  const [uploadDisplayPct, setUploadDisplayPct] = useState(null);
  const uploadTargetRef = React.useRef(null);
  const uploadDisplayRef = React.useRef(0);
  const uploadSmoothRafRef = React.useRef(null);
  const formApi = React.useRef(null);

  uploadTargetRef.current = uploadPct;

  useEffect(() => {
    if (uploadPct == null) {
      if (uploadSmoothRafRef.current != null) {
        cancelAnimationFrame(uploadSmoothRafRef.current);
        uploadSmoothRafRef.current = null;
      }
      uploadDisplayRef.current = 0;
      setUploadDisplayPct(null);
      return;
    }

    let cancelled = false;
    const tick = () => {
      if (cancelled) return;
      const target = uploadTargetRef.current;
      if (target == null) return;
      const cur = uploadDisplayRef.current;
      if (cur < target - 0.01) {
        const next = Math.min(
          target,
          cur + Math.max(0.55, (target - cur) * 0.2),
        );
        uploadDisplayRef.current = next;
        setUploadDisplayPct(Math.round(next));
        uploadSmoothRafRef.current = requestAnimationFrame(tick);
      } else if (cur > target + 0.01) {
        uploadDisplayRef.current = target;
        setUploadDisplayPct(Math.round(target));
        uploadSmoothRafRef.current = requestAnimationFrame(tick);
      } else {
        uploadDisplayRef.current = target;
        setUploadDisplayPct(Math.round(target));
        uploadSmoothRafRef.current = null;
      }
    };

    uploadSmoothRafRef.current = requestAnimationFrame(tick);
    return () => {
      cancelled = true;
      if (uploadSmoothRafRef.current != null) {
        cancelAnimationFrame(uploadSmoothRafRef.current);
        uploadSmoothRafRef.current = null;
      }
    };
  }, [uploadPct]);

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

  /** 申请记录为已通过，且当前账号仍是代理：显示「去代理分销」 */
  const showApprovedForActiveDistributor = app?.status === 2 && isDist;

  /** 记录曾为已通过，但账号已非代理（如被降级）：应允许重新提交 */
  const showReapplyAfterRevoked = app?.status === 2 && !isDist;

  const shouldPrefillFormFromApp = showReapplyAfterRevoked || app?.status === 3;

  useEffect(() => {
    if (!shouldPrefillFormFromApp || !app || loading) return;
    const tid = window.setTimeout(() => {
      try {
        formApi.current?.setValues({
          real_name: app.real_name || '',
          id_card_no: app.id_card_no || '',
          contact: app.contact || '',
        });
        const at = Number(app.apply_type);
        setApplyType(at === 2 ? 2 : 1);
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
  }, [shouldPrefillFormFromApp, app, loading]);

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
        apply_type: applyType,
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
      <div className='distributor-apply-page-root'>
        <div className='w-full max-w-3xl px-1'>
          <ApplyStatusNotice variant='success' className='w-full'>
            {t('您已是代理，无需再次申请')}
          </ApplyStatusNotice>
        </div>
      </div>
    );
  }

  const csPopoverContent = showCsColumn ? (
    <div className='distributor-apply-cs-popover w-[min(100vw-3rem,260px)] max-w-[260px] p-1'>
      <button
        type='button'
        className='w-full cursor-zoom-in rounded-lg border-0 bg-transparent p-0'
        onClick={() => setPreviewUrl(csImage)}
      >
        <img
          src={csImage}
          alt=''
          className='h-auto w-full max-w-full rounded-md object-contain'
        />
      </button>
    </div>
  ) : null;

  const rightBody =
    app?.status === 1 ? (
      <div className='flex min-h-0 flex-1 flex-col items-center justify-center py-8 text-center'>
        <Text className='!text-lg md:!text-xl !leading-relaxed !font-medium text-[var(--semi-color-text-0)]'>
          {t('您的申请正在审核中，请耐心等待。')}
        </Text>
      </div>
    ) : showApprovedForActiveDistributor ? (
      <div className='flex min-h-0 flex-1 flex-col items-center justify-center py-8 text-center'>
        <Text className='!text-lg md:!text-xl !leading-relaxed !font-medium text-[var(--semi-color-text-0)]'>
          {t('申请已通过，请从侧栏进入「代理分销」。')}
        </Text>
      </div>
    ) : (
      <Spin spinning={loading} className='w-full min-w-0'>
        {app?.status === 3 ? (
          <ApplyStatusNotice
            variant='danger'
            className='mb-4 shrink-0'
            title={t('申请已被驳回')}
          >
            {app?.reject_reason ? (
              <div
                className='whitespace-pre-wrap font-normal'
                style={{ lineHeight: 1.6 }}
              >
                {app.reject_reason}
              </div>
            ) : (
              <div className='font-normal'>
                {t('请根据管理员说明修改资料后重新提交。')}
              </div>
            )}
          </ApplyStatusNotice>
        ) : null}
        {showReapplyAfterRevoked ? (
          <ApplyStatusNotice variant='info' className='mb-4 shrink-0'>
            {t(
              '您的代理资格已失效（例如已被管理员调整），可修改资料后重新提交审核。',
            )}
          </ApplyStatusNotice>
        ) : null}
        <Form getFormApi={(f) => (formApi.current = f)} layout='vertical'>
          <div className='mb-4'>
            <Text strong className='mb-2 block'>
              {t('申请类型')}
            </Text>
            <Radio.Group
              type='button'
              value={applyType}
              onChange={(val) => {
                const v =
                  val && typeof val === 'object' && 'target' in val
                    ? val.target.value
                    : val;
                setApplyType(Number(v));
              }}
            >
              <Radio value={1}>{t('个人申请')}</Radio>
              <Radio value={2}>{t('企业申请')}</Radio>
            </Radio.Group>
          </div>
          <Form.Input
            field='real_name'
            label={
              <ApplyRequiredLabel>
                {applyType === 2 ? t('企业名称') : t('姓名')}
              </ApplyRequiredLabel>
            }
            rules={[applyTrimmedRequiredRule(t)]}
          />
          <Form.Input
            field='id_card_no'
            label={
              <ApplyRequiredLabel>
                {applyType === 2 ? t('统一社会信用代码') : t('身份证')}
              </ApplyRequiredLabel>
            }
            rules={[applyTrimmedRequiredRule(t)]}
          />
          <Form.Input
            field='contact'
            label={<ApplyRequiredLabel>{t('联系方式')}</ApplyRequiredLabel>}
            rules={[applyTrimmedRequiredRule(t)]}
          />
          <div className='mb-4'>
            <ApplyRequiredLabel className='block mb-2'>
              {t('资格证书')}
            </ApplyRequiredLabel>
            <Upload
              action=''
              accept='image/*,.pdf'
              showUploadList={false}
              customRequest={async ({
                file,
                onSuccess,
                onError,
                onProgress,
              }) => {
                const fd = new FormData();
                const inst = file.fileInstance || file;
                fd.append('file', inst);
                setUploadPct(0);
                try {
                  const res = await API.post('/api/oss/upload', fd, {
                    skipErrorHandler: true,
                    onUploadProgress: (ev) => {
                      const total = ev.total || ev.loaded || 1;
                      const raw = Math.round((ev.loaded * 100) / total);
                      // 字节传完不等于业务成功：未返回 url 前不显示 100%，避免误导
                      const pct = Math.min(99, raw);
                      setUploadPct(pct);
                      if (typeof onProgress === 'function') {
                        onProgress({ total, loaded: ev.loaded });
                      }
                    },
                  });
                  const { success, message, data } = res.data || {};
                  if (!success || !data?.url) {
                    onError(new Error(message || 'upload'));
                    showError(message || t('上传失败'));
                    return;
                  }
                  setUploadPct(100);
                  setUrls((prev) =>
                    prev.length >= 5 ? prev : [...prev, data.url],
                  );
                  onSuccess(data);
                  showSuccess(t('已上传'));
                } catch (e) {
                  onError(e);
                  showError(e?.response?.data?.message || t('上传失败'));
                } finally {
                  setUploadPct(null);
                }
              }}
              limit={5}
              multiple
              disabled={urls.length >= 5}
            >
              <Button disabled={urls.length >= 5}>{t('上传文件')}</Button>
            </Upload>
            {uploadPct != null ? (
              <Progress
                percent={uploadDisplayPct ?? uploadPct}
                showInfo
                className='mt-2'
              />
            ) : null}
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
                          setUrls((prev) => prev.filter((_, i) => i !== idx));
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
                          setUrls((prev) => prev.filter((_, i) => i !== idx));
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
          <div className='flex justify-center pt-1'>
            <Button
              theme='solid'
              type='primary'
              loading={submitting}
              onClick={onSubmit}
            >
              {showReapplyAfterRevoked ? t('重新提交审核') : t('提交申请')}
            </Button>
          </div>
        </Form>
      </Spin>
    );

  return (
    <div className='distributor-apply-page-root'>
      <div className='mx-auto flex h-full min-h-0 w-full min-w-0 max-w-6xl flex-1 flex-col'>
        <Card className='distributor-apply-main-card !rounded-2xl flex h-full min-h-0 w-full min-w-0 flex-1 flex-col !overflow-hidden !shadow-none'>
          <div className='distributor-apply-card-stacked-scroll flex min-h-0 min-w-0 flex-1 flex-col overflow-y-auto overflow-x-hidden overscroll-y-contain lg:flex-row lg:overflow-hidden'>
            <section className='distributor-apply-panels-divider flex min-w-0 max-lg:shrink-0 flex-col border-b border-solid lg:min-h-0 lg:min-w-0 lg:flex-[3] lg:border-b-0 lg:border-r'>
              <div className='shrink-0 p-4 pb-2 md:px-5 md:pt-5'>
                <header className='distributor-apply-hero relative rounded-xl p-[12px]'>
                  <span
                    className='distributor-apply-hero-orb distributor-apply-hero-orb--a'
                    aria-hidden
                  />
                  <span
                    className='distributor-apply-hero-orb distributor-apply-hero-orb--b'
                    aria-hidden
                  />
                  <div className='relative z-[1] min-w-0'>
                    <div className='flex min-w-0 items-start justify-between gap-2 sm:gap-3'>
                      <div className='flex min-w-0 flex-1 items-start gap-3'>
                        <div
                          className='distributor-apply-hero-icon flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl border border-solid bg-[var(--semi-color-bg-0)] text-[var(--semi-color-primary)] shadow-sm'
                          aria-hidden
                        >
                          <IconUserGroup size={28} />
                        </div>
                        <Typography.Title
                          heading={3}
                          className='distributor-apply-hero-title !mb-0 !mt-0 min-w-0 !font-semibold !tracking-tight !leading-snug sm:!leading-tight'
                        >
                          {t('代理伙伴招募')}
                        </Typography.Title>
                      </div>
                      {showCsColumn ? (
                        <Popover
                          content={csPopoverContent}
                          trigger='click'
                          position='leftTop'
                          showArrow
                        >
                          <Button
                            theme='outline'
                            type='primary'
                            size='small'
                            className='shrink-0 !font-medium'
                          >
                            {t('联系客服')}
                          </Button>
                        </Popover>
                      ) : null}
                    </div>
                  </div>
                </header>
              </div>
              <div className='distributor-apply-intro-scroll min-w-0 px-4 pb-4 pt-0 md:px-5 md:pb-5 max-lg:overflow-visible lg:min-h-0 lg:flex-1 lg:overflow-y-auto lg:overflow-x-hidden lg:overscroll-y-contain'>
                {applyIntroHtml ? (
                  <div
                    className='distributor-apply-intro-html max-w-full break-words text-[15px] leading-relaxed text-[var(--semi-color-text-0)] [overflow-wrap:anywhere] [&_p]:mb-2 [&_p]:last:mb-0 [&_img]:!max-w-full [&_img]:h-auto [&_pre]:max-w-full [&_pre]:overflow-x-auto [&_table]:!max-w-full [&_table]:table-fixed [&_ul]:mb-2 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:mb-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_a]:break-all [&_a]:text-[var(--semi-color-primary)] [&_blockquote]:border-l-2 [&_blockquote]:border-[var(--semi-color-border)] [&_blockquote]:pl-3'
                    dangerouslySetInnerHTML={{ __html: applyIntroHtml }}
                  />
                ) : null}
                <Text
                  type='tertiary'
                  className='distributor-apply-hero-desc mt-2 !block !text-[15px] !leading-relaxed [overflow-wrap:anywhere]'
                >
                  {t(
                    '请如实填写以下信息，审核通过后即可获得代理资格与邀请分成。',
                  )}
                </Text>
              </div>
            </section>

            <section className='flex min-w-0 max-lg:shrink-0 flex-col p-4 md:p-5 max-lg:overflow-visible lg:min-h-0 lg:min-w-0 lg:flex-[2] lg:overflow-y-auto lg:overscroll-y-contain [scrollbar-gutter:stable]'>
              {rightBody}
            </section>
          </div>
        </Card>
      </div>

      <Modal
        title={t('预览')}
        visible={Boolean(previewUrl)}
        onCancel={() => setPreviewUrl(null)}
        footer={null}
        width={Math.min(
          960,
          typeof window !== 'undefined' ? window.innerWidth - 48 : 960,
        )}
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
