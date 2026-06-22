import fs from 'fs';
import path from 'path';

const localesDir = path.join('src/i18n/locales');

const en = {
  BaseURL: 'BaseURL',
  'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于':
    'BaseURL is the base address for OpenAI-compatible clients, with the path fixed at /v1. Typically used for',
  'Query 参数说明': 'Query Parameters',
  'Responses 端点是 OpenAI 新版统一响应接口，适合需要文本生成、多模态输入、工具调用或更统一响应结构的客户端和自建应用。':
    'The Responses endpoint is OpenAI’s unified response API, suitable for clients and custom apps that need text generation, multimodal input, tool calling, or a more unified response structure.',
  'WorkBuddy 快捷接入': 'WorkBuddy Quick Setup',
  '{{count}}选1复制使用': 'Copy one of {{count}} options',
  '不同模型支持的参数可能不同，例如 prompt、image、size、quality、response_format 等，请以模型文档或调用示例为准。':
    'Supported parameters vary by model (e.g. prompt, image, size, quality, response_format). Refer to the model docs or call examples.',
  价格项: 'Price item',
  '向量端点用于把文本转换为 embedding，常用于知识库检索、RAG、相似度搜索、聚类、推荐和语义匹配。':
    'The embeddings endpoint converts text to embeddings, commonly used for knowledge-base retrieval, RAG, similarity search, clustering, recommendations, and semantic matching.',
  '图像端点通常用于文生图、图生图、图片编辑或图片分析类能力，适合绘图工具、自动化工作流和自建视觉应用接入。':
    'Image endpoints are typically used for text-to-image, image-to-image, editing, or analysis—suitable for drawing tools, automation workflows, and custom vision apps.',
  基础地址: 'Base URL',
  '复制API Key': 'Copy API Key',
  '复制API端点': 'Copy API endpoint',
  '复制BaseURL': 'Copy BaseURL',
  '复制可用于调用上述 API 端点的 API Key':
    'Copy an API Key that can be used to call the API endpoints above',
  '复制带渠道路由的模型名，可将请求固定到指定渠道':
    'Copy the channel-routed model name to pin requests to a specific channel',
  复制模型名字: 'Copy model name',
  '多数只要求填写 BaseURL 的工具会自动补全 /v1/chat/completions；如果工具要求填写完整接口地址，就复制这里显示的完整 URL。':
    'Tools that only require BaseURL usually auto-append /v1/chat/completions; if the tool needs a full endpoint URL, copy the complete URL shown here.',
  '如果你的工具明确支持 Responses API，可以优先选择这个端点；老版 OpenAI 兼容工具通常仍使用聊天补全端点。':
    'If your tool explicitly supports the Responses API, prefer this endpoint; older OpenAI-compatible tools usually still use the chat completions endpoint.',
  '如果你的工具是知识库或工作流工具，请在 embedding 模型配置里填写这个完整端点地址和对应 API Key。':
    'For knowledge-base or workflow tools, enter this full endpoint URL and the corresponding API Key in the embedding model settings.',
  '如果客户端只支持 OpenAI 兼容图像接口，通常需要填写完整端点地址，并确认模型名称与通道路由模型名一致。':
    'If the client only supports OpenAI-compatible image APIs, enter the full endpoint URL and ensure the model name matches the channel-routed model name.',
  '如果工具只提供 BaseURL 输入框，通常填写 BaseURL 即可；如果工具提供接口地址或 Endpoint 输入框，请复制这里的完整 URL。':
    'If the tool only has a BaseURL field, BaseURL is usually enough; if it has an endpoint/Endpoint field, copy the full URL shown here.',
  官方价: 'Official price',
  展开全部: 'Show all',
  已复制BaseURL: 'BaseURL copied',
  已复制API端点: 'API endpoint copied',
  '已复制API Key': 'API Key copied',
  平台价: 'Platform price',
  张: 'image',
  '按工具要求复制 BaseURL 或完整 API 端点即可。':
    'Copy BaseURL or the full API endpoint as required by your tool.',
  '按渠道展示稳定性、路由和当前价格':
    'Shows stability, routing, and current pricing per channel',
  '接入前建议确认客户端是否支持对应音频接口格式；如果只支持普通聊天接口，请选择聊天补全端点。':
    'Confirm the client supports the audio API format; if it only supports chat, use the chat completions endpoint.',
  '接入时通常传入 input 和 model，返回的向量会写入向量数据库或检索系统，例如 Milvus、Qdrant、pgvector、Elasticsearch 等。':
    'Typically pass input and model; returned vectors are stored in vector DBs such as Milvus, Qdrant, pgvector, or Elasticsearch.',
  接口地址: 'API endpoint',
  暂无端点路径: 'No endpoint path',
  '更适合 Hermes、自建脚本、自建前端/后端、HTTP 工作流节点、Postman、Apifox、curl，或明确支持 OpenAI Videos API / Sora 视频端点的工具。':
    'Best suited for Hermes, custom scripts, front/back ends, HTTP workflow nodes, Postman, Apifox, curl, or tools that support OpenAI Videos API / Sora video endpoints.',
  '查看API Key': 'View API Key',
  渠道信息与价格: 'Channel info & pricing',
  '用于需要填写完整接口地址的工具和应用，例如':
    'For tools and apps that require a full endpoint URL, such as',
  用途说明: 'Usage notes',
  稳定性: 'Stability',
  端点说明: 'Endpoint notes',
  第一步: 'Step 1',
  第二步: 'Step 2',
  第三步: 'Step 3',
  '等工具、HTTP 工作流工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用；若工具仅支持聊天补全，请优先选择 /v1/chat/completions。':
    ' and similar tools, HTTP workflow tools, or custom services. Use with the model name from Step 2 and the API Key from Step 3; if the tool only supports chat completions, prefer /v1/chat/completions.',
  '等工具、工作流工具或自建应用的服务地址配置。':
    ' and similar tools, workflow tools, or custom apps for service address configuration.',
  '等工具、工作流编排工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用。':
    ' and similar tools, workflow orchestration tools, or custom services. Use with the model name from Step 2 and the API Key from Step 3.',
  '等工具，也适合支持自定义 HTTP 请求的工作流工具或自建应用。':
    ' and similar tools, as well as workflow tools or custom apps that support custom HTTP requests.',
  聊天补全端点适合: 'Chat completions endpoint is suitable for',
  '聊天补全端点，OpenAI 兼容生态中最常用的接口，通常用于 OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、工作流编排工具和自建聊天应用。':
    'Chat completions is the most common OpenAI-compatible endpoint, typically used with OpenAI SDK, Cherry Studio, Chatbox, LobeChat, Dify, workflow tools, and custom chat apps.',
  '视频端点用于创建视频生成任务，常见流程是提交 prompt、model、尺寸或时长等参数后轮询任务结果。适合':
    'Video endpoints create video generation tasks—submit prompt, model, size/duration, then poll for results. Suitable for',
  '视频端点通常用于视频生成、视频理解、分镜或任务型视频处理，适合创作工具、自动化生产流程和异步任务场景。':
    'Video endpoints are used for generation, understanding, storyboarding, or task-based video processing—suitable for creative tools, automated pipelines, and async tasks.',
  '视频能力往往不是一次请求立即返回最终结果，通常需要先创建视频任务，再轮询任务状态，最后下载或读取生成的视频内容。':
    'Video APIs often require creating a task, polling status, then downloading or reading the generated video rather than returning results in one request.',
  '该端点表示模型支持的一类具体 API 能力，完整请求地址由 BaseURL 加端点路径组成，适合需要手动填写接口地址的客户端、工作流工具或自建应用。':
    'This endpoint represents a specific API capability; the full URL is BaseURL plus the path—suitable for clients, workflow tools, or custom apps that need a manual endpoint URL.',
  '说明：如果 WorkBuddy 添加自定义模型时选择了「高级配置」中的「自定义协议」，开启后将直接使用填写的接口地址，不再自动补全 /chat/completions 路径；这种情况下接口地址填写到 /v1 即可，例如：{{url}}。':
    'Note: If WorkBuddy custom model setup uses Advanced → Custom Protocol, the entered endpoint is used as-is without auto-appending /chat/completions; in that case use /v1 only, e.g. {{url}}.',
  '请求一般使用 POST，并在请求体里传入 model、messages、temperature、stream 等参数；支持流式输出时可用于打字机效果、长文本持续返回和实时助手场景。':
    'Requests typically use POST with model, messages, temperature, stream, etc. in the body; streaming enables typewriter effects, long responses, and real-time assistants.',
  请求方法: 'Request method',
  '请求通常使用 POST，模型名称仍需填写通道路由模型名，API Key 使用本侧栏第三步复制的密钥。':
    'Requests typically use POST; use the channel-routed model name and the API Key copied in Step 3 of this panel.',
  '调用时通常需要同时配置 API Key 和模型名称；模型名称建议使用第二步中的通道路由模型名，以便请求固定到指定渠道。':
    'Configure both API Key and model name; use the channel-routed model name from Step 2 to pin requests to a channel.',
  '这类接口常需要上传音频文件或指定 voice、input、format 等参数，部分工具会要求使用 multipart/form-data。':
    'These APIs often require uploading audio or specifying voice, input, format, etc.; some tools require multipart/form-data.',
  选择通道路由模型名: 'Select channel-routed model name',
  '音频端点通常用于语音转文字、文字转语音、翻译或音频理解，适合客服质检、会议纪要、语音助手和媒体处理工作流。':
    'Audio endpoints are used for speech-to-text, text-to-speech, translation, or audio understanding—suitable for QA, meeting notes, voice assistants, and media workflows.',
  '输入 TOKEN 区间': 'Input token range',
  类型: 'Type',
  '模型{{modelName}}复制成功': 'Model {{modelName}} copied',
  '建议直连本站点发起请求，关闭系统代理或 VPN，避免代理干扰模型连接与请求转发。':
    'Connect directly to this site; disable system proxy or VPN to avoid interfering with model connections and request forwarding.',
  '要调用的模型名称。': 'Name of the model to call.',
  '组成对话的消息列表。每条消息通常包含 role 和 content 字段。':
    'List of messages in the conversation; each message usually has role and content.',
  '单条对话消息。': 'A single conversation message.',
  '消息角色，例如 system、user 或 assistant。':
    'Message role, e.g. system, user, or assistant.',
  '消息内容。': 'Message content.',
  '是否使用流式响应。': 'Whether to use streaming responses.',
  '响应 ID。': 'Response ID.',
  '对象类型。': 'Object type.',
  '响应创建时间戳。': 'Response creation timestamp.',
  '本次响应使用的模型名称。': 'Model name used in this response.',
  '模型生成结果列表。': 'List of model generation results.',
  '单个生成结果。': 'A single generation result.',
  '结果索引。': 'Result index.',
  '模型返回的消息。': 'Message returned by the model.',
  '消息角色。': 'Message role.',
  '生成结束原因。': 'Reason generation finished.',
  'Token 使用量统计。': 'Token usage statistics.',
  '输入 Token 数。': 'Input token count.',
  '输出 Token 数。': 'Output token count.',
  '总 Token 数。': 'Total token count.',
};

