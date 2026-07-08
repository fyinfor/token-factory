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
  Button,
  Card,
  Checkbox,
  Col,
  DatePicker,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Select,
  Slider,
  Space,
  Switch,
  Tabs,
  TextArea,
  Typography,
  Upload,
} from '@douyinfe/semi-ui';
import {
  ChevronDown,
  ChevronUp,
  Copy,
  Image as ImageIcon,
  Monitor,
  Palette,
  Plus,
  Save,
  Smartphone,
  Trash2,
  UploadCloud,
} from 'lucide-react';
import Cropper from 'react-easy-crop';
import 'react-easy-crop/react-easy-crop.css';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const ENABLED_KEY = 'HomeHeroCarouselEnabled';
const SLIDES_KEY = 'HomeHeroCarouselSlides';
const INTERVAL_KEY = 'HomeHeroCarouselIntervalSec';
const ASPECT_KEY = 'HomeHeroCarouselAspectRatio';
const DEFAULT_INTERVAL_SEC = 5;
const MIN_INTERVAL_SEC = 2;
const MAX_INTERVAL_SEC = 60;
const DEFAULT_OVERLAY_OPACITY = 0.15;
const DATE_TIME_FORMAT = 'yyyy-MM-dd HH:mm:ss';

const CONTENT_ALIGN_OPTIONS = [
  { label: '居左', value: 'left' },
  { label: '居中', value: 'center' },
  { label: '居右', value: 'right' },
];

const PC_PROFILE = {
  key: 'pc',
  imageKey: 'img_pc',
  label: 'PC 主图',
  helper:
    '按 6.4:1 比例裁剪，不强制导出固定尺寸；建议上传 1920x300 或同等比例大图',
  aspectLabel: '6.4:1',
  width: 1920,
  height: 300,
  maxBytes: 3 * 1024 * 1024,
  icon: <Monitor size={16} />,
};

const MOBILE_PROFILE = {
  key: 'mobile',
  imageKey: 'img_mobile',
  label: '移动端图',
  helper: '750 x 360，移动端独立构图，避免从 PC 图硬裁文字',
  width: 750,
  height: 360,
  maxBytes: 1200 * 1024,
  icon: <Smartphone size={16} />,
};

const HERO_COLOR_PRESETS = [
  {
    key: 'clean-white',
    name: '清透白字',
    description: '通用展示图，干净耐看',
    title_color: '#ffffff',
    subtitle_color: '#e5e7eb',
    button_color: '#ffffff',
    button_text_color: '#111827',
    overlay_opacity: 0.3,
    preview_from: '#1f2937',
    preview_to: '#64748b',
  },
  {
    key: 'tech-blue',
    name: '科技蓝',
    description: '适合产品、能力、平台',
    title_color: '#eef6ff',
    subtitle_color: '#bfdbfe',
    button_color: '#1677ff',
    button_text_color: '#ffffff',
    overlay_opacity: 0.34,
    preview_from: '#0f172a',
    preview_to: '#1d4ed8',
  },
  {
    key: 'warm-gold',
    name: '暖金行动',
    description: '适合促销、推荐、活动',
    title_color: '#fff7ed',
    subtitle_color: '#fed7aa',
    button_color: '#f59e0b',
    button_text_color: '#111827',
    overlay_opacity: 0.36,
    preview_from: '#451a03',
    preview_to: '#b45309',
  },
  {
    key: 'dark-minimal',
    name: '暗色极简',
    description: '适合浅色图、人物图',
    title_color: '#f9fafb',
    subtitle_color: '#d1d5db',
    button_color: '#111827',
    button_text_color: '#ffffff',
    overlay_opacity: 0.42,
    preview_from: '#020617',
    preview_to: '#334155',
  },
  {
    key: 'fresh-green',
    name: '清新绿',
    description: '适合合规、安全、服务',
    title_color: '#f0fdf4',
    subtitle_color: '#bbf7d0',
    button_color: '#16a34a',
    button_text_color: '#ffffff',
    overlay_opacity: 0.32,
    preview_from: '#064e3b',
    preview_to: '#15803d',
  },
  {
    key: 'custom',
    name: '自定义配色',
    description: '手动调整标题、副标题和按钮',
    custom: true,
    preview_from: '#111827',
    preview_to: '#475569',
  },
];

const makeSlideId = () =>
  `hero-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;

function cssUrl(url) {
  return `url("${String(url).replace(/"/g, '\\"')}")`;
}

function normalizeContentAlign(value) {
  return ['left', 'center', 'right'].includes(value) ? value : 'left';
}

function contentAlignClass(value) {
  const align = normalizeContentAlign(value);
  if (align === 'center') {
    return 'items-center text-center';
  }
  if (align === 'right') {
    return 'items-end text-right';
  }
  return 'items-start text-left';
}

const clamp = (value, min, max, fallback) => {
  const n = Number(value);
  if (!Number.isFinite(n)) {
    return fallback;
  }
  return Math.min(max, Math.max(min, n));
};

const toOverlayOpacity = (value) =>
  clamp(value, 0, 0.8, DEFAULT_OVERLAY_OPACITY);
const toOverlayPercent = (value) => Math.round(toOverlayOpacity(value) * 100);
const percentToOverlayOpacity = (value) =>
  clamp(Number(value) / 100, 0, 0.8, DEFAULT_OVERLAY_OPACITY);

const pad2 = (value) => String(value).padStart(2, '0');

