import { useEffect, useMemo, useState } from 'react';
import { API, showError, showSuccess } from '../../../../helpers';
import {
    hasTierRule,
    normalizeTierRule,
    serializeTierRule,
    summarizeTierRule,
    validateTierRule,
} from '../utils/requestTierPricing';

export const PAGE_SIZE = 10;
export const PRICE_SUFFIX = '$/1M';
const EMPTY_CANDIDATE_MODEL_NAMES = [];

const EMPTY_MODEL = {
    name: '',
    billingMode: 'per-token',
    fixedPrice: '',
    inputPrice: '',
    completionPrice: '',
    lockedCompletionRatio: '',
    completionRatioLocked: false,
    cachePrice: '',
    createCachePrice: '',
    imagePrice: '',
    audioInputPrice: '',
    audioOutputPrice: '',
    videoBillingMode: 'per-token',
    videoFixedPrice: '',
    videoTextToVideoRules: [],
    videoImageToVideoRules: [],
    videoUploadRules: [],
    videoGenerateRules: [],
    videoSimilarityThreshold: '',
    rawRatios: {
        modelRatio: '',
        completionRatio: '',
        cacheRatio: '',
        createCacheRatio: '',
        imageRatio: '',
        audioRatio: '',
        audioCompletionRatio: '',
        videoRatio: '',
        videoCompletionRatio: '',
        videoPrice: '',
        videoPricingRules: null,
        requestTierPricing: null,
    },
    requestTierRule: null,
    hasConflict: false,
};

const NUMERIC_INPUT_REGEX = /^(\d+(\.\d*)?|\.\d*)?$/;

export const hasValue = (value) =>
    value !== '' && value !== null && value !== undefined && value !== false;

const toNumericString = (value) => {
    if (!hasValue(value) && value !== 0) {
        return '';
    }
    const num = Number(value);
    return Number.isFinite(num) ? String(num) : '';
};

const toNumberOrNull = (value) => {
    if (!hasValue(value) && value !== 0) {
        return null;
    }
    const num = Number(value);
    return Number.isFinite(num) ? num : null;
};

const formatNumber = (value) => {
    const num = toNumberOrNull(value);
    if (num === null) {
        return '';
    }
    const rounded = Number(num.toFixed(12));
    if (Math.abs(rounded) < 1e-12) {
        return '0';
    }
    const nearestInt = Math.round(rounded);
    // Absorb floating-point epsilon like 33.000000000008 -> 33.
    if (Math.abs(rounded - nearestInt) < 1e-9) {
        return String(nearestInt);
    }
    return rounded.toString();
};

const toNormalizedNumber = (value) => {
    const formatted = formatNumber(value);
    return formatted === '' ? null : Number(formatted);
};

const parseOptionJSON = (rawValue) => {
    if (!rawValue || rawValue.trim() === '') {
        return {};
    }
    try {
        const parsed = JSON.parse(rawValue);
        return parsed && typeof parsed === 'object' ? parsed : {};
    } catch (error) {
        console.error('JSON解析错误:', error);
        return {};
    }
};

const VIDEO_RESOLUTION_REGEX = /^\s*\d+\s*x\s*\d+\s*$/i;
const DEFAULT_TEXT_VIDEO_PIXEL_COMPRESSION = '384';
const DEFAULT_IMAGE_VIDEO_PIXEL_COMPRESSION = '512';
const DEFAULT_VIDEO_VIDEO_PIXEL_COMPRESSION = '384';

const normalizeVideoRuleRow = (row) => ({
    resolution: row?.resolution || '',
    tokenPrice: toNumericString(row?.token_price),
    pixelCompression: toNumericString(row?.pixel_compression),
});

const normalizePerVideoRuleRow = (row) => ({
    resolution: row?.resolution || '',
    videoPrice: toNumericString(row?.video_price),
});

const parseVideoPricingRules = (rawRules) => {
    if (!rawRules || typeof rawRules !== 'object' || Array.isArray(rawRules)) {
        return {
            textToVideo: [],
            imageToVideo: [],
            videoUpload: [],
            videoGenerate: [],
            textToVideoPerVideo: [],
            imageToVideoPerVideo: [],
            videoUploadPerVideo: [],
            videoGeneratePerVideo: [],
            similarityThreshold: '',
        };
    }
    const textToVideo = Array.isArray(rawRules.text_to_video)
        ? rawRules.text_to_video.map(normalizeVideoRuleRow)
        : [];
    const videoToVideo = Array.isArray(rawRules.video_to_video)
        ? rawRules.video_to_video.map(normalizeVideoRuleRow)
        : [];
    const imageToVideoRules = Array.isArray(rawRules.image_to_video_rules)
        ? rawRules.image_to_video_rules.map(normalizeVideoRuleRow)
        : [];
    const videoToVideoInput = Array.isArray(rawRules.video_to_video_input)
        ? rawRules.video_to_video_input.map(normalizeVideoRuleRow)
        : [];
    const videoToVideoOutput = Array.isArray(rawRules.video_to_video_output)
        ? rawRules.video_to_video_output.map(normalizeVideoRuleRow)
        : [];
    const textToVideoPerVideo = Array.isArray(rawRules.text_to_video_per_video)
        ? rawRules.text_to_video_per_video.map(normalizePerVideoRuleRow)
        : [];
    const imageToVideoPerVideo = Array.isArray(rawRules.image_to_video_per_video)
        ? rawRules.image_to_video_per_video.map(normalizePerVideoRuleRow)
        : [];
    const videoToVideoInputPerVideo = Array.isArray(
        rawRules.video_to_video_input_per_video,
    )
        ? rawRules.video_to_video_input_per_video.map(normalizePerVideoRuleRow)
        : [];
    const videoToVideoOutputPerVideo = Array.isArray(
        rawRules.video_to_video_output_per_video,
    )
        ? rawRules.video_to_video_output_per_video.map(normalizePerVideoRuleRow)
        : [];
    const imageRule = rawRules.image_to_video || null;
    return {
        textToVideo,
        imageToVideo:
            imageToVideoRules.length > 0
                ? imageToVideoRules
                : imageRule && imageRule.token_price
                    ? [
                        {
                            resolution: '1280x720',
                            tokenPrice: toNumericString(imageRule.token_price),
                            pixelCompression: toNumericString(
                                imageRule.pixel_compression,
                            ),
                        },
                    ]
                    : [],
        videoUpload: videoToVideoInput,
        videoGenerate:
            videoToVideoOutput.length > 0 ? videoToVideoOutput : videoToVideo,
        textToVideoPerVideo,
        imageToVideoPerVideo,
        videoUploadPerVideo: videoToVideoInputPerVideo,
        videoGeneratePerVideo: videoToVideoOutputPerVideo,
        similarityThreshold: toNumericString(rawRules.similarity_threshold),
    };
};

const ratioToBasePrice = (ratio) => {
    const num = toNumberOrNull(ratio);
    if (num === null) return '';
    return formatNumber(num * 2);
};

const normalizeCompletionRatioMeta = (rawMeta) => {
    if (!rawMeta || typeof rawMeta !== 'object' || Array.isArray(rawMeta)) {
        return {
            locked: false,
            ratio: '',
        };
    }

    return {
        locked: Boolean(rawMeta.locked),
        ratio: toNumericString(rawMeta.ratio),
    };
};