const ja = {
  BaseURL: 'BaseURL',
  'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于':
    'BaseURL は OpenAI 互換クライアントのベースアドレスで、パスは /v1 固定です。通常は',
  'Query 参数说明': 'Query パラメータ',
  'WorkBuddy 快捷接入': 'WorkBuddy クイック接続',
  '{{count}}选1复制使用': '{{count}} 件から 1 つをコピー',
  价格项: '価格項目',
  基础地址: 'ベース URL',
  '复制API Key': 'API Key をコピー',
  '复制API端点': 'API エンドポイントをコピー',
  '复制BaseURL': 'BaseURL をコピー',
  复制模型名字: 'モデル名をコピー',
  官方价: '公式価格',
  展开全部: 'すべて表示',
  已复制BaseURL: 'BaseURL をコピーしました',
  '已复制API Key': 'API Key をコピーしました',
  平台价: 'プラットフォーム価格',
  张: '枚',
  接口地址: 'API エンドポイント',
  暂无端点路径: 'エンドポイントパスなし',
  '查看API Key': 'API Key を表示',
  渠道信息与价格: 'チャネル情報と価格',
  用途说明: '用途説明',
  稳定性: '安定性',
  端点说明: 'エンドポイント説明',
  第一步: 'ステップ 1',
  第二步: 'ステップ 2',
  第三步: 'ステップ 3',
  请求方法: 'リクエストメソッド',
  选择通道路由模型名: 'チャネルルーティングモデル名を選択',
  '输入 TOKEN 区间': '入力 TOKEN 範囲',
  类型: '種類',
  '模型{{modelName}}复制成功': 'モデル {{modelName}} をコピーしました',
};

