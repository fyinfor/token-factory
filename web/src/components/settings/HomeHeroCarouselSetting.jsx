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

import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Col,
  Input,
  Modal,
  Popconfirm,
  Row,
  Slider,
  Space,
  Switch,
  Upload,
} from '@douyinfe/semi-ui';
import {
  IconArrowDown,
  IconArrowUp,
  IconDelete,
  IconPlus,
  IconUpload,
} from '@douyinfe/semi-icons';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import Cropper from 'react-easy-crop';
import 'react-easy-crop/react-easy-crop.css';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const ENABLED_KEY = 'HomeHeroCarouselEnabled';
const SLIDES_KEY = 'HomeHeroCarouselSlides';
const INTERVAL_KEY = 'HomeHeroCarouselIntervalSec';
const ASPECT_KEY = 'HomeHeroCarouselAspectRatio';
const DEFAULT_ASPECT_TEXT = '16:5';
const CROP_OUTPUT_WIDTH = 1920;

const emptySlide = () => ({
  image_url: '',
  link_url: '',
});

const parseAspectRatio = (raw) => {
  const text = String(raw || '').trim();
  if (!text) {
    return null;
  }

  const pair = text.match(/^(\d+(?:\.\d+)?)\s*[:/]\s*(\d+(?:\.\d+)?)$/);
  if (pair) {
    const w = Number(pair[1]);
    const h = Number(pair[2]);
    if (w > 0 && h > 0) {
      return w / h;
    }
  }

  const number = Number(text);
  if (Number.isFinite(number) && number > 0) {
    return number;
  }
  return null;
};

const parseSlides = (raw) => {
  if (!raw || typeof raw !== 'string') {
    return [];
  }
  try {
    const value = JSON.parse(raw);
    if (!Array.isArray(value)) {
      return [];
    }
    return value.map((item) => ({
      image_url: String(item?.image_url || '').trim(),
      link_url: String(item?.link_url || '').trim(),
    }));
  } catch {
    return [];
  }
};

const stringifySlides = (slides) => {
  const cleaned = slides
    .map((slide) => ({
      image_url: String(slide?.image_url || '').trim(),
      link_url: String(slide?.link_url || '').trim(),
    }))
    .filter((slide) => slide.image_url);
  return JSON.stringify(cleaned);
};

const clampInterval = (value) =>
  String(Math.min(60, Math.max(2, Number(value) || 5)));

const makeCroppedFile = (cropState, aspect) =>
  new Promise((resolve, reject) => {
    if (!cropState.croppedAreaPixels) {
      reject(new Error('crop area unavailable'));
      return;
    }

    const image = new Image();
    image.onload = () => {
      const canvas = document.createElement('canvas');
      const outputHeight = Math.max(1, Math.round(CROP_OUTPUT_WIDTH / aspect));
      canvas.width = CROP_OUTPUT_WIDTH;
      canvas.height = outputHeight;
      const ctx = canvas.getContext('2d');
      if (!ctx) {
        reject(new Error('canvas context unavailable'));
        return;
      }

      const area = cropState.croppedAreaPixels;
      ctx.drawImage(
        image,
        area.x,
        area.y,
        area.width,
        area.height,
        0,
        0,
        CROP_OUTPUT_WIDTH,
        outputHeight,
      );

      canvas.toBlob(
        (blob) => {
          if (!blob) {
            reject(new Error('crop failed'));
            return;
          }
          const baseName = String(cropState.file?.name || 'home-hero')
            .replace(/\.[^.]+$/, '')
            .replace(/[^\w.-]+/g, '-');
          resolve(
            new File([blob], `${baseName}-crop.jpg`, { type: blob.type }),
          );
        },
        'image/jpeg',
        0.92,
      );
    };
    image.onerror = () => reject(new Error('image load failed'));
    image.src = cropState.objectUrl;
  });