const buildModelState = (name, sourceMaps) => {
    const modelRatio = toNumericString(sourceMaps.ModelRatio[name]);
    const completionRatio = toNumericString(sourceMaps.CompletionRatio[name]);
    const completionRatioMeta = normalizeCompletionRatioMeta(
        sourceMaps.CompletionRatioMeta?.[name],
    );
    const cacheRatio = toNumericString(sourceMaps.CacheRatio[name]);
    const createCacheRatio = toNumericString(sourceMaps.CreateCacheRatio[name]);
    const imageRatio = toNumericString(sourceMaps.ImageRatio[name]);
    const audioRatio = toNumericString(sourceMaps.AudioRatio[name]);
    const audioCompletionRatio = toNumericString(
        sourceMaps.AudioCompletionRatio[name],
    );
    const videoRatio = toNumericString(sourceMaps.VideoRatio[name]);
    const videoCompletionRatio = toNumericString(
        sourceMaps.VideoCompletionRatio[name],
    );
    const videoPrice = toNumericString(sourceMaps.VideoPrice[name]);
    const videoPricingRules = parseVideoPricingRules(
        sourceMaps.VideoPricingRules?.[name],
    );
    const requestTierRule = sourceMaps.RequestTierPricing?.[name]
        ? normalizeTierRule(sourceMaps.RequestTierPricing[name])
        : null;
    const hasPerVideoTable =
        videoPricingRules.textToVideoPerVideo.length > 0 ||
        videoPricingRules.imageToVideoPerVideo.length > 0 ||
        videoPricingRules.videoUploadPerVideo.length > 0 ||
        videoPricingRules.videoGeneratePerVideo.length > 0;
    const fixedPrice = toNumericString(sourceMaps.ModelPrice[name]);
    const inputPrice = ratioToBasePrice(modelRatio);
    const inputPriceNumber = toNumberOrNull(inputPrice);
    const audioInputPrice =
        inputPriceNumber !== null && hasValue(audioRatio)
            ? formatNumber(inputPriceNumber * Number(audioRatio))
            : '';
    const videoInputPrice =
        inputPriceNumber !== null && hasValue(videoRatio)
            ? formatNumber(inputPriceNumber * Number(videoRatio))
            : '';
    const videoOutputPrice =
        toNumberOrNull(videoInputPrice) !== null && hasValue(videoCompletionRatio)
            ? formatNumber(Number(videoInputPrice) * Number(videoCompletionRatio))
            : '';
    const useLegacyRulesFallback =
        !hasPerVideoTable &&
        videoPricingRules.textToVideo.length === 0 &&
        videoPricingRules.videoGenerate.length === 0 &&
        videoPricingRules.videoUpload.length === 0 &&
        videoPricingRules.imageToVideo.length === 0 &&
        hasValue(videoInputPrice);

    return {
        ...EMPTY_MODEL,
        name,
        billingMode: hasValue(fixedPrice) ? 'per-request' : 'per-token',
        fixedPrice,
        inputPrice,
        completionRatioLocked: completionRatioMeta.locked,
        lockedCompletionRatio: completionRatioMeta.ratio,
        completionPrice:
            inputPriceNumber !== null &&
                hasValue(
                    completionRatioMeta.locked
                        ? completionRatioMeta.ratio
                        : completionRatio,
                )
                ? formatNumber(
                    inputPriceNumber *
                    Number(
                        completionRatioMeta.locked
                            ? completionRatioMeta.ratio
                            : completionRatio,
                    ),
                )
                : '',
        cachePrice:
            inputPriceNumber !== null && hasValue(cacheRatio)
                ? formatNumber(inputPriceNumber * Number(cacheRatio))
                : '',
        createCachePrice:
            inputPriceNumber !== null && hasValue(createCacheRatio)
                ? formatNumber(inputPriceNumber * Number(createCacheRatio))
                : '',
        imagePrice:
            inputPriceNumber !== null && hasValue(imageRatio)
                ? formatNumber(inputPriceNumber * Number(imageRatio))
                : '',
        audioInputPrice,
        audioOutputPrice:
            toNumberOrNull(audioInputPrice) !== null && hasValue(audioCompletionRatio)
                ? formatNumber(Number(audioInputPrice) * Number(audioCompletionRatio))
                : '',
        videoBillingMode: hasPerVideoTable
            ? 'per-video'
            : hasValue(videoPrice)
                ? 'per-video'
                : 'per-token',
        videoFixedPrice: hasPerVideoTable ? '' : videoPrice,
        videoTextToVideoRules: hasPerVideoTable
            ? videoPricingRules.textToVideoPerVideo
            : useLegacyRulesFallback
                ? [
                    {
                        resolution: '1280x720',
                        tokenPrice: videoInputPrice,
                        pixelCompression: DEFAULT_TEXT_VIDEO_PIXEL_COMPRESSION,
                    },
                ]
                : videoPricingRules.textToVideo,
        videoImageToVideoRules: hasPerVideoTable
            ? videoPricingRules.imageToVideoPerVideo
            : useLegacyRulesFallback
                ? [
                    {
                        resolution: '1280x720',
                        tokenPrice: videoInputPrice,
                        pixelCompression: DEFAULT_IMAGE_VIDEO_PIXEL_COMPRESSION,
                    },
                ]
                : videoPricingRules.imageToVideo,
        videoUploadRules: hasPerVideoTable
            ? videoPricingRules.videoUploadPerVideo
            : useLegacyRulesFallback
                ? [
                    {
                        resolution: '1280x720',
                        tokenPrice: videoInputPrice,
                        pixelCompression: DEFAULT_VIDEO_VIDEO_PIXEL_COMPRESSION,
                    },
                ]
                : videoPricingRules.videoUpload,
        videoGenerateRules: hasPerVideoTable
            ? videoPricingRules.videoGeneratePerVideo
            : useLegacyRulesFallback
                ? [
                    {
                        resolution: '1280x720',
                        tokenPrice: videoOutputPrice || videoInputPrice,
                        pixelCompression: DEFAULT_VIDEO_VIDEO_PIXEL_COMPRESSION,
                    },
                ]
                : videoPricingRules.videoGenerate,
        videoSimilarityThreshold: videoPricingRules.similarityThreshold,
        rawRatios: {
            modelRatio,
            completionRatio,
            cacheRatio,
            createCacheRatio,
            imageRatio,
            audioRatio,
            audioCompletionRatio,
            videoRatio,
            videoCompletionRatio,
            videoPrice,
            videoPricingRules:
                sourceMaps.VideoPricingRules?.[name] &&
                    typeof sourceMaps.VideoPricingRules[name] === 'object'
                    ? sourceMaps.VideoPricingRules[name]
                    : null,
            requestTierPricing: requestTierRule,
        },
        requestTierRule,
        hasConflict:
            hasValue(fixedPrice) &&
            [
                modelRatio,
                completionRatio,
                cacheRatio,
                createCacheRatio,
                imageRatio,
                audioRatio,
                audioCompletionRatio,
                videoRatio,
                videoCompletionRatio,
                videoPrice,
                requestTierRule && hasTierRule(requestTierRule) ? 'request_tier' : '',
            ].some(hasValue),
    };
};

export const isBasePricingUnset = (model) =>
    !hasValue(model.fixedPrice) && !hasValue(model.inputPrice);

const hasDuplicateResolution = (rows) => {
    const seen = new Set();
    for (const row of rows || []) {
        const key = String(row?.resolution || '')
            .replace(/\s+/g, '')
            .toLowerCase();
        if (!key) continue;
        if (seen.has(key)) return true;
        seen.add(key);
    }
    return false;
};