const fr = {
  BaseURL: 'BaseURL',
  'Query 参数说明': 'Paramètres Query',
  'WorkBuddy 快捷接入': 'Configuration rapide WorkBuddy',
  '{{count}}选1复制使用': 'Copier une option sur {{count}}',
  价格项: 'Élément de prix',
  基础地址: 'URL de base',
  '复制API Key': 'Copier la clé API',
  '复制API端点': "Copier le point d'accès API",
  '复制BaseURL': 'Copier BaseURL',
  复制模型名字: 'Copier le nom du modèle',
  官方价: 'Prix officiel',
  展开全部: 'Tout afficher',
  已复制BaseURL: 'BaseURL copié',
  '已复制API Key': 'Clé API copiée',
  平台价: 'Prix plateforme',
  张: 'image',
  接口地址: "Adresse de l'API",
  暂无端点路径: "Aucun chemin d'accès",
  '查看API Key': 'Voir la clé API',
  渠道信息与价格: 'Infos canal et tarifs',
  用途说明: "Notes d'utilisation",
  稳定性: 'Stabilité',
  端点说明: "Notes sur le point d'accès",
  第一步: 'Étape 1',
  第二步: 'Étape 2',
  第三步: 'Étape 3',
  请求方法: 'Méthode de requête',
  选择通道路由模型名: 'Sélectionner le nom de modèle routé',
  '输入 TOKEN 区间': 'Plage de tokens en entrée',
  类型: 'Type',
  '模型{{modelName}}复制成功': 'Modèle {{modelName}} copié',
};

