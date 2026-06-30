export const getChannelRouteModelName = (modelData, channel) => {
  const modelName = modelData?.model_name || '';
  if (channel?.route_slug) {
    return `${modelName}/${channel.route_slug}`;
  }
  return `${channel?.supplier_alias || ''}/${modelName}/${channel?.channel_no || ''}`;
};

export const getModelChannelRouteNames = (model) => {
  const channelList = Array.isArray(model?.channel_list)
    ? model.channel_list
    : [];
  if (channelList.length === 0) return [];
  const seen = new Set();
  const names = [];
  channelList.forEach((channel) => {
    const routeName = getChannelRouteModelName(model, channel);
    if (!routeName || seen.has(routeName)) return;
    seen.add(routeName);
    names.push(routeName);
  });
  return names;
};

export const getModelDisplayRouteName = (model) => {
  const routeNames = getModelChannelRouteNames(model);
  if (routeNames.length === 0) {
    return model?.model_name || '';
  }
  return routeNames[0];
};

export const modelMatchesSearchTerm = (model, searchValue) => {
  const term = String(searchValue || '').trim().toLowerCase();
  if (!term) return true;

  const fields = [
    model?.model_name,
    model?.description,
    model?.tags,
    model?.vendor_name,
  ];

  if (fields.some((field) => field && String(field).toLowerCase().includes(term))) {
    return true;
  }

  const channelList = Array.isArray(model?.channel_list)
    ? model.channel_list
    : [];
  return channelList.some((channel) => {
    const routeName = getChannelRouteModelName(model, channel);
    return (
      (routeName && routeName.toLowerCase().includes(term)) ||
      (channel?.route_slug &&
        String(channel.route_slug).toLowerCase().includes(term)) ||
      (channel?.supplier_alias &&
        String(channel.supplier_alias).toLowerCase().includes(term)) ||
      (channel?.channel_no != null &&
        String(channel.channel_no).toLowerCase().includes(term))
    );
  });
};

const appendRouteSelectOption = (options, seen, routeName, model, channel) => {
  if (!routeName || seen.has(routeName)) return;
  seen.add(routeName);
  const searchParts = [
    routeName,
    model?.model_name,
    model?.description,
    model?.tags,
    model?.vendor_name,
    channel?.route_slug,
    channel?.supplier_alias,
    channel?.channel_no,
  ];
  const searchText = searchParts
    .filter((value) => value != null && String(value).trim())
    .map((value) => String(value).toLowerCase())
    .join(' ');
  options.push({ label: routeName, value: routeName, searchText });
};

/** 从 pricing 模型列表生成带渠道路径的下拉选项（每个渠道一条） */
export const buildModelRouteSelectOptions = (models) => {
  const seen = new Set();
  const options = [];
  (Array.isArray(models) ? models : []).forEach((model) => {
    const channelList = Array.isArray(model?.channel_list)
      ? model.channel_list
      : [];
    if (channelList.length === 0) {
      appendRouteSelectOption(options, seen, model?.model_name, model, null);
      return;
    }
    channelList.forEach((channel) => {
      appendRouteSelectOption(
        options,
        seen,
        getChannelRouteModelName(model, channel),
        model,
        channel,
      );
    });
  });
  return options.sort((a, b) => String(a.label).localeCompare(String(b.label)));
};

/** Select 过滤：匹配渠道路径、模型名、route_slug、supplier_alias 等 */
export const modelRouteSelectFilter = (input, option) => {
  if (!input) return true;
  const keyword = input.trim().toLowerCase();
  const searchText = (
    option?.searchText ?? `${option?.label ?? ''} ${option?.value ?? ''}`
  )
    .toString()
    .toLowerCase();
  return searchText.includes(keyword);
};

/** 保留历史 plain model_name 选项，避免编辑旧幻灯片时下拉无匹配值 */
export const augmentModelRouteSelectOptions = (options, currentValue) => {
  const value = String(currentValue || '').trim();
  if (!value) return options;
  if (options.some((option) => option.value === value)) return options;
  return [{ label: value, value, searchText: value.toLowerCase() }, ...options];
};