function formatDateTimeValue(value) {
  if (!value) {
    return '';
  }
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())} ${pad2(date.getHours())}:${pad2(date.getMinutes())}:${pad2(date.getSeconds())}`;
}

function toDatePickerValue(value) {
  if (!value) {
    return undefined;
  }
  const text = String(value).trim();
  if (!text) {
    return undefined;
  }
  const normalized =
    text.includes(' ') && !text.includes('T') ? text.replace(' ', 'T') : text;
  const date = new Date(normalized);
  return Number.isNaN(date.getTime()) ? undefined : date;
}

const emptySlide = (sort = 1) => ({
  id: makeSlideId(),
  enabled: true,
  status: 1,
  sort,
  title: '',
  subtitle: '',
  badge_text: '',
  button_text: '',
  content_align: 'left',
  link_url: '',
  open_mode: 'same',
  img_pc: '',
  img_mobile: '',
  image_url: '',
  overlay_opacity: DEFAULT_OVERLAY_OPACITY,
  text_color: '#ffffff',
  title_color: '#ffffff',
  subtitle_color: '#e5e7eb',
  button_color: '#ffffff',
  button_text_color: '#111827',
  background_color: '#111827',
  start_at: '',
  end_at: '',
});

const normalizeSlide = (item, index = 0) => {
  const legacyImage = String(item?.image_url || '').trim();
  const enabled =
    item?.enabled === false ||
    item?.enabled === 'false' ||
    item?.status === 0 ||
    item?.status === '0'
      ? false
      : true;

  return {
    ...emptySlide(index + 1),
    id: String(item?.id || '').trim() || makeSlideId(),
    enabled,
    status: enabled ? 1 : 0,
    sort: Math.max(1, Math.round(Number(item?.sort) || index + 1)),
    title: String(item?.title || '').trim(),
    subtitle: String(item?.subtitle || '').trim(),
    badge_text: String(item?.badge_text || item?.badge || '').trim(),
    button_text: String(item?.button_text || '').trim(),
    content_align: normalizeContentAlign(item?.content_align),
    link_url: String(item?.link_url || '').trim(),
    open_mode: item?.open_mode === 'blank' ? 'blank' : 'same',
    img_pc: String(item?.img_pc || legacyImage).trim(),
    img_mobile: String(item?.img_mobile || '').trim(),
    image_url: legacyImage,
    overlay_opacity: toOverlayOpacity(item?.overlay_opacity),
    text_color: String(item?.text_color || '#ffffff').trim(),
    title_color: String(
      item?.title_color || item?.text_color || '#ffffff',
    ).trim(),
    subtitle_color: String(
      item?.subtitle_color || item?.text_color || '#e5e7eb',
    ).trim(),
    button_color: String(item?.button_color || '#ffffff').trim(),
    button_text_color: String(item?.button_text_color || '#111827').trim(),
    background_color: String(item?.background_color || '#111827').trim(),
    start_at: String(item?.start_at || '').trim(),
    end_at: String(item?.end_at || '').trim(),
    note: String(item?.note || '').trim(),
  };
};

const normalizeSlides = (rawSlides) =>
  rawSlides.map(normalizeSlide).sort((a, b) => a.sort - b.sort);

const parseSlides = (raw) => {
  if (!raw || typeof raw !== 'string') {
    return [];
  }
  try {
    const value = JSON.parse(raw);
    return Array.isArray(value) ? normalizeSlides(value) : [];
  } catch {
    return [];
  }
};

const syncSort = (items) =>
  items.map((item, index) => ({
    ...item,
    sort: index + 1,
    status: item.enabled ? 1 : 0,
  }));

const slideHasContent = (slide) =>
  Boolean(
    slide.img_pc ||
    slide.img_mobile ||
    slide.image_url ||
    slide.title ||
    slide.subtitle ||
    slide.badge_text ||
    slide.button_text,
  );

const stringifySlides = (slides) => {
  const cleaned = syncSort(slides)
    .filter(slideHasContent)
    .map((slide) => ({
      id: slide.id,
      enabled: Boolean(slide.enabled),
      status: slide.enabled ? 1 : 0,
      sort: slide.sort,
      title: String(slide.title || '').trim(),
      subtitle: String(slide.subtitle || '').trim(),
      badge_text: String(slide.badge_text || '').trim(),
      button_text: String(slide.button_text || '').trim(),
      content_align: normalizeContentAlign(slide.content_align),
      link_url: String(slide.link_url || '').trim(),
      open_mode: slide.open_mode === 'blank' ? 'blank' : 'same',
      img_pc: String(slide.img_pc || slide.image_url || '').trim(),
      img_mobile: String(slide.img_mobile || '').trim(),
      overlay_opacity: toOverlayOpacity(slide.overlay_opacity),
      text_color: String(slide.text_color || '#ffffff').trim(),
      title_color: String(
        slide.title_color || slide.text_color || '#ffffff',
      ).trim(),
      subtitle_color: String(
        slide.subtitle_color || slide.text_color || '#e5e7eb',
      ).trim(),
      button_color: String(slide.button_color || '#ffffff').trim(),
      button_text_color: String(slide.button_text_color || '#111827').trim(),
      background_color: String(slide.background_color || '#111827').trim(),
      start_at: String(slide.start_at || '').trim(),
      end_at: String(slide.end_at || '').trim(),
      note: String(slide.note || '').trim(),
    }));
  return JSON.stringify(cleaned);
};

const clampInterval = (value) =>
  String(
    Math.min(
      MAX_INTERVAL_SEC,
      Math.max(MIN_INTERVAL_SEC, Number(value) || DEFAULT_INTERVAL_SEC),
    ),
  );

const toBlob = (canvas, type, quality) =>
  new Promise((resolve) => canvas.toBlob(resolve, type, quality));

const canvasToOptimizedBlob = async (canvas, maxBytes) => {
  const webpQualities = [0.96, 0.92, 0.88, 0.84];
  const minWebpQuality = webpQualities[webpQualities.length - 1];
  for (const quality of webpQualities) {
    const blob = await toBlob(canvas, 'image/webp', quality);
    if (blob && (blob.size <= maxBytes || quality === minWebpQuality)) {
      return blob;
    }
  }

  const jpgQualities = [0.96, 0.92, 0.88, 0.84];
  const minJpgQuality = jpgQualities[jpgQualities.length - 1];
  for (const quality of jpgQualities) {
    const blob = await toBlob(canvas, 'image/jpeg', quality);
    if (blob && (blob.size <= maxBytes || quality === minJpgQuality)) {
      return blob;
    }
  }

  return null;
};

function normalizeCropArea(area, image) {
  const imageWidth = image.naturalWidth || image.width;
  const imageHeight = image.naturalHeight || image.height;
  const width = Math.min(imageWidth, Math.max(1, Math.round(area.width)));
  const height = Math.min(imageHeight, Math.max(1, Math.round(area.height)));

  return {
    x: Math.min(imageWidth - width, Math.max(0, Math.round(area.x))),
    y: Math.min(imageHeight - height, Math.max(0, Math.round(area.y))),
    width,
    height,
  };
}

const makeCroppedFile = (
  cropState,
  cropAreaPixels = cropState.croppedAreaPixels,
) =>
  new Promise((resolve, reject) => {
    if (!cropAreaPixels) {
      reject(new Error('crop area unavailable'));
      return;
    }

    const image = new Image();
    image.onload = async () => {
      try {
        const area = normalizeCropArea(cropAreaPixels, image);
        const outputWidth =
          cropState.profile.key === 'pc' ? area.width : cropState.profile.width;
        const outputHeight =
          cropState.profile.key === 'pc'
            ? area.height
            : cropState.profile.height;
        const canvas = document.createElement('canvas');
        canvas.width = outputWidth;
        canvas.height = outputHeight;
        const ctx = canvas.getContext('2d');
        if (!ctx) {
          reject(new Error('canvas context unavailable'));
          return;
        }
        ctx.imageSmoothingEnabled = true;
        ctx.imageSmoothingQuality = 'high';

        ctx.drawImage(
          image,
          area.x,
          area.y,
          area.width,
          area.height,
          0,
          0,
          outputWidth,
          outputHeight,
        );

        const blob = await canvasToOptimizedBlob(
          canvas,
          cropState.profile.maxBytes,
        );
        if (!blob) {
          reject(new Error('crop failed'));
          return;
        }
        const baseName = String(cropState.file?.name || 'home-hero')
          .replace(/\.[^.]+$/, '')
          .replace(/[^\w.-]+/g, '-');
        const ext = blob.type === 'image/webp' ? 'webp' : 'jpg';
        resolve(
          new File([blob], `${baseName}-${cropState.profile.key}.${ext}`, {
            type: blob.type,
          }),
        );
      } catch (error) {
        reject(error);
      }
    };
    image.onerror = () => reject(new Error('image load failed'));
    image.src = cropState.objectUrl;
  });

function ColorField({ label, value, onChange }) {
  return (
    <div className='min-w-[132px] flex-1'>
      <Text strong className='!mb-1 !block'>
        {label}
      </Text>
      <div className='flex h-9 items-center gap-2 rounded-md border border-semi-color-border bg-semi-color-bg-1 px-2'>
        <input
          type='color'
          value={value || '#ffffff'}
          onChange={(event) => onChange(event.target.value)}
          className='h-6 w-8 cursor-pointer border-0 bg-transparent p-0'
        />
        <Input value={value} onChange={onChange} size='small' />
      </div>
    </div>
  );
}

function CropResultPreview({ cropState }) {
  const { t } = useTranslation();
  const canvasRef = useRef(null);
  const areaPixels = cropState?.croppedAreaPixels;
  const profile = cropState?.profile;

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || !cropState?.objectUrl || !areaPixels || !profile) {
      return;
    }

    let disposed = false;
    const image = new Image();
    image.onload = () => {
      if (disposed) {
        return;
      }

      const area = normalizeCropArea(areaPixels, image);
      const previewWidth = profile.key === 'pc' ? 960 : 360;
      const previewHeight = Math.round(
        previewWidth * (profile.height / profile.width),
      );
      const ratio = window.devicePixelRatio || 1;
      canvas.width = Math.round(previewWidth * ratio);
      canvas.height = Math.round(previewHeight * ratio);
      canvas.style.width = `${previewWidth}px`;
      canvas.style.height = `${previewHeight}px`;

      const ctx = canvas.getContext('2d');
      if (!ctx) {
        return;
      }
      ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
      ctx.clearRect(0, 0, previewWidth, previewHeight);
      ctx.imageSmoothingEnabled = true;
      ctx.imageSmoothingQuality = 'high';
      ctx.drawImage(
        image,
        area.x,
        area.y,
        area.width,
        area.height,
        0,
        0,
        previewWidth,
        previewHeight,
      );
    };
    image.src = cropState.objectUrl;

    return () => {
      disposed = true;
    };
  }, [areaPixels, cropState?.objectUrl, profile]);

  if (!cropState) {
    return null;
  }

  return (
    <div className='w-full rounded-md border border-semi-color-border bg-semi-color-bg-0 p-3'>
      <div className='mb-2 flex items-center justify-between gap-2'>
        <Text strong size='small'>
          {t('裁剪结果预览')}
        </Text>
        <Text type='tertiary' size='small'>
          {profile?.aspectLabel || `${profile?.width} x ${profile?.height}`}
        </Text>
      </div>
      <div className='flex w-full justify-center overflow-hidden rounded-md bg-[#111827]'>
        <canvas
          ref={canvasRef}
          className='block max-w-full'
          style={{
            aspectRatio: `${profile.width} / ${profile.height}`,
          }}
        />
      </div>
    </div>
  );
}

function HeroColorPresetPanel({ slide, customActive, onApply, onCustom }) {
  const { t } = useTranslation();
  const imageUrl = slide.img_pc || slide.image_url || slide.img_mobile || '';
  const alignClass = contentAlignClass(slide.content_align);

  return (
    <div className='rounded-md border border-semi-color-border bg-semi-color-bg-0 p-3'>
      <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
        <Space spacing='tight'>
          <Palette size={16} />
          <Text strong>{t('配色预设')}</Text>
        </Space>
        <Text type='tertiary' size='small'>
          {t('点击预览卡片即可套用到当前轮播')}
        </Text>
      </div>
      <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3'>
        {HERO_COLOR_PRESETS.map((preset) => {
          const colors = preset.custom
            ? {
                title_color: slide.title_color || slide.text_color || '#ffffff',
                subtitle_color:
                  slide.subtitle_color || slide.text_color || '#e5e7eb',
                button_color: slide.button_color || '#ffffff',
                button_text_color: slide.button_text_color || '#111827',
                overlay_opacity: toOverlayOpacity(slide.overlay_opacity),
                preview_from: slide.background_color || preset.preview_from,
                preview_to: '#475569',
              }
            : preset;
          const backgroundImage = imageUrl
            ? cssUrl(imageUrl)
            : `linear-gradient(135deg, ${colors.preview_from}, ${colors.preview_to})`;
          return (
            <button
              key={preset.key}
              type='button'
              className={`group overflow-hidden rounded-md border bg-semi-color-bg-1 p-0 text-left transition hover:border-semi-color-primary hover:shadow-md ${
                preset.custom && customActive
                  ? 'border-semi-color-primary'
                  : 'border-semi-color-border'
              }`}
              onClick={() => (preset.custom ? onCustom() : onApply(preset))}
            >
              <div
                className='relative h-32 overflow-hidden'
                style={{
                  backgroundImage,
                  backgroundPosition: 'center',
                  backgroundSize: 'cover',
                }}
              >
                <div
                  className='absolute inset-0'
                  style={{
                    backgroundColor: `rgba(0,0,0,${colors.overlay_opacity})`,
                  }}
                />
                <div
                  className={`relative z-[1] flex h-full flex-col justify-center px-4 py-3 ${alignClass}`}
                >
                  <div
                    className='text-lg font-semibold leading-tight'
                    style={{ color: colors.title_color }}
                  >
                    {slide.title || t('标题预览')}
                  </div>
                  <div
                    className='mt-1 line-clamp-2 text-xs leading-relaxed'
                    style={{ color: colors.subtitle_color }}
                  >
                    {slide.subtitle || t('副标题展示效果')}
                  </div>
                  <span
                    className='mt-3 inline-flex w-fit items-center rounded-md px-3 py-1 text-xs font-semibold'
                    style={{
                      backgroundColor: colors.button_color,
                      color: colors.button_text_color,
                    }}
                  >
                    {slide.button_text || t('按钮')}
                  </span>
                </div>
              </div>
              <div className='flex items-center justify-between gap-2 px-3 py-2'>
                <div>
                  <Text strong size='small'>
                    {t(preset.name)}
                  </Text>
                  <div className='text-xs text-semi-color-text-2'>
                    {t(preset.description)}
                  </div>
                </div>
                <Text type='tertiary' size='small'>
                  {toOverlayPercent(colors.overlay_opacity)}%
                </Text>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function ImageUploadPanel({
  profile,
  slide,
  loading,
  onOpenCropper,
  onUpdate,
}) {
  const { t } = useTranslation();
  const imageUrl = slide[profile.imageKey] || '';

  return (
    <div className='rounded-md border border-semi-color-border bg-semi-color-bg-0 p-3'>
      <div className='mb-2 flex items-center justify-between gap-2'>
        <Space spacing='tight'>
          {profile.icon}
          <Text strong>{t(profile.label)}</Text>
        </Space>
        <Text type='tertiary' size='small'>
          {profile.aspectLabel
            ? t('比例 {{ratio}}', { ratio: profile.aspectLabel })
            : `${profile.width} x ${profile.height}`}
        </Text>
      </div>

      <div
        className='mb-3 flex w-full items-center justify-center overflow-hidden rounded-md border border-semi-color-border bg-semi-color-fill-0'
        style={{ aspectRatio: `${profile.width} / ${profile.height}` }}
      >
        {imageUrl ? (
          <img
            src={imageUrl}
            alt=''
            className='h-full w-full object-cover object-center'
          />
        ) : (
          <div className='flex flex-col items-center gap-2 text-semi-color-text-2'>
            <ImageIcon size={22} />
            <Text type='tertiary' size='small'>
              {t('暂无图片')}
            </Text>
          </div>
        )}
      </div>

      <Space vertical align='start' spacing='tight' style={{ width: '100%' }}>
        <Upload
          action=''
          accept='image/*'
          showUploadList={false}
          customRequest={onOpenCropper(profile)}
        >
          <Button icon={<UploadCloud size={16} />} loading={loading} block>
            {t('裁剪上传')}
          </Button>
        </Upload>
        <Input
          value={imageUrl}
          placeholder={t('也可以直接粘贴图片地址')}
          onChange={(value) => onUpdate({ [profile.imageKey]: value })}
        />
        <Text type='tertiary' size='small'>
          {t(profile.helper)}
        </Text>
      </Space>
    </div>
  );
}

export default function HomeHeroCarouselSetting() {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState(false);
  const [intervalSec, setIntervalSec] = useState(String(DEFAULT_INTERVAL_SEC));
  const [slides, setSlides] = useState([]);
  const [activeSlideId, setActiveSlideId] = useState('');
  const [selectedIds, setSelectedIds] = useState([]);
  const [customColorSlideIds, setCustomColorSlideIds] = useState([]);
  const [loading, setLoading] = useState(false);
  const [cropState, setCropState] = useState(null);
  const cropAreaPixelsRef = useRef(null);

  const selectedSet = useMemo(() => new Set(selectedIds), [selectedIds]);
  const customColorSet = useMemo(
    () => new Set(customColorSlideIds),
    [customColorSlideIds],
  );
  const selectedCount = selectedIds.length;
  const enabledCount = slides.filter((slide) => slide.enabled).length;

  useEffect(() => {
    if (slides.length === 0) {
      if (activeSlideId) {
        setActiveSlideId('');
      }
      return;
    }
    if (!slides.some((slide) => slide.id === activeSlideId)) {
      setActiveSlideId(slides[0].id);
    }
  }, [activeSlideId, slides]);

  const loadOptions = async () => {
    try {
      const res = await API.get('/api/option/');
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载设置失败'));
        return;
      }
      const optionMap = {};
      (Array.isArray(data) ? data : []).forEach((item) => {
        optionMap[item.key] = item.value;
      });
      setEnabled(optionMap[ENABLED_KEY] === 'true');
      setIntervalSec(optionMap[INTERVAL_KEY] || String(DEFAULT_INTERVAL_SEC));
      const nextSlides = parseSlides(optionMap[SLIDES_KEY] || '[]');
      setSlides(nextSlides);
      setActiveSlideId(nextSlides[0]?.id || '');
      setCustomColorSlideIds([]);
      setSelectedIds([]);
    } catch (error) {
      showError(error?.message || t('加载设置失败'));
    }
  };

  useEffect(() => {
    loadOptions();
  }, []);

  const updateSlide = (id, patch) => {
    setSlides((items) =>
      items.map((item) =>
        item.id === id
          ? {
              ...item,
              ...patch,
              status:
                patch.enabled == null
                  ? item.enabled
                    ? 1
                    : 0
                  : patch.enabled
                    ? 1
                    : 0,
            }
          : item,
      ),
    );
  };

  const addSlide = () => {
    const slide = emptySlide(slides.length + 1);
    setSlides((items) => [...items, slide]);
    setActiveSlideId(slide.id);
  };

  const duplicateSlide = (slide) => {
    const copied = {
      ...slide,
      id: makeSlideId(),
      title: slide.title ? `${slide.title} Copy` : '',
    };
    setSlides((items) => syncSort([...items, copied]));
    setActiveSlideId(copied.id);
  };

  const removeSlide = (id) => {
    setSlides((items) => {
      const index = items.findIndex((item) => item.id === id);
      const next = syncSort(items.filter((item) => item.id !== id));
      if (activeSlideId === id) {
        setActiveSlideId(next[Math.min(index, next.length - 1)]?.id || '');
      }
      return next;
    });
    setSelectedIds((items) => items.filter((item) => item !== id));
    setCustomColorSlideIds((items) => items.filter((item) => item !== id));
  };

  const removeSelected = () => {
    setSlides((items) => {
      const next = syncSort(items.filter((item) => !selectedSet.has(item.id)));
      if (selectedSet.has(activeSlideId)) {
        setActiveSlideId(next[0]?.id || '');
      }
      return next;
    });
    setSelectedIds([]);
    setCustomColorSlideIds((items) =>
      items.filter((item) => !selectedSet.has(item)),
    );
  };

  const moveSlide = (index, offset) => {
    setSlides((items) => {
      const nextIndex = index + offset;
      if (nextIndex < 0 || nextIndex >= items.length) {
        return items;
      }
      const next = [...items];
      const [item] = next.splice(index, 1);
      next.splice(nextIndex, 0, item);
      return syncSort(next);
    });
  };

  const uploadFile = async (slideId, profile, file) => {
    setLoading(true);
    try {
      const fd = new FormData();
      fd.append('file', file);
      const res = await API.post('/api/oss/upload', fd, {
        skipErrorHandler: true,
      });
      const { success, message, data } = res.data || {};
      const url = data?.url;
      if (!success || !url) {
        throw new Error(message || t('上传失败'));
      }
      updateSlide(slideId, {
        [profile.imageKey]: url,
        ...(profile.key === 'pc' ? { image_url: url } : {}),
      });
      showSuccess(t('图片上传成功，请点击保存设置'));
      return data;
    } finally {
      setLoading(false);
    }
  };

  const openCropper =
    (slideId) =>
    (profile) =>
    ({ file, onSuccess, onError }) => {
      const inst = file?.fileInstance || file;
      if (!inst) {
        onError?.(new Error('no file'));
        return;
      }

      const objectUrl = URL.createObjectURL(inst);
      cropAreaPixelsRef.current = null;
      setCropState({
        visible: true,
        slideId,
        profile,
        file: inst,
        objectUrl,
        crop: { x: 0, y: 0 },
        zoom: 1,
        croppedAreaPixels: null,
        onSuccess,
        onError,
      });
    };

  const closeCropper = (notifyCancel = false) => {
    if (cropState?.objectUrl) {
      URL.revokeObjectURL(cropState.objectUrl);
    }
    if (notifyCancel) {
      cropState?.onError?.(new Error('cancelled'));
    }
    setCropState(null);
    cropAreaPixelsRef.current = null;
  };

  const handleCropComplete = useCallback((_, croppedAreaPixels) => {
    cropAreaPixelsRef.current = croppedAreaPixels;
    setCropState((state) => (state ? { ...state, croppedAreaPixels } : state));
  }, []);

  const confirmCrop = async () => {
    if (!cropState) {
      return;
    }
    try {
      const croppedFile = await makeCroppedFile(
        cropState,
        cropAreaPixelsRef.current || cropState.croppedAreaPixels,
      );
      const data = await uploadFile(
        cropState.slideId,
        cropState.profile,
        croppedFile,
      );
      cropState.onSuccess?.(data);
      closeCropper(false);
    } catch (error) {
      cropState.onError?.(error);
      showError(error?.message || t('裁剪上传失败'));
    }
  };

  const applyColorPreset = (slideId, preset) => {
    setCustomColorSlideIds((items) => items.filter((item) => item !== slideId));
    updateSlide(slideId, {
      text_color: preset.title_color,
      title_color: preset.title_color,
      subtitle_color: preset.subtitle_color,
      button_color: preset.button_color,
      button_text_color: preset.button_text_color,
      overlay_opacity: preset.overlay_opacity,
    });
  };

  const showCustomColorFields = (slideId) => {
    setCustomColorSlideIds((items) =>
      items.includes(slideId) ? items : [...items, slideId],
    );
  };

  const save = async () => {
    try {
      setLoading(true);
      const normalizedInterval = clampInterval(intervalSec);
      const slidesValue = stringifySlides(slides);
      const requests = [
        API.put('/api/option/', {
          key: ENABLED_KEY,
          value: String(enabled),
        }),
        API.put('/api/option/', {
          key: INTERVAL_KEY,
          value: normalizedInterval,
        }),
        API.put('/api/option/', {
          key: ASPECT_KEY,
          value: '1920:300',
        }),
        API.put('/api/option/', {
          key: SLIDES_KEY,
          value: slidesValue,
        }),
      ];
      const results = await Promise.all(requests);
      const failed = results.find((res) => !res.data?.success);
      if (failed) {
        throw new Error(failed.data?.message || t('保存失败'));
      }
      setIntervalSec(normalizedInterval);
      const savedSlides = parseSlides(slidesValue);
      setSlides(savedSlides);
      setActiveSlideId(
        savedSlides.some((slide) => slide.id === activeSlideId)
          ? activeSlideId
          : savedSlides[0]?.id || '',
      );
      setSelectedIds([]);
      showSuccess(t('首页主轮播设置已保存'));
    } catch (error) {
      showError(error?.message || t('保存失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card
      title={t('首页沉浸主轮播')}
      style={{ marginTop: 16, marginBottom: 16 }}
      headerExtraContent={
        <Button
          icon={<Save size={16} />}
          type='primary'
          loading={loading}
          onClick={save}
        >
          {t('保存设置')}
        </Button>
      }
    >
      <Space vertical align='start' spacing='medium' style={{ width: '100%' }}>
        <div className='grid w-full grid-cols-1 gap-3 rounded-md border border-semi-color-border bg-semi-color-fill-0 p-4 lg:grid-cols-[1fr_auto]'>
          <Space vertical align='start' spacing='tight'>
            <Space wrap>
              <Switch checked={enabled} onChange={setEnabled} />
              <Text strong>{t('启用首页沉浸主轮播')}</Text>
              <Text type='tertiary' size='small'>
                {t('当前 {{count}} 张，{{enabledCount}} 张上架', {
                  count: slides.length,
                  enabledCount,
                })}
              </Text>
            </Space>
            <Text type='tertiary' size='small'>
              {t(
                'PC 按 6.4:1 比例裁剪（如 1920x300），移动端使用 750x360；核心文字、按钮、LOGO 请放在安全区内。',
              )}
            </Text>
          </Space>
          <Space wrap>
            <Text type='tertiary' size='small'>
              {t('轮播间隔')}
            </Text>
            <InputNumber
              min={MIN_INTERVAL_SEC}
              max={MAX_INTERVAL_SEC}
              value={Number(intervalSec)}
              onChange={(value) =>
                setIntervalSec(String(value || DEFAULT_INTERVAL_SEC))
              }
              style={{ width: 112 }}
            />
            <Text type='tertiary' size='small'>
              {t('秒')}
            </Text>
            <Button icon={<Plus size={16} />} theme='light' onClick={addSlide}>
              {t('新增轮播')}
            </Button>
            <Popconfirm
              title={t('确定删除已选择的轮播吗？')}
              onConfirm={removeSelected}
              disabled={selectedCount === 0}
            >
              <Button
                icon={<Trash2 size={16} />}
                type='danger'
                theme='light'
                disabled={selectedCount === 0}
              >
                {t('批量删除')}
              </Button>
            </Popconfirm>
          </Space>
        </div>

        {slides.length === 0 ? (
          <div className='flex min-h-[180px] w-full flex-col items-center justify-center gap-3 rounded-md border border-dashed border-semi-color-border bg-semi-color-fill-0'>
            <ImageIcon size={30} className='text-semi-color-text-2' />
            <Text type='tertiary'>{t('还没有主轮播，先新增一张开始配置')}</Text>
            <Button icon={<Plus size={16} />} type='primary' onClick={addSlide}>
              {t('新增轮播')}
            </Button>
          </div>
        ) : (
          <Tabs
            type='card'
            activeKey={activeSlideId}
            onChange={setActiveSlideId}
            className='w-full'
          >
            {slides.map((slide, index) => (
              <Tabs.TabPane
                key={slide.id}
                itemKey={slide.id}
                tab={
                  <span className='inline-block max-w-[150px] truncate align-bottom'>
                    {index + 1}. {slide.title || t('未命名轮播')}
                  </span>
                }
              >
                <div className='w-full rounded-md border border-semi-color-border bg-semi-color-bg-1 p-4'>
                  <div className='mb-4 flex flex-wrap items-center justify-between gap-3'>
                    <Space wrap>
                      <Checkbox
                        checked={selectedSet.has(slide.id)}
                        onChange={(event) => {
                          const checked = event.target.checked;
                          setSelectedIds((items) =>
                            checked
                              ? [...items, slide.id]
                              : items.filter((id) => id !== slide.id),
                          );
                        }}
                      />
                      <span className='rounded-md bg-semi-color-fill-0 px-2 py-1 text-xs font-semibold text-semi-color-text-1'>
                        #{index + 1}
                      </span>
                      <Switch
                        checked={slide.enabled}
                        onChange={(checked) =>
                          updateSlide(slide.id, { enabled: checked })
                        }
                      />
                      <Text strong>{slide.title || t('未命名轮播')}</Text>
                    </Space>
                    <Space wrap>
                      <Button
                        icon={<ChevronUp size={16} />}
                        theme='borderless'
                        disabled={index === 0}
                        onClick={() => moveSlide(index, -1)}
                      />
                      <Button
                        icon={<ChevronDown size={16} />}
                        theme='borderless'
                        disabled={index === slides.length - 1}
                        onClick={() => moveSlide(index, 1)}
                      />
                      <Button
                        icon={<Copy size={16} />}
                        theme='borderless'
                        onClick={() => duplicateSlide(slide)}
                      >
                        {t('复制')}
                      </Button>
                      <Popconfirm
                        title={t('确定删除这张轮播吗？')}
                        position='left'
                        onConfirm={() => removeSlide(slide.id)}
                      >
                        <Button
                          icon={<Trash2 size={16} />}
                          type='danger'
                          theme='borderless'
                        >
                          {t('删除')}
                        </Button>
                      </Popconfirm>
                    </Space>
                  </div>

                  <Row gutter={[16, 16]}>
                    <Col xs={24} lg={12}>
                      <ImageUploadPanel
                        profile={PC_PROFILE}
                        slide={slide}
                        loading={loading}
                        onOpenCropper={openCropper(slide.id)}
                        onUpdate={(patch) => updateSlide(slide.id, patch)}
                      />
                    </Col>
                    <Col xs={24} lg={12}>
                      <ImageUploadPanel
                        profile={MOBILE_PROFILE}
                        slide={slide}
                        loading={loading}
                        onOpenCropper={openCropper(slide.id)}
                        onUpdate={(patch) => updateSlide(slide.id, patch)}
                      />
                    </Col>
                  </Row>

                  <Row gutter={[16, 12]} style={{ marginTop: 16 }}>
                    <Col xs={24} md={12}>
                      <Text strong className='!mb-1 !block'>
                        {t('标题')}
                      </Text>
                      <Input
                        value={slide.title}
                        placeholder={t('例如：安全可靠的 AI 能力平台')}
                        onChange={(value) =>
                          updateSlide(slide.id, { title: value })
                        }
                      />
                    </Col>
                    <Col xs={24} md={12}>
                      <Text strong className='!mb-1 !block'>
                        {t('副标题')}
                      </Text>
                      <Input
                        value={slide.subtitle}
                        placeholder={t('一句话说明当前轮播重点')}
                        onChange={(value) =>
                          updateSlide(slide.id, { subtitle: value })
                        }
                      />
                    </Col>
                    <Col xs={24} md={8}>
                      <Text strong className='!mb-1 !block'>
                        {t('优惠标签')}
                      </Text>
                      <Input
                        value={slide.badge_text}
                        placeholder={t('限时 / 新品 / 推荐')}
                        onChange={(value) =>
                          updateSlide(slide.id, { badge_text: value })
                        }
                      />
                    </Col>
                    <Col xs={24} md={8}>
                      <Text strong className='!mb-1 !block'>
                        {t('按钮文案')}
                      </Text>
                      <Input
                        value={slide.button_text}
                        placeholder={t('立即体验')}
                        onChange={(value) =>
                          updateSlide(slide.id, { button_text: value })
                        }
                      />
                    </Col>
                    <Col xs={24} md={8}>
                      <Text strong className='!mb-1 !block'>
                        {t('内容对齐')}
                      </Text>
                      <Select
                        value={slide.content_align}
                        onChange={(value) =>
                          updateSlide(slide.id, {
                            content_align: normalizeContentAlign(value),
                          })
                        }
                        optionList={CONTENT_ALIGN_OPTIONS.map((item) => ({
                          ...item,
                          label: t(item.label),
                        }))}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col xs={24} md={16}>
                      <Text strong className='!mb-1 !block'>
                        {t('跳转链接')}
                      </Text>
                      <Input
                        value={slide.link_url}
                        placeholder={t('为空时不跳转')}
                        onChange={(value) =>
                          updateSlide(slide.id, { link_url: value })
                        }
                      />
                    </Col>
                    <Col xs={24} md={8}>
                      <Text strong className='!mb-1 !block'>
                        {t('打开方式')}
                      </Text>
                      <Select
                        value={slide.open_mode}
                        onChange={(value) =>
                          updateSlide(slide.id, { open_mode: value })
                        }
                        optionList={[
                          { label: t('当前窗口'), value: 'same' },
                          { label: t('新窗口'), value: 'blank' },
                        ]}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col xs={24} md={8}>
                      <Text strong className='!mb-1 !block'>
                        {t('排序数字')}
                      </Text>
                      <InputNumber
                        min={1}
                        value={slide.sort}
                        onChange={(value) =>
                          updateSlide(slide.id, {
                            sort: Number(value) || index + 1,
                          })
                        }
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col xs={24} md={8}>
                      <Text strong className='!mb-1 !block'>
                        {t('自动上架时间')}
                      </Text>
                      <DatePicker
                        type='dateTime'
                        format={DATE_TIME_FORMAT}
                        value={toDatePickerValue(slide.start_at)}
                        placeholder='2026-07-01 00:00'
                        showClear
                        onChange={(value) => {
                          updateSlide(slide.id, {
                            start_at: formatDateTimeValue(value),
                          });
                        }}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col xs={24} md={8}>
                      <Text strong className='!mb-1 !block'>
                        {t('自动下架时间')}
                      </Text>
                      <DatePicker
                        type='dateTime'
                        format={DATE_TIME_FORMAT}
                        value={toDatePickerValue(slide.end_at)}
                        placeholder='2026-07-31 23:59'
                        showClear
                        onChange={(value) => {
                          updateSlide(slide.id, {
                            end_at: formatDateTimeValue(value),
                          });
                        }}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col xs={24}>
                      <HeroColorPresetPanel
                        slide={slide}
                        customActive={customColorSet.has(slide.id)}
                        onApply={(preset) => applyColorPreset(slide.id, preset)}
                        onCustom={() => showCustomColorFields(slide.id)}
                      />
                    </Col>
                    {customColorSet.has(slide.id) ? (
                      <Col xs={24}>
                        <div className='grid grid-cols-1 gap-3 rounded-md border border-semi-color-border bg-semi-color-fill-0 p-3 md:grid-cols-2 xl:grid-cols-6'>
                          <div className='min-w-[132px] flex-1'>
                            <Text strong className='!mb-1 !block'>
                              {t('遮罩透明度')}
                            </Text>
                            <div className='flex h-9 items-center gap-2'>
                              <InputNumber
                                min={0}
                                max={80}
                                step={5}
                                value={toOverlayPercent(slide.overlay_opacity)}
                                onChange={(value) =>
                                  updateSlide(slide.id, {
                                    overlay_opacity:
                                      percentToOverlayOpacity(value),
                                  })
                                }
                                style={{ width: 92 }}
                              />
                              <span className='text-sm text-semi-color-text-2'>
                                %
                              </span>
                            </div>
                            <Text type='tertiary' size='small'>
                              {t('默认 15%，数值越大图片越暗')}
                            </Text>
                          </div>
                          <ColorField
                            label={t('标题颜色')}
                            value={slide.title_color}
                            onChange={(value) =>
                              updateSlide(slide.id, {
                                text_color: value,
                                title_color: value,
                              })
                            }
                          />
                          <ColorField
                            label={t('副标题颜色')}
                            value={slide.subtitle_color}
                            onChange={(value) =>
                              updateSlide(slide.id, { subtitle_color: value })
                            }
                          />
                          <ColorField
                            label={t('按钮颜色')}
                            value={slide.button_color}
                            onChange={(value) =>
                              updateSlide(slide.id, { button_color: value })
                            }
                          />
                          <ColorField
                            label={t('按钮文字')}
                            value={slide.button_text_color}
                            onChange={(value) =>
                              updateSlide(slide.id, {
                                button_text_color: value,
                              })
                            }
                          />
                          <ColorField
                            label={t('无图底色')}
                            value={slide.background_color}
                            onChange={(value) =>
                              updateSlide(slide.id, {
                                background_color: value,
                              })
                            }
                          />
                        </div>
                      </Col>
                    ) : null}
                    <Col xs={24}>
                      <Text strong className='!mb-1 !block'>
                        {t('备注')}
                      </Text>
                      <TextArea
                        rows={2}
                        value={slide.note || ''}
                        placeholder={t(
                          '仅后台管理可见，可记录素材来源或投放说明',
                        )}
                        onChange={(value) =>
                          updateSlide(slide.id, { note: value })
                        }
                      />
                    </Col>
                  </Row>
                </div>
              </Tabs.TabPane>
            ))}
          </Tabs>
        )}
      </Space>

      <Modal
        title={
          cropState
            ? t('裁剪 {{label}}', { label: t(cropState.profile.label) })
            : t('裁剪图片')
        }
        visible={Boolean(cropState?.visible)}
        onCancel={() => closeCropper(true)}
        onOk={confirmCrop}
        confirmLoading={loading}
        okText={t('裁剪并上传')}
        cancelText={t('取消')}
        width={1280}
        style={{ maxWidth: '96vw' }}
      >
        {cropState ? (
          <Space
            vertical
            align='start'
            spacing='medium'
            style={{ width: '100%' }}
          >
            <Text type='tertiary' size='small'>
              {t(
                '图片会按当前比例裁剪并铺满画面，不会导出背景空白。拖动图片或缩放即可调整，PC 图会显示中间安全区和左右裁切区参考线。',
              )}
            </Text>
            <div
              style={{
                position: 'relative',
                width:
                  cropState.profile.key === 'pc' ? '100%' : 'min(100%, 750px)',
                ...(cropState.profile.key === 'pc'
                  ? { height: 'min(360px, 52vh)', minHeight: 260 }
                  : {
                      aspectRatio: `${cropState.profile.width} / ${cropState.profile.height}`,
                    }),
                margin: '0 auto',
                background: '#111827',
                overflow: 'hidden',
                border: '1px solid var(--semi-color-border)',
                borderRadius: 6,
              }}
            >
              <Cropper
                image={cropState.objectUrl}
                crop={cropState.crop}
                zoom={cropState.zoom}
                aspect={cropState.profile.width / cropState.profile.height}
                minZoom={1}
                maxZoom={6}
                objectFit='cover'
                restrictPosition
                showGrid
                cropShape='rect'
                onCropChange={(crop) =>
                  setCropState((state) => (state ? { ...state, crop } : state))
                }
                onZoomChange={(zoom) =>
                  setCropState((state) => (state ? { ...state, zoom } : state))
                }
                onCropComplete={handleCropComplete}
              />
              {cropState.profile.key === 'pc' ? (
                <div className='pointer-events-none absolute inset-0 z-[2]'>
                  <div
                    className='absolute border border-dashed border-white/70'
                    style={{
                      left: '18.75%',
                      right: '18.75%',
                      top: 0,
                      bottom: 0,
                    }}
                  />
                  <div
                    className='absolute bottom-0 left-0 top-0 border-r border-dashed border-white/35 bg-white/5'
                    style={{ width: '18.75%' }}
                  />
                  <div
                    className='absolute bottom-0 right-0 top-0 border-l border-dashed border-white/35 bg-white/5'
                    style={{ width: '18.75%' }}
                  />
                  <div className='absolute left-[18.75%] top-2 rounded bg-black/40 px-2 py-1 text-xs text-white'>
                    {t('1200px 安全区')}
                  </div>
                </div>
              ) : null}
            </div>
            <CropResultPreview cropState={cropState} />
            <div style={{ width: '100%' }}>
              <Text type='tertiary' size='small'>
                {t('缩放')}
              </Text>
              <Slider
                min={1}
                max={6}
                step={0.01}
                value={cropState.zoom}
                onChange={(value) =>
                  setCropState((state) =>
                    state ? { ...state, zoom: Number(value) } : state,
                  )
                }
              />
            </div>
          </Space>
        ) : null}
      </Modal>
    </Card>
  );
}