export const getModelWarnings = (model, t) => {
    if (!model) {
        return [];
    }
    const warnings = [];
    const hasDerivedPricing = [
        model.inputPrice,
        model.completionPrice,
        model.cachePrice,
        model.createCachePrice,
        model.imagePrice,
        model.audioInputPrice,
        model.audioOutputPrice,
        model.videoTextToVideoRules?.length > 0 ? 'video_text' : '',
        model.videoImageToVideoRules?.length > 0 ? 'video_image' : '',
        model.videoUploadRules?.length > 0 ? 'video_upload' : '',
        model.videoGenerateRules?.length > 0 ? 'video_generate' : '',
    ].some(hasValue);

    if (model.hasConflict) {
        warnings.push(
            t('当前模型同时存在按次价格和倍率配置，保存时会按当前计费方式覆盖。'),
        );
    }
    if (model.billingMode === 'per-request' && hasTierRule(model.requestTierRule)) {
        warnings.push(t('当前模型为固定价格计费，阶梯计费规则会被保留但不会生效。'));
    }

    if (
        !hasValue(model.inputPrice) &&
        [
            model.rawRatios.completionRatio,
            model.rawRatios.cacheRatio,
            model.rawRatios.createCacheRatio,
            model.rawRatios.imageRatio,
            model.rawRatios.audioRatio,
            model.rawRatios.audioCompletionRatio,
            model.rawRatios.videoRatio,
            model.rawRatios.videoCompletionRatio,
        ].some(hasValue)
    ) {
        warnings.push(
            t(
                '当前模型存在未显式设置输入倍率的扩展倍率；填写输入价格后会自动换算为价格字段。',
            ),
        );
    }

    if (
        model.billingMode === 'per-token' &&
        hasDerivedPricing &&
        !hasValue(model.inputPrice)
    ) {
        warnings.push(t('按量计费下需要先填写输入价格，才能保存其它价格项。'));
    }

    if (
        model.billingMode === 'per-token' &&
        hasValue(model.audioOutputPrice) &&
        !hasValue(model.audioInputPrice)
    ) {
        warnings.push(t('填写音频输出价格前，需要先填写音频输入价格。'));
    }

    if (model.billingMode === 'per-token' && model.videoBillingMode === 'per-token') {
        const hasInvalidTextRule = (model.videoTextToVideoRules || []).some(
            (row) =>
                (hasValue(row.resolution) &&
                    !VIDEO_RESOLUTION_REGEX.test(row.resolution)) ||
                (hasValue(row.tokenPrice) && !hasValue(row.pixelCompression)),
        );
        const hasInvalidVideoRule = (model.videoGenerateRules || []).some(
            (row) =>
                (hasValue(row.resolution) &&
                    !VIDEO_RESOLUTION_REGEX.test(row.resolution)) ||
                (hasValue(row.tokenPrice) && !hasValue(row.pixelCompression)),
        );
        const hasInvalidVideoUploadRule = (model.videoUploadRules || []).some(
            (row) =>
                (hasValue(row.resolution) &&
                    !VIDEO_RESOLUTION_REGEX.test(row.resolution)) ||
                (hasValue(row.tokenPrice) && !hasValue(row.pixelCompression)),
        );
        const hasInvalidImageRule = (model.videoImageToVideoRules || []).some(
            (row) =>
                (hasValue(row.resolution) &&
                    !VIDEO_RESOLUTION_REGEX.test(row.resolution)) ||
                (hasValue(row.tokenPrice) && !hasValue(row.pixelCompression)),
        );
        if (hasInvalidTextRule || hasInvalidVideoRule || hasInvalidVideoUploadRule || hasInvalidImageRule) {
            warnings.push(
                t(
                    '视频分辨率行中存在非法格式，请使用如 1280x720 的分辨率，并同时填写 token 价格与像素压缩系数。',
                ),
            );
        }
        if (
            hasDuplicateResolution(model.videoTextToVideoRules) ||
            hasDuplicateResolution(model.videoGenerateRules) ||
            hasDuplicateResolution(model.videoUploadRules) ||
            hasDuplicateResolution(model.videoImageToVideoRules)
        ) {
            warnings.push(t('同一规则组内分辨率不能重复，请删除重复项。'));
        }
    }

    if (model.billingMode === 'per-token' && model.videoBillingMode === 'per-video') {
        const hasInvalidPerVideoRow = (rows) =>
            (rows || []).some(
                (row) =>
                    (hasValue(row.resolution) &&
                        !VIDEO_RESOLUTION_REGEX.test(row.resolution)) ||
                    (hasValue(row.videoPrice) &&
                        toNumberOrNull(row.videoPrice) === null),
            );
        if (
            hasInvalidPerVideoRow(model.videoTextToVideoRules) ||
            hasInvalidPerVideoRow(model.videoImageToVideoRules) ||
            hasInvalidPerVideoRow(model.videoUploadRules) ||
            hasInvalidPerVideoRow(model.videoGenerateRules)
        ) {
            warnings.push(
                t(
                    '按视频计费的分辨率行请使用如 1280x720 的格式，并填写大于 0 的每条成片价格（与站点额度展示币种一致）。',
                ),
            );
        }
        if (
            hasDuplicateResolution(model.videoTextToVideoRules) ||
            hasDuplicateResolution(model.videoGenerateRules) ||
            hasDuplicateResolution(model.videoUploadRules) ||
            hasDuplicateResolution(model.videoImageToVideoRules)
        ) {
            warnings.push(t('同一规则组内分辨率不能重复，请删除重复项。'));
        }
    }

    return warnings;
};

export const buildSummaryText = (model, t) => {
    if (model.billingMode === 'per-request' && hasValue(model.fixedPrice)) {
        return `${t('按次')} $${model.fixedPrice} / ${t('次')}`;
    }

    if (hasValue(model.inputPrice)) {
        const inputLabel = `$${model.inputPrice}`;
        const outputLabel = hasValue(model.completionPrice)
            ? `$${model.completionPrice}`
            : '-';
        return `${t('输入')}：${inputLabel}｜${t('输出')}：${outputLabel}`;
    }

    return t('未设置价格');
};

export const buildOptionalFieldToggles = (model) => ({
    completionPrice:
        model.completionRatioLocked || hasValue(model.completionPrice),
    cachePrice: hasValue(model.cachePrice),
    createCachePrice: hasValue(model.createCachePrice),
    imagePrice: hasValue(model.imagePrice),
    audioInputPrice: hasValue(model.audioInputPrice),
    audioOutputPrice: hasValue(model.audioOutputPrice),
    video:
        (model.videoTextToVideoRules || []).length > 0 ||
        (model.videoImageToVideoRules || []).length > 0 ||
        (model.videoUploadRules || []).length > 0 ||
        (model.videoGenerateRules || []).length > 0 ||
        hasValue(model.videoFixedPrice) ||
        (model.videoTextToVideoRules || []).some((r) => hasValue(r?.videoPrice)) ||
        (model.videoImageToVideoRules || []).some((r) => hasValue(r?.videoPrice)) ||
        (model.videoUploadRules || []).some((r) => hasValue(r?.videoPrice)) ||
        (model.videoGenerateRules || []).some((r) => hasValue(r?.videoPrice)),
});

const normalizePerVideoPricingRows = (rows) =>
    (rows || [])
        .filter(
            (row) =>
                hasValue(row?.resolution) &&
                hasValue(row?.videoPrice) &&
                VIDEO_RESOLUTION_REGEX.test(row.resolution),
        )
        .map((row) => {
            const videoPrice = toNumberOrNull(row.videoPrice);
            if (videoPrice === null || videoPrice <= 0) {
                return null;
            }
            return {
                resolution: row.resolution.replace(/\s+/g, ''),
                video_price: videoPrice,
            };
        })
        .filter(Boolean)
        .filter(
            (row, index, arr) =>
                arr.findIndex((item) => item.resolution === row.resolution) ===
                index,
        );

