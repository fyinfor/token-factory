export const VIDEO_ENDPOINT_TYPES = new Set([
  'openai-video',
  'hidream-video',
  'tokenfactory-video',
  'videogenerator',
  'tencentcloud-vod-video',
  'ali-video',
]);

export const hasNumericValue = (value) =>
  value != null && value !== '' && Number.isFinite(Number(value));

export const isVideoPricingModel = (model) => {
  if (!model) return false;
  const endpointTypes = Array.isArray(model.supported_endpoint_types)
    ? model.supported_endpoint_types
    : [];
  return (
    endpointTypes.some((type) => VIDEO_ENDPOINT_TYPES.has(type)) ||
    hasNumericValue(model.video_ratio) ||
    hasNumericValue(model.video_completion_ratio) ||
    hasNumericValue(model.video_price) ||
    !!model.video_flat_clip_hint
  );
};