const ru = {
  BaseURL: 'BaseURL',
  'Query 参数说明': 'Query-параметры',
  'WorkBuddy 快捷接入': 'Быстрая настройка WorkBuddy',
  '{{count}}选1复制使用': 'Скопировать один из {{count}}',
  价格项: 'Позиция цены',
  基础地址: 'Базовый URL',
  '复制API Key': 'Копировать API Key',
  '复制API端点': 'Копировать API endpoint',
  '复制BaseURL': 'Копировать BaseURL',
  复制模型名字: 'Копировать имя модели',
  官方价: 'Официальная цена',
  展开全部: 'Показать все',
  已复制BaseURL: 'BaseURL скопирован',
  '已复制API Key': 'API Key скопирован',
  平台价: 'Цена платформы',
  张: 'изображение',
  接口地址: 'Адрес API',
  暂无端点路径: 'Нет пути endpoint',
  '查看API Key': 'Просмотр API Key',
  渠道信息与价格: 'Информация о канале и цены',
  用途说明: 'Описание использования',
  稳定性: 'Стабильность',
  端点说明: 'Описание endpoint',
  第一步: 'Шаг 1',
  第二步: 'Шаг 2',
  第三步: 'Шаг 3',
  请求方法: 'Метод запроса',
  选择通道路由模型名: 'Выберите имя модели с маршрутизацией',
  '输入 TOKEN 区间': 'Диапазон входных TOKEN',
  类型: 'Тип',
  '模型{{modelName}}复制成功': 'Модель {{modelName}} скопирована',
};

const vi = {
  BaseURL: 'BaseURL',
  'Query 参数说明': 'Tham số Query',
  'WorkBuddy 快捷接入': 'Thiết lập nhanh WorkBuddy',
  '{{count}}选1复制使用': 'Sao chép 1 trong {{count}}',
  价格项: 'Mục giá',
  基础地址: 'URL cơ sở',
  '复制API Key': 'Sao chép API Key',
  '复制API端点': 'Sao chép endpoint API',
  '复制BaseURL': 'Sao chép BaseURL',
  复制模型名字: 'Sao chép tên mô hình',
  官方价: 'Giá chính thức',
  展开全部: 'Xem tất cả',
  已复制BaseURL: 'Đã sao chép BaseURL',
  '已复制API Key': 'Đã sao chép API Key',
  平台价: 'Giá nền tảng',
  张: 'ảnh',
  接口地址: 'Địa chỉ API',
  暂无端点路径: 'Không có đường dẫn endpoint',
  '查看API Key': 'Xem API Key',
  渠道信息与价格: 'Thông tin kênh và giá',
  用途说明: 'Hướng dẫn sử dụng',
  稳定性: 'Độ ổn định',
  端点说明: 'Mô tả endpoint',
  第一步: 'Bước 1',
  第二步: 'Bước 2',
  第三步: 'Bước 3',
  请求方法: 'Phương thức yêu cầu',
  选择通道路由模型名: 'Chọn tên mô hình định tuyến kênh',
  '输入 TOKEN 区间': 'Khoảng TOKEN đầu vào',
  类型: 'Loại',
  '模型{{modelName}}复制成功': 'Đã sao chép mô hình {{modelName}}',
};