const serializeModel = (model, t) => {
    const result = {
        ModelPrice: null,
        ModelRatio: null,
        CompletionRatio: null,
        CacheRatio: null,
        CreateCacheRatio: null,
        ImageRatio: null,
        AudioRatio: null,
        AudioCompletionRatio: null,
        VideoRatio: null,
        VideoCompletionRatio: null,
        VideoPrice: null,
        VideoPricingRules: null,
        RequestTierPricing: null,
    };

    if (model.billingMode === 'per-request') {
        if (hasValue(model.fixedPrice)) {
            result.ModelPrice = toNormalizedNumber(model.fixedPrice);
        }
        if (hasTierRule(model.requestTierRule)) {
            result.RequestTierPricing = serializeTierRule(model.requestTierRule);
        }
        return result;
    }

    if (hasTierRule(model.requestTierRule)) {
        const tierError = validateTierRule(model.requestTierRule, t);
        if (tierError) {
            throw new Error(`${model.name}: ${tierError}`);
        }
        result.RequestTierPricing = serializeTierRule(model.requestTierRule);
    }

    const inputPrice = toNumberOrNull(model.inputPrice);
    const completionPrice = toNumberOrNull(model.completionPrice);
    const cachePrice = toNumberOrNull(model.cachePrice);
    const createCachePrice = toNumberOrNull(model.createCachePrice);
    const imagePrice = toNumberOrNull(model.imagePrice);
    const audioInputPrice = toNumberOrNull(model.audioInputPrice);
    const audioOutputPrice = toNumberOrNull(model.audioOutputPrice);
    const videoFixedPrice = toNumberOrNull(model.videoFixedPrice);
    const videoPerToken = model.videoBillingMode === 'per-token';
    const videoPerVideo = model.videoBillingMode === 'per-video';

    const hasDependentPrice = [
        completionPrice,
        cachePrice,
        createCachePrice,
        imagePrice,
        audioInputPrice,
        audioOutputPrice,
        videoPerToken && (model.videoTextToVideoRules || []).length > 0 ? 1 : null,
        videoPerToken && (model.videoImageToVideoRules || []).length > 0 ? 1 : null,
        videoPerToken && (model.videoUploadRules || []).length > 0 ? 1 : null,
        videoPerToken && (model.videoGenerateRules || []).length > 0 ? 1 : null,
        videoPerVideo &&
            ((model.videoTextToVideoRules || []).some(
                (row) =>
                    hasValue(row?.resolution) && hasValue(row?.videoPrice),
            ) ||
                (model.videoImageToVideoRules || []).some(
                    (row) =>
                        hasValue(row?.resolution) && hasValue(row?.videoPrice),
                ) ||
                (model.videoUploadRules || []).some(
                    (row) =>
                        hasValue(row?.resolution) && hasValue(row?.videoPrice),
                ) ||
                (model.videoGenerateRules || []).some(
                    (row) =>
                        hasValue(row?.resolution) && hasValue(row?.videoPrice),
                ))
            ? 1
            : null,
    ].some((value) => value !== null);

    if (inputPrice === null) {
        if (hasDependentPrice) {
            throw new Error(
                t(
                    '模型 {{name}} 缺少输入价格，无法计算输出/缓存/图片/音频价格对应的倍率',
                    {
                        name: model.name,
                    },
                ),
            );
        }

        if (hasValue(model.rawRatios.modelRatio)) {
            result.ModelRatio = toNormalizedNumber(model.rawRatios.modelRatio);
        }
        if (hasValue(model.rawRatios.completionRatio)) {
            result.CompletionRatio = toNormalizedNumber(
                model.rawRatios.completionRatio,
            );
        }
        if (hasValue(model.rawRatios.cacheRatio)) {
            result.CacheRatio = toNormalizedNumber(model.rawRatios.cacheRatio);
        }
        if (hasValue(model.rawRatios.createCacheRatio)) {
            result.CreateCacheRatio = toNormalizedNumber(
                model.rawRatios.createCacheRatio,
            );
        }
        if (hasValue(model.rawRatios.imageRatio)) {
            result.ImageRatio = toNormalizedNumber(model.rawRatios.imageRatio);
        }
        if (hasValue(model.rawRatios.audioRatio)) {
            result.AudioRatio = toNormalizedNumber(model.rawRatios.audioRatio);
        }
        if (hasValue(model.rawRatios.audioCompletionRatio)) {
            result.AudioCompletionRatio = toNormalizedNumber(
                model.rawRatios.audioCompletionRatio,
            );
        }
        if (videoPerToken) {
            if (hasValue(model.rawRatios.videoRatio)) {
                result.VideoRatio = toNormalizedNumber(model.rawRatios.videoRatio);
            }
            if (hasValue(model.rawRatios.videoCompletionRatio)) {
                result.VideoCompletionRatio = toNormalizedNumber(
                    model.rawRatios.videoCompletionRatio,
                );
            }
            if (
                model.rawRatios.videoPricingRules &&
                typeof model.rawRatios.videoPricingRules === 'object'
            ) {
                result.VideoPricingRules = model.rawRatios.videoPricingRules;
            }
        } else if (videoPerVideo) {
            const textPV = normalizePerVideoPricingRows(
                model.videoTextToVideoRules,
            );
            const imagePV = normalizePerVideoPricingRows(
                model.videoImageToVideoRules,
            );
            const uploadPV = normalizePerVideoPricingRows(model.videoUploadRules);
            const genPV = normalizePerVideoPricingRows(model.videoGenerateRules);
            if (
                textPV.length > 0 ||
                imagePV.length > 0 ||
                uploadPV.length > 0 ||
                genPV.length > 0
            ) {
                const pricingRules = {};
                if (textPV.length > 0) {
                    pricingRules.text_to_video_per_video = textPV;
                }
                if (imagePV.length > 0) {
                    pricingRules.image_to_video_per_video = imagePV;
                }
                if (uploadPV.length > 0) {
                    pricingRules.video_to_video_input_per_video = uploadPV;
                }
                if (genPV.length > 0) {
                    pricingRules.video_to_video_output_per_video = genPV;
                }
                result.VideoPricingRules = pricingRules;
            } else if (videoFixedPrice !== null) {
                result.VideoPrice = toNormalizedNumber(videoFixedPrice);
            }
        }
        return result;
    }

    result.ModelRatio = toNormalizedNumber(inputPrice / 2);

    if (!model.completionRatioLocked && completionPrice !== null) {
        result.CompletionRatio = toNormalizedNumber(completionPrice / inputPrice);
    } else if (
        model.completionRatioLocked &&
        hasValue(model.rawRatios.completionRatio)
    ) {
        result.CompletionRatio = toNormalizedNumber(
            model.rawRatios.completionRatio,
        );
    }
    if (cachePrice !== null) {
        result.CacheRatio = toNormalizedNumber(cachePrice / inputPrice);
    }
    if (createCachePrice !== null) {
        result.CreateCacheRatio = toNormalizedNumber(createCachePrice / inputPrice);
    }
    if (imagePrice !== null) {
        result.ImageRatio = toNormalizedNumber(imagePrice / inputPrice);
    }
    if (audioInputPrice !== null) {
        result.AudioRatio = toNormalizedNumber(audioInputPrice / inputPrice);
    }
    if (audioOutputPrice !== null) {
        if (audioInputPrice === null || audioInputPrice === 0) {
            throw new Error(
                t('模型 {{name}} 缺少音频输入价格，无法计算音频输出倍率', {
                    name: model.name,
                }),
            );
        }
        result.AudioCompletionRatio = toNormalizedNumber(
            audioOutputPrice / audioInputPrice,
        );
    }

    if (videoPerToken) {
        const normalizeRows = (rows) =>
            (rows || [])
                .filter(
                    (row) =>
                        hasValue(row?.resolution) &&
                        hasValue(row?.tokenPrice) &&
                        VIDEO_RESOLUTION_REGEX.test(row.resolution),
                )
                .map((row) => {
                    const tokenPrice = toNumberOrNull(row.tokenPrice);
                    const pixelCompression = toNumberOrNull(row.pixelCompression);
                    if (tokenPrice === null || tokenPrice <= 0) {
                        return null;
                    }
                    return {
                        resolution: row.resolution.replace(/\s+/g, ''),
                        token_price: tokenPrice,
                        pixel_compression:
                            pixelCompression !== null && pixelCompression > 0
                                ? pixelCompression
                                : 384,
                    };
                })
                .filter(Boolean)
                .filter((row, index, arr) =>
                    arr.findIndex((item) => item.resolution === row.resolution) ===
                    index,
                );
        const textToVideo = normalizeRows(model.videoTextToVideoRules);
        const imageToVideoRules = normalizeRows(model.videoImageToVideoRules);
        const videoUploadRules = normalizeRows(model.videoUploadRules);
        const videoGenerateRules = normalizeRows(model.videoGenerateRules);
        const similarityThreshold = toNumberOrNull(model.videoSimilarityThreshold);
        if (
            textToVideo.length > 0 ||
            imageToVideoRules.length > 0 ||
            videoUploadRules.length > 0 ||
            videoGenerateRules.length > 0
        ) {
            const pricingRules = {};
            if (textToVideo.length > 0) {
                pricingRules.text_to_video = textToVideo;
            }
            if (imageToVideoRules.length > 0) {
                pricingRules.image_to_video_rules = imageToVideoRules;
            }
            if (videoUploadRules.length > 0) {
                pricingRules.video_to_video_input = videoUploadRules;
            }
            if (videoGenerateRules.length > 0) {
                pricingRules.video_to_video_output = videoGenerateRules;
            }
            if (similarityThreshold !== null && similarityThreshold > 0) {
                pricingRules.similarity_threshold = similarityThreshold;
            }
            result.VideoPricingRules = pricingRules;
        }
    } else if (videoPerVideo) {
        const textPV = normalizePerVideoPricingRows(model.videoTextToVideoRules);
        const imagePV = normalizePerVideoPricingRows(
            model.videoImageToVideoRules,
        );
        const uploadPV = normalizePerVideoPricingRows(model.videoUploadRules);
        const genPV = normalizePerVideoPricingRows(model.videoGenerateRules);
        if (
            textPV.length > 0 ||
            imagePV.length > 0 ||
            uploadPV.length > 0 ||
            genPV.length > 0
        ) {
            const pricingRules = {};
            if (textPV.length > 0) {
                pricingRules.text_to_video_per_video = textPV;
            }
            if (imagePV.length > 0) {
                pricingRules.image_to_video_per_video = imagePV;
            }
            if (uploadPV.length > 0) {
                pricingRules.video_to_video_input_per_video = uploadPV;
            }
            if (genPV.length > 0) {
                pricingRules.video_to_video_output_per_video = genPV;
            }
            result.VideoPricingRules = pricingRules;
        } else if (videoFixedPrice !== null) {
            result.VideoPrice = toNormalizedNumber(videoFixedPrice);
        }
    }

    return result;
};

