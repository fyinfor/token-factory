const MAX_RESPONSE_BYTES = 1024 * 1024;
const REQUEST_TIMEOUT_MS = 10000;

class ClientError extends Error {
  constructor(message, code = "QUERY_FAILED") {
    super(message);
    this.name = "ClientError";
    this.code = code;
  }
}

function normalizeBaseUrl(value) {
  const raw = String(value || "").trim();
  if (!raw) throw new ClientError("请输入站点地址", "INVALID_URL");
  if (/(^|\/)(\.{1,2}|%2e(?:%2e)?)(\/|$)/i.test(raw)) {
    throw new ClientError("站点地址包含非法路径", "INVALID_URL");
  }

  let url;
  try {
    url = new URL(raw);
  } catch {
    throw new ClientError("站点地址格式无效", "INVALID_URL");
  }

  if (!['http:', 'https:'].includes(url.protocol)) {
    throw new ClientError("站点地址仅支持 HTTP 或 HTTPS", "INVALID_URL");
  }
  if (!url.hostname || url.username || url.password || url.search || url.hash) {
    throw new ClientError("站点地址不能包含账号、查询参数或锚点", "INVALID_URL");
  }

  url.pathname = url.pathname.replace(/\/+$/, "") || "/";
  return url.toString().replace(/\/$/, "");
}

function buildEndpoint(baseUrl, pathname) {
  const normalized = normalizeBaseUrl(baseUrl);
  const endpoint = String(pathname || "").replace(/^\/+/, "");
  return `${normalized}/${endpoint}`;
}

function validateApiKey(value) {
  const key = String(value || "").trim();
  if (key.length < 8) throw new ClientError("请输入有效的 API Key", "INVALID_KEY");
  return key;
}

async function requestJson(url, apiKey = "") {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

  try {
    const headers = { accept: "application/json" };
    if (apiKey) headers.authorization = `Bearer ${apiKey}`;

    const response = await fetch(url, {
      method: "GET",
      headers,
      redirect: "error",
      signal: controller.signal
    });

    const declaredLength = Number(response.headers.get("content-length") || 0);
    if (declaredLength > MAX_RESPONSE_BYTES) {
      throw new ClientError("服务响应过大", "RESPONSE_TOO_LARGE");
    }

    const bytes = new Uint8Array(await response.arrayBuffer());
    if (bytes.byteLength > MAX_RESPONSE_BYTES) {
      throw new ClientError("服务响应过大", "RESPONSE_TOO_LARGE");
    }

    let payload;
    try {
      payload = JSON.parse(new TextDecoder().decode(bytes));
    } catch {
      throw new ClientError("服务返回了无效数据", "INVALID_RESPONSE");
    }

    if (!response.ok || !payload || payload.success !== true) {
      throw new ClientError(payload?.message || `查询失败 (${response.status})`);
    }
    return payload;
  } catch (error) {
    if (error instanceof ClientError) throw error;
    if (error?.name === "AbortError") {
      throw new ClientError("查询超时，请检查站点地址", "TIMEOUT");
    }
    throw new ClientError("无法连接查询服务", "NETWORK_ERROR");
  } finally {
    clearTimeout(timer);
  }
}

async function queryTokenFactory({ baseUrl, apiKey }) {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl);
  const normalizedApiKey = validateApiKey(apiKey);
  const [status, usage, logs] = await Promise.all([
    requestJson(buildEndpoint(normalizedBaseUrl, "/api/status")),
    requestJson(buildEndpoint(normalizedBaseUrl, "/api/usage/user/"), normalizedApiKey),
    requestJson(
      buildEndpoint(normalizedBaseUrl, "/api/log/user?p=1&page_size=10&type=2"),
      normalizedApiKey
    )
  ]);

  return { baseUrl: normalizedBaseUrl, status, usage, logs };
}

module.exports = {
  MAX_RESPONSE_BYTES,
  REQUEST_TIMEOUT_MS,
  ClientError,
  normalizeBaseUrl,
  buildEndpoint,
  validateApiKey,
  requestJson,
  queryTokenFactory
};