const id = {
  BaseURL: 'BaseURL',
  'Query 参数说明': 'Parameter Query',
  'WorkBuddy 快捷接入': 'Pengaturan cepat WorkBuddy',
  '{{count}}选1复制使用': 'Salin 1 dari {{count}}',
  价格项: 'Item harga',
  基础地址: 'URL dasar',
  '复制API Key': 'Salin API Key',
  '复制API端点': 'Salin endpoint API',
  '复制BaseURL': 'Salin BaseURL',
  复制模型名字: 'Salin nama model',
  官方价: 'Harga resmi',
  展开全部: 'Tampilkan semua',
  已复制BaseURL: 'BaseURL disalin',
  '已复制API Key': 'API Key disalin',
  平台价: 'Harga platform',
  张: 'gambar',
  接口地址: 'Alamat API',
  暂无端点路径: 'Tidak ada path endpoint',
  '查看API Key': 'Lihat API Key',
  渠道信息与价格: 'Info saluran & harga',
  用途说明: 'Catatan penggunaan',
  稳定性: 'Stabilitas',
  端点说明: 'Catatan endpoint',
  第一步: 'Langkah 1',
  第二步: 'Langkah 2',
  第三步: 'Langkah 3',
  请求方法: 'Metode permintaan',
  选择通道路由模型名: 'Pilih nama model routing saluran',
  '输入 TOKEN 区间': 'Rentang TOKEN input',
  类型: 'Tipe',
  '模型{{modelName}}复制成功': 'Model {{modelName}} disalin',
};

const ms = { ...id };
const th = { ...id };
const sw = { ...id };

const localeMaps = {
  en,
  ja: { ...en, ...ja },
  fr: { ...en, ...fr },
  ru: { ...en, ...ru },
  vi: { ...en, ...vi },
  id: { ...en, ...id },
  ms: { ...en, ...ms },
  th: { ...en, ...th },
  sw: { ...en, ...sw },
  'zh-CN': Object.fromEntries(Object.keys(en).map((k) => [k, k])),
  'zh-TW': Object.fromEntries(
    Object.keys(en).map((k) => [
      k,
      k
        .replace(/复制/g, '複製')
        .replace(/说明/g, '說明')
        .replace(/视频/g, '影片')
        .replace(/图像/g, '圖像')
        .replace(/图片/g, '圖片')
        .replace(/输入/g, '輸入')
        .replace(/输出/g, '輸出')
        .replace(/加载/g, '載入')
        .replace(/展开/g, '展開')
        .replace(/选择/g, '選擇')
        .replace(/接口/g, '介面')
        .replace(/应用/g, '應用')
        .replace(/密钥/g, '密鑰')
        .replace(/侧栏/g, '側欄')
        .replace(/请求/g, '請求')
        .replace(/响应/g, '回應')
        .replace(/统计/g, '統計')
        .replace(/组成/g, '組成')
        .replace(/消息/g, '訊息')
        .replace(/内容/g, '內容')
        .replace(/是否/g, '是否')
        .replace(/对象/g, '物件')
        .replace(/时间戳/g, '時間戳')
        .replace(/生成/g, '產生')
        .replace(/结果/g, '結果')
        .replace(/结束/g, '結束')
        .replace(/总/g, '總')
        .replace(/区间/g, '區間')
        .replace(/类型/g, '類型')
        .replace(/直连/g, '直連')
        .replace(/关闭/g, '關閉')
        .replace(/干扰/g, '干擾')
        .replace(/连接/g, '連線')
        .replace(/转发/g, '轉發'),
    ]),
  ),
};

const files = fs.readdirSync(localesDir).filter((f) => f.endsWith('.json'));

for (const file of files) {
  const lang = file.replace('.json', '');
  const filePath = path.join(localesDir, file);
  const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  const map = localeMaps[lang] || en;
  let added = 0;
  for (const [key, value] of Object.entries(map)) {
    if (!data.translation[key]) {
      data.translation[key] = value;
      added++;
    }
  }
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + '\n');
  console.log(`${file}: added ${added} keys`);
}