export const buildPreviewRows = (model, t) => {
    if (!model) return [];

    if (model.billingMode === 'per-request') {
        return [
            {
                key: 'ModelPrice',
                label: 'ModelPrice',
                value: hasValue(model.fixedPrice) ? model.fixedPrice : t('空'),
            },
        ];
    }

    const inputPrice = toNumberOrNull(model.inputPrice);
    const videoPerToken = model.videoBillingMode === 'per-token';
    const videoPerVideo = model.videoBillingMode === 'per-video';
    const hasPerVideoPricingRuleRows =
        videoPerVideo &&
        ((model.videoTextToVideoRules || []).some(
            (r) => hasValue(r?.resolution) && hasValue(r?.videoPrice),
        ) ||
            (model.videoImageToVideoRules || []).some(
                (r) => hasValue(r?.resolution) && hasValue(r?.videoPrice),
            ) ||
            (model.videoUploadRules || []).some(
                (r) => hasValue(r?.resolution) && hasValue(r?.videoPrice),
            ) ||
            (model.videoGenerateRules || []).some(
                (r) => hasValue(r?.resolution) && hasValue(r?.videoPrice),
            ));
    if (inputPrice === null) {
        return [
            {
                key: 'ModelRatio',
                label: 'ModelRatio',
                value: hasValue(model.rawRatios.modelRatio)
                    ? model.rawRatios.modelRatio
                    : t('空'),
            },
            {
                key: 'CompletionRatio',
                label: 'CompletionRatio',
                value: hasValue(model.rawRatios.completionRatio)
                    ? model.rawRatios.completionRatio
                    : t('空'),
            },
            {
                key: 'CacheRatio',
                label: 'CacheRatio',
                value: hasValue(model.rawRatios.cacheRatio)
                    ? model.rawRatios.cacheRatio
                    : t('空'),
            },
            {
                key: 'CreateCacheRatio',
                label: 'CreateCacheRatio',
                value: hasValue(model.rawRatios.createCacheRatio)
                    ? model.rawRatios.createCacheRatio
                    : t('空'),
            },
            {
                key: 'ImageRatio',
                label: 'ImageRatio',
                value: hasValue(model.rawRatios.imageRatio)
                    ? model.rawRatios.imageRatio
                    : t('空'),
            },
            {
                key: 'AudioRatio',
                label: 'AudioRatio',
                value: hasValue(model.rawRatios.audioRatio)
                    ? model.rawRatios.audioRatio
                    : t('空'),
            },
            {
                key: 'AudioCompletionRatio',
                label: 'AudioCompletionRatio',
                value: hasValue(model.rawRatios.audioCompletionRatio)
                    ? model.rawRatios.audioCompletionRatio
                    : t('空'),
            },
            {
                key: 'VideoPricingRules',
                label: 'VideoPricingRules',
                value:
                    (videoPerToken &&
                        model.rawRatios.videoPricingRules &&
                        typeof model.rawRatios.videoPricingRules === 'object') ||
                    hasPerVideoPricingRuleRows
                        ? t('已配置')
                        : t('空'),
            },
            {
                key: 'VideoPrice',
                label: 'VideoPrice',
                value:
                    videoPerVideo &&
                        hasValue(model.videoFixedPrice) &&
                        !hasPerVideoPricingRuleRows
                        ? model.videoFixedPrice
                        : videoPerVideo && hasPerVideoPricingRuleRows
                            ? t('按分辨率见 VideoPricingRules')
                            : t('空'),
            },
        ];
    }

    const completionPrice = toNumberOrNull(model.completionPrice);
    const cachePrice = toNumberOrNull(model.cachePrice);
    const createCachePrice = toNumberOrNull(model.createCachePrice);
    const imagePrice = toNumberOrNull(model.imagePrice);
    const audioInputPrice = toNumberOrNull(model.audioInputPrice);
    const audioOutputPrice = toNumberOrNull(model.audioOutputPrice);
    const videoFixedPrice = toNumberOrNull(model.videoFixedPrice);

    return [
        {
            key: 'ModelRatio',
            label: 'ModelRatio',
            value: formatNumber(inputPrice / 2),
        },
        {
            key: 'CompletionRatio',
            label: 'CompletionRatio',
            value: model.completionRatioLocked
                ? `${model.lockedCompletionRatio || t('空')} (${t('后端固定')})`
                : completionPrice !== null
                    ? formatNumber(completionPrice / inputPrice)
                    : t('空'),
        },
        {
            key: 'CacheRatio',
            label: 'CacheRatio',
            value:
                cachePrice !== null ? formatNumber(cachePrice / inputPrice) : t('空'),
        },
        {
            key: 'CreateCacheRatio',
            label: 'CreateCacheRatio',
            value:
                createCachePrice !== null
                    ? formatNumber(createCachePrice / inputPrice)
                    : t('空'),
        },
        {
            key: 'ImageRatio',
            label: 'ImageRatio',
            value:
                imagePrice !== null ? formatNumber(imagePrice / inputPrice) : t('空'),
        },
        {
            key: 'AudioRatio',
            label: 'AudioRatio',
            value:
                audioInputPrice !== null
                    ? formatNumber(audioInputPrice / inputPrice)
                    : t('空'),
        },
        {
            key: 'AudioCompletionRatio',
            label: 'AudioCompletionRatio',
            value:
                audioOutputPrice !== null &&
                    audioInputPrice !== null &&
                    audioInputPrice !== 0
                    ? formatNumber(audioOutputPrice / audioInputPrice)
                    : t('空'),
        },
        {
            key: 'VideoPricingRules',
            label: 'VideoPricingRules',
            value:
                (videoPerToken &&
                    ((model.videoTextToVideoRules || []).length > 0 ||
                        (model.videoImageToVideoRules || []).length > 0 ||
                        (model.videoUploadRules || []).length > 0 ||
                        (model.videoGenerateRules || []).length > 0)) ||
                hasPerVideoPricingRuleRows
                    ? t('已配置')
                    : t('空'),
        },
        {
            key: 'VideoPrice',
            label: 'VideoPrice',
            value:
                videoPerVideo &&
                    videoFixedPrice !== null &&
                    !hasPerVideoPricingRuleRows
                    ? formatNumber(videoFixedPrice)
                    : videoPerVideo && hasPerVideoPricingRuleRows
                        ? t('按分辨率见 VideoPricingRules')
                        : t('空'),
        },
        {
            key: 'RequestTierPricing',
            label: 'RequestTierPricing',
            value: hasTierRule(model.requestTierRule)
                ? summarizeTierRule(model.requestTierRule, t)
                : t('空'),
        },
    ];
};