export default function HomeHeroCarouselSetting() {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState(false);
  const [intervalSec, setIntervalSec] = useState('5');
  const [aspectText, setAspectText] = useState(DEFAULT_ASPECT_TEXT);
  const [slides, setSlides] = useState([]);
  const [loading, setLoading] = useState(false);
  const [cropState, setCropState] = useState(null);

  const aspect = useMemo(
    () => parseAspectRatio(aspectText) || parseAspectRatio(DEFAULT_ASPECT_TEXT),
    [aspectText],
  );
  const aspectCss = useMemo(() => `${aspect} / 1`, [aspect]);
  const slideCountText = useMemo(
    () => t('{{count}} 张', { count: slides.length }),
    [slides.length, t],
  );

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
      setIntervalSec(optionMap[INTERVAL_KEY] || '5');
      setAspectText(optionMap[ASPECT_KEY] || DEFAULT_ASPECT_TEXT);
      setSlides(parseSlides(optionMap[SLIDES_KEY] || '[]'));
    } catch (error) {
      showError(error?.message || t('加载设置失败'));
    }
  };

  useEffect(() => {
    loadOptions();
  }, []);

  const updateSlide = (index, key, value) => {
    setSlides((items) =>
      items.map((item, i) => (i === index ? { ...item, [key]: value } : item)),
    );
  };

  const addSlide = () => {
    setSlides((items) => [...items, emptySlide()]);
  };

  const removeSlide = (index) => {
    setSlides((items) => items.filter((_, i) => i !== index));
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
      return next;
    });
  };

  const uploadFile = async (index, file) => {
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
      updateSlide(index, 'image_url', url);
      showSuccess(t('图片上传成功，请点击保存设置'));
      return data;
    } finally {
      setLoading(false);
    }
  };

  const openCropper =
    (index) =>
    ({ file, onSuccess, onError }) => {
      const inst = file?.fileInstance || file;
      if (!inst) {
        onError?.(new Error('no file'));
        return;
      }

      const currentAspect = parseAspectRatio(aspectText);
      if (!currentAspect) {
        const err = new Error(t('请先填写有效裁剪比例'));
        onError?.(err);
        showError(err.message);
        return;
      }

      const objectUrl = URL.createObjectURL(inst);
      setCropState({
        visible: true,
        index,
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
  };

  const confirmCrop = async () => {
    if (!cropState) {
      return;
    }
    const currentAspect = parseAspectRatio(aspectText);
    if (!currentAspect) {
      showError(t('请先填写有效裁剪比例'));
      return;
    }
    try {
      const croppedFile = await makeCroppedFile(cropState, currentAspect);
      const data = await uploadFile(cropState.index, croppedFile);
      cropState.onSuccess?.(data);
      closeCropper(false);
    } catch (error) {
      cropState.onError?.(error);
      showError(error?.message || t('裁剪上传失败'));
    }
  };

  const save = async () => {
    try {
      setLoading(true);
      const normalizedInterval = clampInterval(intervalSec);
      const normalizedAspectText = String(aspectText || '').trim();
      if (!parseAspectRatio(normalizedAspectText)) {
        throw new Error(t('裁剪比例格式不正确'));
      }
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
          value: normalizedAspectText,
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
      setAspectText(normalizedAspectText);
      setSlides(parseSlides(slidesValue));
      showSuccess(t('首页大图沉浸轮播设置已保存'));
    } catch (error) {
      showError(error?.message || t('保存失败'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card
      title={t('首页大图沉浸轮播')}
      style={{ marginTop: 16, marginBottom: 16 }}
    >
      <Space vertical align='start' spacing='medium' style={{ width: '100%' }}>
        <Space wrap style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space align='center' wrap>
            <Text>{t('启用')}</Text>
            <Switch checked={enabled} onChange={setEnabled} />
            <Text type='tertiary' size='small'>
              {t('当前 {{countText}}，单图静态展示，多图自动轮播', {
                countText: slideCountText,
              })}
            </Text>
          </Space>
          <Space align='center' wrap>
            <Text type='tertiary' size='small'>
              {t('轮播间隔（秒）')}
            </Text>
            <Input
              type='number'
              min={2}
              max={60}
              value={intervalSec}
              onChange={setIntervalSec}
              style={{ width: 96 }}
            />
            <Text type='tertiary' size='small'>
              {t('裁剪比例')}
            </Text>
            <Input
              value={aspectText}
              placeholder='16:5'
              onChange={setAspectText}
              style={{ width: 112 }}
            />
            <Button icon={<IconPlus />} theme='light' onClick={addSlide}>
              {t('添加图片')}
            </Button>
            <Button type='primary' loading={loading} onClick={save}>
              {t('保存设置')}
            </Button>
          </Space>
        </Space>

        <Text type='tertiary' size='small'>
          {t(
            '每张图只有图片和跳转链接；跳转链接为空时不可点击。裁剪比例可填 16:5、3:1、1920:600 或 3.2。',
          )}
        </Text>

        {slides.length === 0 ? (
          <Button icon={<IconPlus />} theme='light' onClick={addSlide}>
            {t('添加第一张图片')}
          </Button>
        ) : (
          <Space
            vertical
            align='start'
            spacing='medium'
            style={{ width: '100%' }}
          >
            {slides.map((slide, index) => (
              <Card
                key={index}
                title={`${t('图片')} #${index + 1}`}
                style={{ width: '100%' }}
                headerExtraContent={
                  <Space spacing='tight'>
                    <Button
                      icon={<IconArrowUp />}
                      disabled={index === 0}
                      theme='borderless'
                      onClick={() => moveSlide(index, -1)}
                    />
                    <Button
                      icon={<IconArrowDown />}
                      disabled={index === slides.length - 1}
                      theme='borderless'
                      onClick={() => moveSlide(index, 1)}
                    />
                    <Popconfirm
                      title={t('确定删除这张图片？')}
                      position='left'
                      onConfirm={() => removeSlide(index)}
                    >
                      <Button
                        icon={<IconDelete />}
                        type='danger'
                        theme='borderless'
                      />
                    </Popconfirm>
                  </Space>
                }
              >
                <Row gutter={16}>
                  <Col xs={24} md={6}>
                    {slide.image_url ? (
                      <img
                        src={slide.image_url}
                        alt={t('轮播图预览')}
                        style={{
                          width: '100%',
                          aspectRatio: aspectCss,
                          objectFit: 'cover',
                          borderRadius: 8,
                          border: '1px solid var(--semi-color-border)',
                          background: 'var(--semi-color-fill-0)',
                        }}
                      />
                    ) : (
                      <div
                        style={{
                          width: '100%',
                          aspectRatio: aspectCss,
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                          borderRadius: 8,
                          border: '1px dashed var(--semi-color-border)',
                          color: 'var(--semi-color-text-2)',
                          background: 'var(--semi-color-fill-0)',
                        }}
                      >
                        {t('暂无图片')}
                      </div>
                    )}
                    <Upload
                      action=''
                      accept='image/*'
                      showUploadList={false}
                      customRequest={openCropper(index)}
                    >
                      <Button
                        icon={<IconUpload />}
                        loading={loading}
                        style={{ marginTop: 8, width: '100%' }}
                      >
                        {t('裁剪上传')}
                      </Button>
                    </Upload>
                  </Col>
                  <Col xs={24} md={18}>
                    <Space
                      vertical
                      align='start'
                      spacing='tight'
                      style={{ width: '100%' }}
                    >
                      <Input
                        value={slide.image_url}
                        placeholder={t('图片地址')}
                        onChange={(value) =>
                          updateSlide(index, 'image_url', value)
                        }
                      />
                      <Input
                        value={slide.link_url}
                        placeholder={t('跳转链接（可留空）')}
                        onChange={(value) =>
                          updateSlide(index, 'link_url', value)
                        }
                      />
                    </Space>
                  </Col>
                </Row>
              </Card>
            ))}
          </Space>
        )}
      </Space>

      <Modal
        title={t('裁剪首页大图')}
        visible={Boolean(cropState?.visible)}
        onCancel={() => closeCropper(true)}
        onOk={confirmCrop}
        confirmLoading={loading}
        okText={t('裁剪并上传')}
        cancelText={t('取消')}
        style={{ width: 820, maxWidth: '94vw' }}
      >
        {cropState ? (
          <Space
            vertical
            align='start'
            spacing='medium'
            style={{ width: '100%' }}
          >
            <Text type='tertiary' size='small'>
              {t('拖动图片调整裁剪区域，滚轮或缩放条可调整大小。')}
            </Text>
            <div
              style={{
                position: 'relative',
                width: '100%',
                height: 420,
                background: '#111',
                borderRadius: 8,
                overflow: 'hidden',
                border: '1px solid var(--semi-color-border)',
              }}
            >
              <Cropper
                image={cropState.objectUrl}
                crop={cropState.crop}
                zoom={cropState.zoom}
                aspect={aspect}
                onCropChange={(crop) =>
                  setCropState((state) => (state ? { ...state, crop } : state))
                }
                onZoomChange={(zoom) =>
                  setCropState((state) => (state ? { ...state, zoom } : state))
                }
                onCropComplete={(_, croppedAreaPixels) =>
                  setCropState((state) =>
                    state ? { ...state, croppedAreaPixels } : state,
                  )
                }
              />
            </div>
            <div style={{ width: '100%' }}>
              <Text type='tertiary' size='small'>
                {t('缩放')}
              </Text>
              <Slider
                min={1}
                max={4}
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
