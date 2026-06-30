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