export function useModelPricingEditorState({
    options,
    refresh,
    t,
    candidateModelNames = EMPTY_CANDIDATE_MODEL_NAMES,
    strictCandidateModelNames = false,
    filterMode = 'all',
    optionKeys,
    onSaveOutput,
}) {
    const [models, setModels] = useState([]);
    const [initialVisibleModelNames, setInitialVisibleModelNames] = useState([]);
    const [selectedModelName, setSelectedModelName] = useState('');
    const [selectedModelNames, setSelectedModelNames] = useState([]);
    const [searchText, setSearchText] = useState('');
    const [currentPage, setCurrentPage] = useState(1);
    const [loading, setLoading] = useState(false);
    const [conflictOnly, setConflictOnly] = useState(false);
    const [optionalFieldToggles, setOptionalFieldToggles] = useState({});
    const resolvedOptionKeys = useMemo(
        () => ({
            ModelPrice: optionKeys?.ModelPrice || 'ModelPrice',
            ModelRatio: optionKeys?.ModelRatio || 'ModelRatio',
            CompletionRatio: optionKeys?.CompletionRatio || 'CompletionRatio',
            CompletionRatioMeta:
                optionKeys?.CompletionRatioMeta || 'CompletionRatioMeta',
            CacheRatio: optionKeys?.CacheRatio || 'CacheRatio',
            CreateCacheRatio: optionKeys?.CreateCacheRatio || 'CreateCacheRatio',
            ImageRatio: optionKeys?.ImageRatio || 'ImageRatio',
            AudioRatio: optionKeys?.AudioRatio || 'AudioRatio',
            AudioCompletionRatio:
                optionKeys?.AudioCompletionRatio || 'AudioCompletionRatio',
            VideoRatio: optionKeys?.VideoRatio || 'VideoRatio',
            VideoCompletionRatio:
                optionKeys?.VideoCompletionRatio || 'VideoCompletionRatio',
            VideoPrice: optionKeys?.VideoPrice || 'VideoPrice',
            VideoPricingRules:
                optionKeys?.VideoPricingRules || 'VideoPricingRules',
            RequestTierPricing:
                optionKeys?.RequestTierPricing || 'RequestTierPricing',
        }),
        [optionKeys],
    );

    useEffect(() => {
        const sourceMaps = {
            ModelPrice: parseOptionJSON(options[resolvedOptionKeys.ModelPrice]),
            ModelRatio: parseOptionJSON(options[resolvedOptionKeys.ModelRatio]),
            CompletionRatio: parseOptionJSON(
                options[resolvedOptionKeys.CompletionRatio],
            ),
            CompletionRatioMeta: parseOptionJSON(
                options[resolvedOptionKeys.CompletionRatioMeta],
            ),
            CacheRatio: parseOptionJSON(options[resolvedOptionKeys.CacheRatio]),
            CreateCacheRatio: parseOptionJSON(
                options[resolvedOptionKeys.CreateCacheRatio],
            ),
            ImageRatio: parseOptionJSON(options[resolvedOptionKeys.ImageRatio]),
            AudioRatio: parseOptionJSON(options[resolvedOptionKeys.AudioRatio]),
            AudioCompletionRatio: parseOptionJSON(
                options[resolvedOptionKeys.AudioCompletionRatio],
            ),
            VideoRatio: parseOptionJSON(options[resolvedOptionKeys.VideoRatio]),
            VideoCompletionRatio: parseOptionJSON(
                options[resolvedOptionKeys.VideoCompletionRatio],
            ),
            VideoPrice: parseOptionJSON(options[resolvedOptionKeys.VideoPrice]),
            VideoPricingRules: parseOptionJSON(
                options[resolvedOptionKeys.VideoPricingRules],
            ),
            RequestTierPricing: parseOptionJSON(
                options[resolvedOptionKeys.RequestTierPricing],
            ),
        };

        // strictCandidateModelNames=true 时，模型列表严格限制为外部传入候选模型（用于按渠道筛模型）。
        const names = strictCandidateModelNames
            ? new Set(candidateModelNames)
            : new Set([
                ...candidateModelNames,
                ...Object.keys(sourceMaps.ModelPrice),
                ...Object.keys(sourceMaps.ModelRatio),
                ...Object.keys(sourceMaps.CompletionRatio),
                ...Object.keys(sourceMaps.CompletionRatioMeta),
                ...Object.keys(sourceMaps.CacheRatio),
                ...Object.keys(sourceMaps.CreateCacheRatio),
                ...Object.keys(sourceMaps.ImageRatio),
                ...Object.keys(sourceMaps.AudioRatio),
                ...Object.keys(sourceMaps.AudioCompletionRatio),
                ...Object.keys(sourceMaps.VideoRatio),
                ...Object.keys(sourceMaps.VideoCompletionRatio),
                ...Object.keys(sourceMaps.VideoPrice),
                ...Object.keys(sourceMaps.VideoPricingRules),
                ...Object.keys(sourceMaps.RequestTierPricing),
            ]);

        const nextModels = Array.from(names)
            .map((name) => buildModelState(name, sourceMaps))
            .sort((a, b) => a.name.localeCompare(b.name));

        setModels(nextModels);
        setInitialVisibleModelNames(
            filterMode === 'unset'
                ? nextModels
                    .filter((model) => isBasePricingUnset(model))
                    .map((model) => model.name)
                : nextModels.map((model) => model.name),
        );
        setOptionalFieldToggles(
            nextModels.reduce((acc, model) => {
                acc[model.name] = buildOptionalFieldToggles(model);
                return acc;
            }, {}),
        );
        setSelectedModelName((previous) => {
            if (previous && nextModels.some((model) => model.name === previous)) {
                return previous;
            }
            const nextVisibleModels =
                filterMode === 'unset'
                    ? nextModels.filter((model) => isBasePricingUnset(model))
                    : nextModels;
            return nextVisibleModels[0]?.name || '';
        });
    }, [
        candidateModelNames,
        filterMode,
        options,
        resolvedOptionKeys,
        strictCandidateModelNames,
    ]);

    const visibleModels = useMemo(() => {
        return filterMode === 'unset'
            ? models.filter((model) => initialVisibleModelNames.includes(model.name))
            : models;
    }, [filterMode, initialVisibleModelNames, models]);

    const filteredModels = useMemo(() => {
        return visibleModels.filter((model) => {
            const keyword = searchText.trim().toLowerCase();
            const keywordMatch = keyword
                ? model.name.toLowerCase().includes(keyword)
                : true;
            const conflictMatch = conflictOnly ? model.hasConflict : true;
            return keywordMatch && conflictMatch;
        });
    }, [conflictOnly, searchText, visibleModels]);

    const pagedData = useMemo(() => {
        const start = (currentPage - 1) * PAGE_SIZE;
        return filteredModels.slice(start, start + PAGE_SIZE);
    }, [currentPage, filteredModels]);

    const selectedModel = useMemo(
        () =>
            visibleModels.find((model) => model.name === selectedModelName) || null,
        [selectedModelName, visibleModels],
    );

    const selectedWarnings = useMemo(
        () => getModelWarnings(selectedModel, t),
        [selectedModel, t],
    );

    const previewRows = useMemo(
        () => buildPreviewRows(selectedModel, t),
        [selectedModel, t],
    );

    useEffect(() => {
        setCurrentPage(1);
    }, [searchText, conflictOnly, filterMode, candidateModelNames]);

    useEffect(() => {
        setSelectedModelNames((previous) =>
            previous.filter((name) =>
                visibleModels.some((model) => model.name === name),
            ),
        );
    }, [visibleModels]);

    useEffect(() => {
        if (visibleModels.length === 0) {
            setSelectedModelName('');
            return;
        }
        if (!visibleModels.some((model) => model.name === selectedModelName)) {
            setSelectedModelName(visibleModels[0].name);
        }
    }, [selectedModelName, visibleModels]);

    const upsertModel = (name, updater) => {
        setModels((previous) =>
            previous.map((model) => {
                if (model.name !== name) return model;
                return typeof updater === 'function' ? updater(model) : updater;
            }),
        );
    };

    const isOptionalFieldEnabled = (model, field) => {
        if (!model) return false;
        const modelToggles = optionalFieldToggles[model.name];
        if (modelToggles && typeof modelToggles[field] === 'boolean') {
            return modelToggles[field];
        }
        return buildOptionalFieldToggles(model)[field];
    };

    const updateOptionalFieldToggle = (modelName, field, checked) => {
        setOptionalFieldToggles((prev) => ({
            ...prev,
            [modelName]: {
                ...(prev[modelName] || {}),
                [field]: checked,
            },
        }));
    };

    const handleOptionalFieldToggle = (field, checked) => {
        if (!selectedModel) return;

        updateOptionalFieldToggle(selectedModel.name, field, checked);

        if (checked) {
            return;
        }

        upsertModel(selectedModel.name, (model) => {
            const nextModel = { ...model, [field]: '' };

            if (field === 'audioInputPrice') {
                nextModel.audioOutputPrice = '';
                setOptionalFieldToggles((prev) => ({
                    ...prev,
                    [selectedModel.name]: {
                        ...(prev[selectedModel.name] || {}),
                        audioInputPrice: false,
                        audioOutputPrice: false,
                    },
                }));
            }

            if (field === 'video') {
                nextModel.videoFixedPrice = '';
                nextModel.videoTextToVideoRules = [];
                nextModel.videoImageToVideoRules = [];
                nextModel.videoUploadRules = [];
                nextModel.videoGenerateRules = [];
                nextModel.videoSimilarityThreshold = '';
            }

            return nextModel;
        });
    };

    const handleVideoBillingModeChange = (value) => {
        if (!selectedModel) return;
        upsertModel(selectedModel.name, (model) => ({
            ...model,
            videoBillingMode: value,
        }));
    };

    const updateVideoRuleRow = (section, index, key, value) => {
        if (!selectedModel) return;
        const numericKeys = new Set([
            'tokenPrice',
            'pixelCompression',
            'videoPrice',
        ]);
        if (key !== 'resolution' && !numericKeys.has(key)) return;
        if (key !== 'resolution' && !NUMERIC_INPUT_REGEX.test(value)) return;
        upsertModel(selectedModel.name, (model) => {
            const target = {
                text: 'videoTextToVideoRules',
                image: 'videoImageToVideoRules',
                videoUpload: 'videoUploadRules',
                videoGenerate: 'videoGenerateRules',
            }[section];
            if (!target) return model;
            const rows = [...(model[target] || [])];
            if (!rows[index]) return model;
            rows[index] = { ...rows[index], [key]: value };
            return { ...model, [target]: rows };
        });
    };

    const addVideoRuleRow = (section) => {
        if (!selectedModel) return;
        upsertModel(selectedModel.name, (model) => {
            const target = {
                text: 'videoTextToVideoRules',
                image: 'videoImageToVideoRules',
                videoUpload: 'videoUploadRules',
                videoGenerate: 'videoGenerateRules',
            }[section];
            if (!target) return model;
            const isPerVideo = model.videoBillingMode === 'per-video';
            const newRow = isPerVideo
                ? { resolution: '', videoPrice: '' }
                : {
                    resolution: '',
                    tokenPrice: '',
                    pixelCompression:
                        section === 'image'
                            ? DEFAULT_IMAGE_VIDEO_PIXEL_COMPRESSION
                            : section === 'text'
                            ? DEFAULT_TEXT_VIDEO_PIXEL_COMPRESSION
                            : DEFAULT_VIDEO_VIDEO_PIXEL_COMPRESSION,
                };
            return {
                ...model,
                [target]: [...(model[target] || []), newRow],
            };
        });
    };

    const removeVideoRuleRow = (section, index) => {
        if (!selectedModel) return;
        upsertModel(selectedModel.name, (model) => {
            const target = {
                text: 'videoTextToVideoRules',
                image: 'videoImageToVideoRules',
                videoUpload: 'videoUploadRules',
                videoGenerate: 'videoGenerateRules',
            }[section];
            if (!target) return model;
            return {
                ...model,
                [target]: (model[target] || []).filter((_, i) => i !== index),
            };
        });
    };

    const fillDerivedPricesFromBase = (model, nextInputPrice) => {
        const baseNumber = toNumberOrNull(nextInputPrice);
        if (baseNumber === null) {
            return model;
        }

        return {
            ...model,
            completionPrice:
                model.completionRatioLocked && hasValue(model.lockedCompletionRatio)
                    ? formatNumber(baseNumber * Number(model.lockedCompletionRatio))
                    : !hasValue(model.completionPrice) &&
                        hasValue(model.rawRatios.completionRatio)
                        ? formatNumber(baseNumber * Number(model.rawRatios.completionRatio))
                        : model.completionPrice,
            cachePrice:
                !hasValue(model.cachePrice) && hasValue(model.rawRatios.cacheRatio)
                    ? formatNumber(baseNumber * Number(model.rawRatios.cacheRatio))
                    : model.cachePrice,
            createCachePrice:
                !hasValue(model.createCachePrice) &&
                    hasValue(model.rawRatios.createCacheRatio)
                    ? formatNumber(baseNumber * Number(model.rawRatios.createCacheRatio))
                    : model.createCachePrice,
            imagePrice:
                !hasValue(model.imagePrice) && hasValue(model.rawRatios.imageRatio)
                    ? formatNumber(baseNumber * Number(model.rawRatios.imageRatio))
                    : model.imagePrice,
            audioInputPrice:
                !hasValue(model.audioInputPrice) && hasValue(model.rawRatios.audioRatio)
                    ? formatNumber(baseNumber * Number(model.rawRatios.audioRatio))
                    : model.audioInputPrice,
            audioOutputPrice:
                !hasValue(model.audioOutputPrice) &&
                    hasValue(model.rawRatios.audioRatio) &&
                    hasValue(model.rawRatios.audioCompletionRatio)
                    ? formatNumber(
                        baseNumber *
                        Number(model.rawRatios.audioRatio) *
                        Number(model.rawRatios.audioCompletionRatio),
                    )
                    : model.audioOutputPrice,
        };
    };

    const handleNumericFieldChange = (field, value) => {
        if (!selectedModel || !NUMERIC_INPUT_REGEX.test(value)) {
            return;
        }

        upsertModel(selectedModel.name, (model) => {
            const updatedModel = { ...model, [field]: value };

            if (field === 'inputPrice') {
                return fillDerivedPricesFromBase(updatedModel, value);
            }

            return updatedModel;
        });
    };

    const handleBillingModeChange = (value) => {
        if (!selectedModel) return;
        upsertModel(selectedModel.name, (model) => ({
            ...model,
            billingMode: value,
        }));
    };

    const addModel = (modelName) => {
        const trimmedName = modelName.trim();
        if (!trimmedName) {
            showError(t('请输入模型名称'));
            return false;
        }
        if (models.some((model) => model.name === trimmedName)) {
            showError(t('模型名称已存在'));
            return false;
        }

        const nextModel = {
            ...EMPTY_MODEL,
            name: trimmedName,
            rawRatios: { ...EMPTY_MODEL.rawRatios },
            requestTierRule: null,
        };

        setModels((previous) => [nextModel, ...previous]);
        setOptionalFieldToggles((prev) => ({
            ...prev,
            [trimmedName]: buildOptionalFieldToggles(nextModel),
        }));
        setSelectedModelName(trimmedName);
        setCurrentPage(1);
        return true;
    };

    const deleteModel = (name) => {
        const nextModels = models.filter((model) => model.name !== name);
        setModels(nextModels);
        setOptionalFieldToggles((prev) => {
            const next = { ...prev };
            delete next[name];
            return next;
        });
        setSelectedModelNames((previous) =>
            previous.filter((item) => item !== name),
        );
        if (selectedModelName === name) {
            setSelectedModelName(nextModels[0]?.name || '');
        }
    };

    const applySelectedModelPricing = () => {
        if (!selectedModel) {
            showError(t('请先选择一个作为模板的模型'));
            return false;
        }
        if (selectedModelNames.length === 0) {
            showError(t('请先勾选需要批量设置的模型'));
            return false;
        }

        const sourceToggles = optionalFieldToggles[selectedModel.name] || {};

        setModels((previous) =>
            previous.map((model) => {
                if (!selectedModelNames.includes(model.name)) {
                    return model;
                }

                const nextModel = {
                    ...model,
                    billingMode: selectedModel.billingMode,
                    fixedPrice: selectedModel.fixedPrice,
                    inputPrice: selectedModel.inputPrice,
                    completionPrice: selectedModel.completionPrice,
                    cachePrice: selectedModel.cachePrice,
                    createCachePrice: selectedModel.createCachePrice,
                    imagePrice: selectedModel.imagePrice,
                    audioInputPrice: selectedModel.audioInputPrice,
                    audioOutputPrice: selectedModel.audioOutputPrice,
                    videoBillingMode: selectedModel.videoBillingMode,
                    videoFixedPrice: selectedModel.videoFixedPrice,
                    videoTextToVideoRules: selectedModel.videoTextToVideoRules,
                    videoImageToVideoRules: selectedModel.videoImageToVideoRules,
                    videoUploadRules: selectedModel.videoUploadRules,
                    videoGenerateRules: selectedModel.videoGenerateRules,
                    videoSimilarityThreshold: selectedModel.videoSimilarityThreshold,
                    requestTierRule: selectedModel.requestTierRule
                        ? normalizeTierRule(selectedModel.requestTierRule)
                        : null,
                };

                if (
                    nextModel.billingMode === 'per-token' &&
                    nextModel.completionRatioLocked &&
                    hasValue(nextModel.inputPrice) &&
                    hasValue(nextModel.lockedCompletionRatio)
                ) {
                    nextModel.completionPrice = formatNumber(
                        Number(nextModel.inputPrice) *
                        Number(nextModel.lockedCompletionRatio),
                    );
                }

                return nextModel;
            }),
        );

        setOptionalFieldToggles((previous) => {
            const next = { ...previous };
            selectedModelNames.forEach((modelName) => {
                const targetModel = models.find((item) => item.name === modelName);
                next[modelName] = {
                    completionPrice: targetModel?.completionRatioLocked
                        ? true
                        : Boolean(sourceToggles.completionPrice),
                    cachePrice: Boolean(sourceToggles.cachePrice),
                    createCachePrice: Boolean(sourceToggles.createCachePrice),
                    imagePrice: Boolean(sourceToggles.imagePrice),
                    audioInputPrice: Boolean(sourceToggles.audioInputPrice),
                    audioOutputPrice:
                        Boolean(sourceToggles.audioInputPrice) &&
                        Boolean(sourceToggles.audioOutputPrice),
                    video: Boolean(sourceToggles.video),
                };
            });
            return next;
        });

        showSuccess(
            t('已将模型 {{name}} 的价格配置批量应用到 {{count}} 个模型', {
                name: selectedModel.name,
                count: selectedModelNames.length,
            }),
        );
        return true;
    };

    const handleSubmit = async () => {
        setLoading(true);
        try {
            const output = {
                ModelPrice: {},
                ModelRatio: {},
                CompletionRatio: {},
                CacheRatio: {},
                CreateCacheRatio: {},
                ImageRatio: {},
                AudioRatio: {},
                AudioCompletionRatio: {},
                VideoRatio: {},
                VideoCompletionRatio: {},
                VideoPrice: {},
                VideoPricingRules: {},
                RequestTierPricing: {},
            };

            for (const model of models) {
                const serialized = serializeModel(model, t);
                Object.entries(serialized).forEach(([key, value]) => {
                    if (value !== null) {
                        output[key][model.name] = value;
                    }
                });
            }

            if (onSaveOutput) {
                await onSaveOutput(output);
            } else {
                const requestQueue = Object.entries(output).map(([key, value]) =>
                    API.put('/api/option/', {
                        key: resolvedOptionKeys[key] || key,
                        value: JSON.stringify(value, null, 2),
                    }),
                );

                const results = await Promise.all(requestQueue);
                for (const res of results) {
                    if (!res?.data?.success) {
                        throw new Error(res?.data?.message || t('保存失败，请重试'));
                    }
                }
            }

            showSuccess(t('保存成功'));
            await refresh();
        } catch (error) {
            console.error('保存失败:', error);
            showError(error.message || t('保存失败，请重试'));
        } finally {
            setLoading(false);
        }
    };

    const updateRequestTierRule = (rule) => {
        if (!selectedModel) return;
        upsertModel(selectedModel.name, (model) => ({
            ...model,
            requestTierRule: rule ? normalizeTierRule(rule) : null,
        }));
    };

    const applyRequestTierTemplate = (template) => {
        if (!selectedModel || !template) return;
        upsertModel(selectedModel.name, (model) => ({
            ...model,
            requestTierRule: normalizeTierRule(template),
        }));
    };

    return {
        models,
        selectedModel,
        selectedModelName,
        selectedModelNames,
        setSelectedModelName,
        setSelectedModelNames,
        searchText,
        setSearchText,
        currentPage,
        setCurrentPage,
        loading,
        conflictOnly,
        setConflictOnly,
        filteredModels,
        pagedData,
        selectedWarnings,
        previewRows,
        isOptionalFieldEnabled,
        handleOptionalFieldToggle,
        handleNumericFieldChange,
        handleBillingModeChange,
        handleVideoBillingModeChange,
        updateVideoRuleRow,
        addVideoRuleRow,
        removeVideoRuleRow,
        updateRequestTierRule,
        applyRequestTierTemplate,
        handleSubmit,
        addModel,
        deleteModel,
        applySelectedModelPricing,
    };
}
