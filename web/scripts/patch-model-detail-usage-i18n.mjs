import fs from 'fs';
import path from 'path';

const localesDir = path.join('src/i18n/locales');

/** 用途说明 / 端点说明相关 key（覆盖已有英文回退） */
const usageKeys = [
  'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于',
  'Responses 端点是 OpenAI 新版统一响应接口，适合需要文本生成、多模态输入、工具调用或更统一响应结构的客户端和自建应用。',
  '不同模型支持的参数可能不同，例如 prompt、image、size、quality、response_format 等，请以模型文档或调用示例为准。',
  '向量端点用于把文本转换为 embedding，常用于知识库检索、RAG、相似度搜索、聚类、推荐和语义匹配。',
  '图像端点通常用于文生图、图生图、图片编辑或图片分析类能力，适合绘图工具、自动化工作流和自建视觉应用接入。',
  '复制可用于调用上述 API 端点的 API Key',
  '复制带渠道路由的模型名，可将请求固定到指定渠道',
  '多数只要求填写 BaseURL 的工具会自动补全 /v1/chat/completions；如果工具要求填写完整接口地址，就复制这里显示的完整 URL。',
  '如果你的工具明确支持 Responses API，可以优先选择这个端点；老版 OpenAI 兼容工具通常仍使用聊天补全端点。',
  '如果你的工具是知识库或工作流工具，请在 embedding 模型配置里填写这个完整端点地址和对应 API Key。',
  '如果客户端只支持 OpenAI 兼容图像接口，通常需要填写完整端点地址，并确认模型名称与通道路由模型名一致。',
  '如果工具只提供 BaseURL 输入框，通常填写 BaseURL 即可；如果工具提供接口地址或 Endpoint 输入框，请复制这里的完整 URL。',
  '按工具要求复制 BaseURL 或完整 API 端点即可。',
  '按渠道展示稳定性、路由和当前价格',
  '接入前建议确认客户端是否支持对应音频接口格式；如果只支持普通聊天接口，请选择聊天补全端点。',
  '接入时通常传入 input 和 model，返回的向量会写入向量数据库或检索系统，例如 Milvus、Qdrant、pgvector、Elasticsearch 等。',
  '更适合 Hermes、自建脚本、自建前端/后端、HTTP 工作流节点、Postman、Apifox、curl，或明确支持 OpenAI Videos API / Sora 视频端点的工具。',
  '用于需要填写完整接口地址的工具和应用，例如',
  '等工具、HTTP 工作流工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用；若工具仅支持聊天补全，请优先选择 /v1/chat/completions。',
  '等工具、工作流工具或自建应用的服务地址配置。',
  '等工具、工作流编排工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用。',
  '等工具，也适合支持自定义 HTTP 请求的工作流工具或自建应用。',
  '聊天补全端点适合',
  '聊天补全端点，OpenAI 兼容生态中最常用的接口，通常用于 OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、工作流编排工具和自建聊天应用。',
  '视频端点用于创建视频生成任务，常见流程是提交 prompt、model、尺寸或时长等参数后轮询任务结果。适合',
  '视频端点通常用于视频生成、视频理解、分镜或任务型视频处理，适合创作工具、自动化生产流程和异步任务场景。',
  '视频能力往往不是一次请求立即返回最终结果，通常需要先创建视频任务，再轮询任务状态，最后下载或读取生成的视频内容。',
  '该端点表示模型支持的一类具体 API 能力，完整请求地址由 BaseURL 加端点路径组成，适合需要手动填写接口地址的客户端、工作流工具或自建应用。',
  '说明：如果 WorkBuddy 添加自定义模型时选择了「高级配置」中的「自定义协议」，开启后将直接使用填写的接口地址，不再自动补全 /chat/completions 路径；这种情况下接口地址填写到 /v1 即可，例如：{{url}}。',
  '请求一般使用 POST，并在请求体里传入 model、messages、temperature、stream 等参数；支持流式输出时可用于打字机效果、长文本持续返回和实时助手场景。',
  '请求通常使用 POST，模型名称仍需填写通道路由模型名，API Key 使用本侧栏第三步复制的密钥。',
  '调用时通常需要同时配置 API Key 和模型名称；模型名称建议使用第二步中的通道路由模型名，以便请求固定到指定渠道。',
  '这类接口常需要上传音频文件或指定 voice、input、format 等参数，部分工具会要求使用 multipart/form-data。',
  '音频端点通常用于语音转文字、文字转语音、翻译或音频理解，适合客服质检、会议纪要、语音助手和媒体处理工作流。',
  '建议直连本站点发起请求，关闭系统代理或 VPN，避免代理干扰模型连接与请求转发。',
  '要调用的模型名称。',
  '组成对话的消息列表。每条消息通常包含 role 和 content 字段。',
  '单条对话消息。',
  '消息角色，例如 system、user 或 assistant。',
  '消息内容。',
  '是否使用流式响应。',
  '响应 ID。',
  '对象类型。',
  '响应创建时间戳。',
  '本次响应使用的模型名称。',
  '模型生成结果列表。',
  '单个生成结果。',
  '结果索引。',
  '模型返回的消息。',
  '消息角色。',
  '生成结束原因。',
  'Token 使用量统计。',
  '输入 Token 数。',
  '输出 Token 数。',
  '总 Token 数。',
];

const ja = {
  'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于':
    'BaseURL は OpenAI 互換クライアントのベースアドレスで、パスは /v1 固定です。通常は',
  'Responses 端点是 OpenAI 新版统一响应接口，适合需要文本生成、多模态输入、工具调用或更统一响应结构的客户端和自建应用。':
    'Responses エンドポイントは OpenAI の統合レスポンス API で、テキスト生成、マルチモーダル入力、ツール呼び出し、統一レスポンス構造が必要なクライアントや自作アプリ向けです。',
  '不同模型支持的参数可能不同，例如 prompt、image、size、quality、response_format 等，请以模型文档或调用示例为准。':
    'モデルごとに対応パラメータが異なります（prompt、image、size、quality、response_format など）。モデルドキュメントまたは呼び出し例を参照してください。',
  '向量端点用于把文本转换为 embedding，常用于知识库检索、RAG、相似度搜索、聚类、推荐和语义匹配。':
    'ベクトルエンドポイントはテキストを embedding に変換し、ナレッジベース検索、RAG、類似度検索、クラスタリング、レコメンド、意味マッチングに使われます。',
  '图像端点通常用于文生图、图生图、图片编辑或图片分析类能力，适合绘图工具、自动化工作流和自建视觉应用接入。':
    '画像エンドポイントはテキスト→画像、画像→画像、編集、分析などに使われ、描画ツール、自動化ワークフロー、自作ビジョンアプリ向けです。',
  '复制可用于调用上述 API 端点的 API Key':
    '上記 API エンドポイント呼び出し用の API Key をコピー',
  '复制带渠道路由的模型名，可将请求固定到指定渠道':
    'チャネルルーティング付きモデル名をコピーし、指定チャネルへリクエストを固定',
  '多数只要求填写 BaseURL 的工具会自动补全 /v1/chat/completions；如果工具要求填写完整接口地址，就复制这里显示的完整 URL。':
    'BaseURL のみ必要なツールは /v1/chat/completions を自動補完します。完全 URL が必要な場合はここに表示された URL をコピーしてください。',
  '如果你的工具明确支持 Responses API，可以优先选择这个端点；老版 OpenAI 兼容工具通常仍使用聊天补全端点。':
    'Responses API を明示的にサポートするツールはこのエンドポイントを優先。旧 OpenAI 互換ツールは通常チャット補完エンドポイントを使用します。',
  '如果你的工具是知识库或工作流工具，请在 embedding 模型配置里填写这个完整端点地址和对应 API Key。':
    'ナレッジベースやワークフローツールでは、embedding モデル設定にこの完全 URL と API Key を入力してください。',
  '如果客户端只支持 OpenAI 兼容图像接口，通常需要填写完整端点地址，并确认模型名称与通道路由模型名一致。':
    'OpenAI 互換画像 API のみ対応のクライアントでは完全 URL を入力し、モデル名がチャネルルーティング名と一致することを確認してください。',
  '如果工具只提供 BaseURL 输入框，通常填写 BaseURL 即可；如果工具提供接口地址或 Endpoint 输入框，请复制这里的完整 URL。':
    'BaseURL 入力のみのツールは BaseURL で十分です。エンドポイント入力欄がある場合はここの完全 URL をコピーしてください。',
  '按工具要求复制 BaseURL 或完整 API 端点即可。':
    'ツールの要件に応じて BaseURL または完全 API エンドポイントをコピーしてください。',
  '按渠道展示稳定性、路由和当前价格':
    'チャネルごとの安定性、ルーティング、現在価格を表示',
  '接入前建议确认客户端是否支持对应音频接口格式；如果只支持普通聊天接口，请选择聊天补全端点。':
    'クライアントが音声 API 形式に対応しているか確認してください。チャットのみの場合はチャット補完エンドポイントを選択してください。',
  '接入时通常传入 input 和 model，返回的向量会写入向量数据库或检索系统，例如 Milvus、Qdrant、pgvector、Elasticsearch 等。':
    '通常 input と model を渡し、返却ベクトルは Milvus、Qdrant、pgvector、Elasticsearch などのベクトル DB に保存されます。',
  '更适合 Hermes、自建脚本、自建前端/后端、HTTP 工作流节点、Postman、Apifox、curl，或明确支持 OpenAI Videos API / Sora 视频端点的工具。':
    'Hermes、自作スクリプト、フロント/バックエンド、HTTP ワークフロー、Postman、Apifox、curl、OpenAI Videos API / Sora 対応ツールに最適です。',
  '用于需要填写完整接口地址的工具和应用，例如':
    '完全なエンドポイント URL が必要なツール・アプリ向け。例：',
  '等工具、HTTP 工作流工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用；若工具仅支持聊天补全，请优先选择 /v1/chat/completions。':
    ' などのツール、HTTP ワークフロー、自作サービス。ステップ 2 のモデル名とステップ 3 の API Key と併用。チャット補完のみの場合は /v1/chat/completions を優先してください。',
  '等工具、工作流工具或自建应用的服务地址配置。':
    ' などのツール、ワークフローツール、自作アプリのサービスアドレス設定に使用。',
  '等工具、工作流编排工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用。':
    ' などのツール、ワークフロー編成ツール、自作サービス。ステップ 2 のモデル名とステップ 3 の API Key と併用してください。',
  '等工具，也适合支持自定义 HTTP 请求的工作流工具或自建应用。':
    ' などのツール、カスタム HTTP リクエスト対応のワークフローツールや自作アプリにも適しています。',
  聊天补全端点适合: 'チャット補完エンドポイントは以下に適しています：',
  '聊天补全端点，OpenAI 兼容生态中最常用的接口，通常用于 OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、工作流编排工具和自建聊天应用。':
    'チャット補完は OpenAI 互換で最も一般的なエンドポイント。OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、ワークフローツール、自作チャットアプリで使用されます。',
  '视频端点用于创建视频生成任务，常见流程是提交 prompt、model、尺寸或时长等参数后轮询任务结果。适合':
    '動画エンドポイントは動画生成タスクを作成します。prompt、model、サイズ/長さを送信後、結果をポーリング。以下に適しています：',
  '视频端点通常用于视频生成、视频理解、分镜或任务型视频处理，适合创作工具、自动化生产流程和异步任务场景。':
    '動画エンドポイントは生成、理解、絵コンテ、タスク型処理に使われ、創作ツール、自動化パイプライン、非同期タスク向けです。',
  '视频能力往往不是一次请求立即返回最终结果，通常需要先创建视频任务，再轮询任务状态，最后下载或读取生成的视频内容。':
    '動画 API は 1 リクエストで結果を返さないことが多く、タスク作成→ステータスポーリング→ダウンロード/読み取りの流れになります。',
  '该端点表示模型支持的一类具体 API 能力，完整请求地址由 BaseURL 加端点路径组成，适合需要手动填写接口地址的客户端、工作流工具或自建应用。':
    'このエンドポイントはモデルの特定 API 機能を表します。完全 URL は BaseURL + パス。手動で URL を入力するクライアント、ワークフロー、自作アプリ向けです。',
  '说明：如果 WorkBuddy 添加自定义模型时选择了「高级配置」中的「自定义协议」，开启后将直接使用填写的接口地址，不再自动补全 /chat/completions 路径；这种情况下接口地址填写到 /v1 即可，例如：{{url}}。':
    '注：WorkBuddy でカスタムモデル追加時に「高度な設定」→「カスタムプロトコル」を有効にすると、入力 URL をそのまま使用し /chat/completions は自動補完されません。この場合 /v1 までで十分です。例：{{url}}。',
  '请求一般使用 POST，并在请求体里传入 model、messages、temperature、stream 等参数；支持流式输出时可用于打字机效果、长文本持续返回和实时助手场景。':
    '通常 POST で model、messages、temperature、stream 等をボディに渡します。ストリーミングでタイプライター表示、長文逐次返却、リアルタイムアシスタントに使えます。',
  '请求通常使用 POST，模型名称仍需填写通道路由模型名，API Key 使用本侧栏第三步复制的密钥。':
    '通常 POST を使用。モデル名はチャネルルーティング名、API Key は本パネル ステップ 3 でコピーしたキーを使用してください。',
  '调用时通常需要同时配置 API Key 和模型名称；模型名称建议使用第二步中的通道路由模型名，以便请求固定到指定渠道。':
    'API Key とモデル名の両方を設定。ステップ 2 のチャネルルーティング名を使うと指定チャネルに固定できます。',
  '这类接口常需要上传音频文件或指定 voice、input、format 等参数，部分工具会要求使用 multipart/form-data。':
    '音声ファイルのアップロードや voice、input、format 等が必要な場合があります。multipart/form-data が必要なツールもあります。',
  '音频端点通常用于语音转文字、文字转语音、翻译或音频理解，适合客服质检、会议纪要、语音助手和媒体处理工作流。':
    '音声エンドポイントは STT、TTS、翻訳、音声理解に使われ、QA、議事録、音声アシスタント、メディア処理向けです。',
  '建议直连本站点发起请求，关闭系统代理或 VPN，避免代理干扰模型连接与请求转发。':
    '本サイトへ直接接続し、システムプロキシや VPN をオフにしてください。プロキシが接続や転送を妨げる可能性があります。',
  '要调用的模型名称。': '呼び出すモデル名。',
  '组成对话的消息列表。每条消息通常包含 role 和 content 字段。':
    '会話を構成するメッセージのリスト。各メッセージには通常 role と content があります。',
  '单条对话消息。': '単一の会話メッセージ。',
  '消息角色，例如 system、user 或 assistant。':
    'メッセージの役割。例：system、user、assistant。',
  '消息内容。': 'メッセージの内容。',
  '是否使用流式响应。': 'ストリーミングレスポンスを使用するか。',
  '响应 ID。': 'レスポンス ID。',
  '对象类型。': 'オブジェクトタイプ。',
  '响应创建时间戳。': 'レスポンス作成タイムスタンプ。',
  '本次响应使用的模型名称。': 'このレスポンスで使用されたモデル名。',
  '模型生成结果列表。': 'モデル生成結果のリスト。',
  '单个生成结果。': '単一の生成結果。',
  '结果索引。': '結果インデックス。',
  '模型返回的消息。': 'モデルが返したメッセージ。',
  '消息角色。': 'メッセージの役割。',
  '生成结束原因。': '生成終了理由。',
  'Token 使用量统计。': 'Token 使用量統計。',
  '输入 Token 数。': '入力 Token 数。',
  '输出 Token 数。': '出力 Token 数。',
  '总 Token 数。': '合計 Token 数。',
};

const fr = {
  'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于':
    "BaseURL est l'adresse de base des clients compatibles OpenAI, chemin fixé à /v1. Utilisé généralement pour",
  'Responses 端点是 OpenAI 新版统一响应接口，适合需要文本生成、多模态输入、工具调用或更统一响应结构的客户端和自建应用。':
    "Le point d'accès Responses est l'API unifiée d'OpenAI, adaptée aux clients et apps personnalisées nécessitant génération de texte, entrée multimodale, appels d'outils ou structure de réponse unifiée.",
  '不同模型支持的参数可能不同，例如 prompt、image、size、quality、response_format 等，请以模型文档或调用示例为准。':
    'Les paramètres varient selon le modèle (prompt, image, size, quality, response_format, etc.). Consultez la doc ou les exemples.',
  '向量端点用于把文本转换为 embedding，常用于知识库检索、RAG、相似度搜索、聚类、推荐和语义匹配。':
    "Le point d'accès embeddings convertit le texte en embeddings pour la recherche documentaire, RAG, similarité, clustering, recommandations et correspondance sémantique.",
  '图像端点通常用于文生图、图生图、图片编辑或图片分析类能力，适合绘图工具、自动化工作流和自建视觉应用接入。':
    "Les points d'accès image servent au texte-vers-image, image-vers-image, édition ou analyse — outils de dessin, workflows et apps vision.",
  '复制可用于调用上述 API 端点的 API Key':
    "Copier une clé API utilisable pour appeler les points d'accès ci-dessus",
  '复制带渠道路由的模型名，可将请求固定到指定渠道':
    'Copier le nom de modèle routé pour fixer les requêtes à un canal',
  '多数只要求填写 BaseURL 的工具会自动补全 /v1/chat/completions；如果工具要求填写完整接口地址，就复制这里显示的完整 URL。':
    'Les outils ne demandant que BaseURL ajoutent souvent /v1/chat/completions ; sinon copiez l’URL complète affichée ici.',
  '如果你的工具明确支持 Responses API，可以优先选择这个端点；老版 OpenAI 兼容工具通常仍使用聊天补全端点。':
    "Si votre outil supporte Responses API, préférez ce point d'accès ; les anciens outils OpenAI utilisent le chat completions.",
  '如果你的工具是知识库或工作流工具，请在 embedding 模型配置里填写这个完整端点地址和对应 API Key。':
    "Pour bases de connaissances ou workflows, saisissez cette URL complète et la clé API dans la config embedding.",
  '如果客户端只支持 OpenAI 兼容图像接口，通常需要填写完整端点地址，并确认模型名称与通道路由模型名一致。':
    "Si le client ne supporte que les API image OpenAI, saisissez l'URL complète et vérifiez le nom routé.",
  '如果工具只提供 BaseURL 输入框，通常填写 BaseURL 即可；如果工具提供接口地址或 Endpoint 输入框，请复制这里的完整 URL。':
    "Avec un champ BaseURL seul, BaseURL suffit ; avec un champ Endpoint, copiez l'URL complète ici.",
  '按工具要求复制 BaseURL 或完整 API 端点即可。':
    "Copiez BaseURL ou le point d'accès complet selon votre outil.",
  '按渠道展示稳定性、路由和当前价格':
    'Affiche stabilité, routage et tarifs par canal',
  '接入前建议确认客户端是否支持对应音频接口格式；如果只支持普通聊天接口，请选择聊天补全端点。':
    "Vérifiez le format audio du client ; s'il ne supporte que le chat, utilisez chat completions.",
  '接入时通常传入 input 和 model，返回的向量会写入向量数据库或检索系统，例如 Milvus、Qdrant、pgvector、Elasticsearch 等。':
    'Passez input et model ; les vecteurs vont dans Milvus, Qdrant, pgvector, Elasticsearch, etc.',
  '更适合 Hermes、自建脚本、自建前端/后端、HTTP 工作流节点、Postman、Apifox、curl，或明确支持 OpenAI Videos API / Sora 视频端点的工具。':
    'Idéal pour Hermes, scripts, front/back, workflows HTTP, Postman, Apifox, curl, ou outils Videos API / Sora.',
  '用于需要填写完整接口地址的工具和应用，例如':
    "Pour les outils nécessitant une URL complète, par ex.",
  '等工具、HTTP 工作流工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用；若工具仅支持聊天补全，请优先选择 /v1/chat/completions。':
    ' etc. Utilisez avec le modèle (étape 2) et la clé API (étape 3) ; si chat seulement, préférez /v1/chat/completions.',
  '等工具、工作流工具或自建应用的服务地址配置。':
    ' etc. pour la configuration d’adresse de service.',
  '等工具、工作流编排工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用。':
    ' etc. Utilisez avec le nom de modèle (étape 2) et la clé API (étape 3).',
  '等工具，也适合支持自定义 HTTP 请求的工作流工具或自建应用。':
    ' etc., ainsi que workflows ou apps avec requêtes HTTP personnalisées.',
  聊天补全端点适合: 'Le point chat completions convient à',
  '聊天补全端点，OpenAI 兼容生态中最常用的接口，通常用于 OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、工作流编排工具和自建聊天应用。':
    'Chat completions est le plus courant (OpenAI SDK, Cherry Studio, Chatbox, LobeChat, Dify, workflows, apps chat).',
  '视频端点用于创建视频生成任务，常见流程是提交 prompt、model、尺寸或时长等参数后轮询任务结果。适合':
    'Les endpoints vidéo créent des tâches — soumettez prompt, model, taille/durée puis interrogez le statut. Convient à',
  '视频端点通常用于视频生成、视频理解、分镜或任务型视频处理，适合创作工具、自动化生产流程和异步任务场景。':
    'Génération, compréhension, storyboard ou traitement vidéo — outils créatifs, pipelines automatisés, tâches async.',
  '视频能力往往不是一次请求立即返回最终结果，通常需要先创建视频任务，再轮询任务状态，最后下载或读取生成的视频内容。':
    'Souvent : créer une tâche, interroger le statut, puis télécharger/lire la vidéo générée.',
  '该端点表示模型支持的一类具体 API 能力，完整请求地址由 BaseURL 加端点路径组成，适合需要手动填写接口地址的客户端、工作流工具或自建应用。':
    "Ce point d'accès = capacité API ; URL = BaseURL + chemin. Pour clients, workflows ou apps avec URL manuelle.",
  '说明：如果 WorkBuddy 添加自定义模型时选择了「高级配置」中的「自定义协议」，开启后将直接使用填写的接口地址，不再自动补全 /chat/completions 路径；这种情况下接口地址填写到 /v1 即可，例如：{{url}}。':
    'Note WorkBuddy : avec Protocole personnalisé (Avancé), l’URL saisie est utilisée telle quelle ; /v1 suffit, ex. {{url}}.',
  '请求一般使用 POST，并在请求体里传入 model、messages、temperature、stream 等参数；支持流式输出时可用于打字机效果、长文本持续返回和实时助手场景。':
    'POST avec model, messages, temperature, stream… Streaming pour effet machine à écrire et assistants temps réel.',
  '请求通常使用 POST，模型名称仍需填写通道路由模型名，API Key 使用本侧栏第三步复制的密钥。':
    'POST ; nom routé + clé API copiée à l’étape 3 de ce panneau.',
  '调用时通常需要同时配置 API Key 和模型名称；模型名称建议使用第二步中的通道路由模型名，以便请求固定到指定渠道。':
    'Configurez clé API et nom ; utilisez le nom routé (étape 2) pour fixer le canal.',
  '这类接口常需要上传音频文件或指定 voice、input、format 等参数，部分工具会要求使用 multipart/form-data。':
    'Upload audio ou paramètres voice, input, format ; parfois multipart/form-data.',
  '音频端点通常用于语音转文字、文字转语音、翻译或音频理解，适合客服质检、会议纪要、语音助手和媒体处理工作流。':
    'STT, TTS, traduction, compréhension audio — QA, comptes rendus, assistants vocaux, média.',
  '建议直连本站点发起请求，关闭系统代理或 VPN，避免代理干扰模型连接与请求转发。':
    'Connectez-vous directement ; désactivez proxy/VPN pour éviter les interférences.',
  '要调用的模型名称。': 'Nom du modèle à appeler.',
  '组成对话的消息列表。每条消息通常包含 role 和 content 字段。':
    'Liste des messages ; chaque message a role et content.',
  '单条对话消息。': 'Un message de conversation.',
  '消息角色，例如 system、user 或 assistant。': 'Rôle : system, user ou assistant.',
  '消息内容。': 'Contenu du message.',
  '是否使用流式响应。': 'Utiliser le streaming.',
  '响应 ID。': 'ID de réponse.',
  '对象类型。': "Type d'objet.",
  '响应创建时间戳。': 'Horodatage de création.',
  '本次响应使用的模型名称。': 'Modèle utilisé dans cette réponse.',
  '模型生成结果列表。': 'Liste des résultats générés.',
  '单个生成结果。': 'Un résultat de génération.',
  '结果索引。': 'Index du résultat.',
  '模型返回的消息。': 'Message retourné par le modèle.',
  '消息角色。': 'Rôle du message.',
  '生成结束原因。': 'Raison de fin de génération.',
  'Token 使用量统计。': 'Statistiques d’utilisation des tokens.',
  '输入 Token 数。': 'Tokens en entrée.',
  '输出 Token 数。': 'Tokens en sortie.',
  '总 Token 数。': 'Total de tokens.',
};

const ru = {
  'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于':
    'BaseURL — базовый адрес OpenAI-совместимых клиентов, путь фиксирован как /v1. Обычно используется для',
  'Responses 端点是 OpenAI 新版统一响应接口，适合需要文本生成、多模态输入、工具调用或更统一响应结构的客户端和自建应用。':
    'Responses — унифицированный API OpenAI для генерации текста, мультимодального ввода, вызова инструментов и единой структуры ответа.',
  '不同模型支持的参数可能不同，例如 prompt、image、size、quality、response_format 等，请以模型文档或调用示例为准。':
    'Параметры зависят от модели (prompt, image, size, quality, response_format и т.д.). См. документацию.',
  '向量端点用于把文本转换为 embedding，常用于知识库检索、RAG、相似度搜索、聚类、推荐和语义匹配。':
    'Embeddings преобразуют текст в векторы для RAG, поиска, кластеризации, рекомендаций и семантического сопоставления.',
  '图像端点通常用于文生图、图生图、图片编辑或图片分析类能力，适合绘图工具、自动化工作流和自建视觉应用接入。':
    'Image API для text-to-image, image-to-image, редактирования и анализа — инструменты рисования и vision-приложения.',
  '复制可用于调用上述 API 端点的 API Key':
    'Скопировать API Key для вызова указанных endpoint',
  '复制带渠道路由的模型名，可将请求固定到指定渠道':
    'Скопировать имя модели с маршрутизацией для фиксации канала',
  '多数只要求填写 BaseURL 的工具会自动补全 /v1/chat/completions；如果工具要求填写完整接口地址，就复制这里显示的完整 URL。':
    'Инструменты с полем BaseURL часто дополняют /v1/chat/completions; иначе скопируйте полный URL отсюда.',
  '如果你的工具明确支持 Responses API，可以优先选择这个端点；老版 OpenAI 兼容工具通常仍使用聊天补全端点。':
    'При поддержке Responses API выбирайте этот endpoint; старые инструменты используют chat completions.',
  '如果你的工具是知识库或工作流工具，请在 embedding 模型配置里填写这个完整端点地址和对应 API Key。':
    'Для баз знаний и workflow укажите полный URL и API Key в настройках embedding-модели.',
  '如果客户端只支持 OpenAI 兼容图像接口，通常需要填写完整端点地址，并确认模型名称与通道路由模型名一致。':
    'Для OpenAI-совместимых image API укажите полный URL и проверьте имя маршрутизированной модели.',
  '如果工具只提供 BaseURL 输入框，通常填写 BaseURL 即可；如果工具提供接口地址或 Endpoint 输入框，请复制这里的完整 URL。':
    'Если только BaseURL — достаточно его; если есть поле Endpoint — скопируйте полный URL.',
  '按工具要求复制 BaseURL 或完整 API 端点即可。':
    'Скопируйте BaseURL или полный endpoint согласно требованиям инструмента.',
  '按渠道展示稳定性、路由和当前价格':
    'Показывает стабильность, маршрутизацию и цены по каналам',
  '接入前建议确认客户端是否支持对应音频接口格式；如果只支持普通聊天接口，请选择聊天补全端点。':
    'Убедитесь, что клиент поддерживает аудио API; для чата используйте chat completions.',
  '接入时通常传入 input 和 model，返回的向量会写入向量数据库或检索系统，例如 Milvus、Qdrant、pgvector、Elasticsearch 等。':
    'Передайте input и model; векторы сохраняются в Milvus, Qdrant, pgvector, Elasticsearch и т.д.',
  '更适合 Hermes、自建脚本、自建前端/后端、HTTP 工作流节点、Postman、Apifox、curl，或明确支持 OpenAI Videos API / Sora 视频端点的工具。':
    'Подходит для Hermes, скриптов, front/back, HTTP workflow, Postman, Apifox, curl, Videos API / Sora.',
  '用于需要填写完整接口地址的工具和应用，例如':
    'Для инструментов, требующих полный URL, например',
  '等工具、HTTP 工作流工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用；若工具仅支持聊天补全，请优先选择 /v1/chat/completions。':
    ' и др. Используйте с моделью (шаг 2) и API Key (шаг 3); для chat-only — /v1/chat/completions.',
  '等工具、工作流工具或自建应用的服务地址配置。':
    ' и др. для настройки адреса сервиса.',
  '等工具、工作流编排工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用。':
    ' и др. Используйте с моделью (шаг 2) и API Key (шаг 3).',
  '等工具，也适合支持自定义 HTTP 请求的工作流工具或自建应用。':
    ' и др., а также workflow и приложения с кастомными HTTP-запросами.',
  聊天补全端点适合: 'Chat completions подходит для',
  '聊天补全端点，OpenAI 兼容生态中最常用的接口，通常用于 OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、工作流编排工具和自建聊天应用。':
    'Chat completions — самый распространённый endpoint (OpenAI SDK, Cherry Studio, Chatbox, LobeChat, Dify, workflow, chat-apps).',
  '视频端点用于创建视频生成任务，常见流程是提交 prompt、model、尺寸或时长等参数后轮询任务结果。适合':
    'Video endpoint создаёт задачи генерации — отправьте prompt, model, размер/длительность и опрашивайте статус. Подходит для',
  '视频端点通常用于视频生成、视频理解、分镜或任务型视频处理，适合创作工具、自动化生产流程和异步任务场景。':
    'Генерация, понимание, раскадровка видео — творческие инструменты, автоматизация, async-задачи.',
  '视频能力往往不是一次请求立即返回最终结果，通常需要先创建视频任务，再轮询任务状态，最后下载或读取生成的视频内容。':
    'Обычно: создать задачу → опрос статуса → скачать/прочитать видео.',
  '该端点表示模型支持的一类具体 API 能力，完整请求地址由 BaseURL 加端点路径组成，适合需要手动填写接口地址的客户端、工作流工具或自建应用。':
    'Endpoint = возможность API; полный URL = BaseURL + путь. Для клиентов с ручным вводом URL.',
  '说明：如果 WorkBuddy 添加自定义模型时选择了「高级配置」中的「自定义协议」，开启后将直接使用填写的接口地址，不再自动补全 /chat/completions 路径；这种情况下接口地址填写到 /v1 即可，例如：{{url}}。':
    'WorkBuddy: при «Пользовательском протоколе» URL используется как есть; достаточно /v1, напр. {{url}}.',
  '请求一般使用 POST，并在请求体里传入 model、messages、temperature、stream 等参数；支持流式输出时可用于打字机效果、长文本持续返回和实时助手场景。':
    'POST с model, messages, temperature, stream… Streaming для эффекта печати и realtime-ассистентов.',
  '请求通常使用 POST，模型名称仍需填写通道路由模型名，API Key 使用本侧栏第三步复制的密钥。':
    'POST; имя маршрутизированной модели + API Key из шага 3 этой панели.',
  '调用时通常需要同时配置 API Key 和模型名称；模型名称建议使用第二步中的通道路由模型名，以便请求固定到指定渠道。':
    'Настройте API Key и имя модели; используйте имя из шага 2 для фиксации канала.',
  '这类接口常需要上传音频文件或指定 voice、input、format 等参数，部分工具会要求使用 multipart/form-data。':
    'Может потребоваться загрузка аудио или voice, input, format; иногда multipart/form-data.',
  '音频端点通常用于语音转文字、文字转语音、翻译或音频理解，适合客服质检、会议纪要、语音助手和媒体处理工作流。':
    'STT, TTS, перевод, понимание аудио — QA, протоколы, голосовые ассистенты, медиа.',
  '建议直连本站点发起请求，关闭系统代理或 VPN，避免代理干扰模型连接与请求转发。':
    'Подключайтесь напрямую; отключите прокси/VPN, чтобы не мешать соединению.',
  '要调用的模型名称。': 'Имя вызываемой модели.',
  '组成对话的消息列表。每条消息通常包含 role 和 content 字段。':
    'Список сообщений диалога; каждое обычно содержит role и content.',
  '单条对话消息。': 'Одно сообщение диалога.',
  '消息角色，例如 system、user 或 assistant。': 'Роль: system, user или assistant.',
  '消息内容。': 'Содержимое сообщения.',
  '是否使用流式响应。': 'Использовать потоковый ответ.',
  '响应 ID。': 'ID ответа.',
  '对象类型。': 'Тип объекта.',
  '响应创建时间戳。': 'Время создания ответа.',
  '本次响应使用的模型名称。': 'Модель, использованная в ответе.',
  '模型生成结果列表。': 'Список результатов генерации.',
  '单个生成结果。': 'Один результат генерации.',
  '结果索引。': 'Индекс результата.',
  '模型返回的消息。': 'Сообщение, возвращённое моделью.',
  '消息角色。': 'Роль сообщения.',
  '生成结束原因。': 'Причина завершения генерации.',
  'Token 使用量统计。': 'Статистика использования токенов.',
  '输入 Token 数。': 'Входные токены.',
  '输出 Token 数。': 'Выходные токены.',
  '总 Token 数。': 'Всего токенов.',
};

const vi = {
  'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于':
    'BaseURL là địa chỉ cơ sở cho client tương thích OpenAI, đường dẫn cố định /v1. Thường dùng cho',
  'Responses 端点是 OpenAI 新版统一响应接口，适合需要文本生成、多模态输入、工具调用或更统一响应结构的客户端和自建应用。':
    'Responses là API phản hồi thống nhất của OpenAI, phù hợp tạo văn bản, đầu vào đa phương thức, gọi công cụ và cấu trúc phản hồi thống nhất.',
  '不同模型支持的参数可能不同，例如 prompt、image、size、quality、response_format 等，请以模型文档或调用示例为准。':
    'Tham số khác nhau theo mô hình (prompt, image, size, quality, response_format…). Xem tài liệu hoặc ví dụ.',
  '向量端点用于把文本转换为 embedding，常用于知识库检索、RAG、相似度搜索、聚类、推荐和语义匹配。':
    'Endpoint embedding chuyển văn bản thành vector cho RAG, tìm kiếm, phân cụm, gợi ý và khớp ngữ nghĩa.',
  '图像端点通常用于文生图、图生图、图片编辑或图片分析类能力，适合绘图工具、自动化工作流和自建视觉应用接入。':
    'Endpoint hình ảnh cho text-to-image, image-to-image, chỉnh sửa, phân tích — công cụ vẽ và ứng dụng vision.',
  '复制可用于调用上述 API 端点的 API Key':
    'Sao chép API Key để gọi các endpoint API ở trên',
  '复制带渠道路由的模型名，可将请求固定到指定渠道':
    'Sao chép tên mô hình định tuyến kênh để cố định yêu cầu vào kênh',
  '多数只要求填写 BaseURL 的工具会自动补全 /v1/chat/completions；如果工具要求填写完整接口地址，就复制这里显示的完整 URL。':
    'Công cụ chỉ cần BaseURL thường tự thêm /v1/chat/completions; nếu cần URL đầy đủ, sao chép URL hiển thị ở đây.',
  '如果你的工具明确支持 Responses API，可以优先选择这个端点；老版 OpenAI 兼容工具通常仍使用聊天补全端点。':
    'Nếu hỗ trợ Responses API, ưu tiên endpoint này; công cụ OpenAI cũ thường dùng chat completions.',
  '如果你的工具是知识库或工作流工具，请在 embedding 模型配置里填写这个完整端点地址和对应 API Key。':
    'Với cơ sở tri thức/workflow, nhập URL đầy đủ và API Key trong cấu hình embedding.',
  '如果客户端只支持 OpenAI 兼容图像接口，通常需要填写完整端点地址，并确认模型名称与通道路由模型名一致。':
    'Client chỉ hỗ trợ image API OpenAI cần URL đầy đủ và tên mô hình khớp tên định tuyến kênh.',
  '如果工具只提供 BaseURL 输入框，通常填写 BaseURL 即可；如果工具提供接口地址或 Endpoint 输入框，请复制这里的完整 URL。':
    'Chỉ có BaseURL thì BaseURL là đủ; có trường Endpoint thì sao chép URL đầy đủ ở đây.',
  '按工具要求复制 BaseURL 或完整 API 端点即可。':
    'Sao chép BaseURL hoặc endpoint API đầy đủ theo yêu cầu công cụ.',
  '按渠道展示稳定性、路由和当前价格':
    'Hiển thị độ ổn định, định tuyến và giá theo kênh',
  '接入前建议确认客户端是否支持对应音频接口格式；如果只支持普通聊天接口，请选择聊天补全端点。':
    'Xác nhận client hỗ trợ định dạng audio; nếu chỉ chat, chọn chat completions.',
  '接入时通常传入 input 和 model，返回的向量会写入向量数据库或检索系统，例如 Milvus、Qdrant、pgvector、Elasticsearch 等。':
    'Truyền input và model; vector lưu vào Milvus, Qdrant, pgvector, Elasticsearch…',
  '更适合 Hermes、自建脚本、自建前端/后端、HTTP 工作流节点、Postman、Apifox、curl，或明确支持 OpenAI Videos API / Sora 视频端点的工具。':
    'Phù hợp Hermes, script, front/back, workflow HTTP, Postman, Apifox, curl, Videos API / Sora.',
  '用于需要填写完整接口地址的工具和应用，例如':
    'Cho công cụ/ứng dụng cần URL endpoint đầy đủ, ví dụ',
  '等工具、HTTP 工作流工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用；若工具仅支持聊天补全，请优先选择 /v1/chat/completions。':
    ' v.v. Dùng với tên mô hình (bước 2) và API Key (bước 3); chat-only thì ưu tiên /v1/chat/completions.',
  '等工具、工作流工具或自建应用的服务地址配置。':
    ' v.v. để cấu hình địa chỉ dịch vụ.',
  '等工具、工作流编排工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用。':
    ' v.v. Dùng với tên mô hình (bước 2) và API Key (bước 3).',
  '等工具，也适合支持自定义 HTTP 请求的工作流工具或自建应用。':
    ' v.v., cũng phù hợp workflow hoặc ứng dụng hỗ trợ HTTP tùy chỉnh.',
  聊天补全端点适合: 'Endpoint chat completions phù hợp với',
  '聊天补全端点，OpenAI 兼容生态中最常用的接口，通常用于 OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、工作流编排工具和自建聊天应用。':
    'Chat completions là endpoint OpenAI phổ biến nhất (OpenAI SDK, Cherry Studio, Chatbox, LobeChat, Dify, workflow, chat app).',
  '视频端点用于创建视频生成任务，常见流程是提交 prompt、model、尺寸或时长等参数后轮询任务结果。适合':
    'Endpoint video tạo tác vụ sinh video — gửi prompt, model, kích thước/thời lượng rồi poll kết quả. Phù hợp',
  '视频端点通常用于视频生成、视频理解、分镜或任务型视频处理，适合创作工具、自动化生产流程和异步任务场景。':
    'Sinh video, hiểu video, storyboard — công cụ sáng tạo, pipeline tự động, tác vụ bất đồng bộ.',
  '视频能力往往不是一次请求立即返回最终结果，通常需要先创建视频任务，再轮询任务状态，最后下载或读取生成的视频内容。':
    'Thường: tạo tác vụ → poll trạng thái → tải/đọc video đã sinh.',
  '该端点表示模型支持的一类具体 API 能力，完整请求地址由 BaseURL 加端点路径组成，适合需要手动填写接口地址的客户端、工作流工具或自建应用。':
    'Endpoint = khả năng API; URL đầy đủ = BaseURL + đường dẫn. Cho client/workflow nhập URL thủ công.',
  '说明：如果 WorkBuddy 添加自定义模型时选择了「高级配置」中的「自定义协议」，开启后将直接使用填写的接口地址，不再自动补全 /chat/completions 路径；这种情况下接口地址填写到 /v1 即可，例如：{{url}}。':
    'WorkBuddy: bật Giao thức tùy chỉnh (Nâng cao) thì dùng URL nhập trực tiếp; chỉ cần /v1, vd. {{url}}.',
  '请求一般使用 POST，并在请求体里传入 model、messages、temperature、stream 等参数；支持流式输出时可用于打字机效果、长文本持续返回和实时助手场景。':
    'POST với model, messages, temperature, stream… Streaming cho hiệu ứng gõ và trợ lý thời gian thực.',
  '请求通常使用 POST，模型名称仍需填写通道路由模型名，API Key 使用本侧栏第三步复制的密钥。':
    'POST; tên mô hình định tuyến + API Key sao chép ở bước 3.',
  '调用时通常需要同时配置 API Key 和模型名称；模型名称建议使用第二步中的通道路由模型名，以便请求固定到指定渠道。':
    'Cấu hình API Key và tên mô hình; dùng tên bước 2 để cố định kênh.',
  '这类接口常需要上传音频文件或指定 voice、input、format 等参数，部分工具会要求使用 multipart/form-data。':
    'Có thể cần tải audio hoặc voice, input, format; một số công cụ cần multipart/form-data.',
  '音频端点通常用于语音转文字、文字转语音、翻译或音频理解，适合客服质检、会议纪要、语音助手和媒体处理工作流。':
    'STT, TTS, dịch, hiểu audio — QA, biên bản, trợ lý giọng nói, xử lý media.',
  '建议直连本站点发起请求，关闭系统代理或 VPN，避免代理干扰模型连接与请求转发。':
    'Kết nối trực tiếp; tắt proxy/VPN hệ thống để tránh ảnh hưởng kết nối.',
  '要调用的模型名称。': 'Tên mô hình cần gọi.',
  '组成对话的消息列表。每条消息通常包含 role 和 content 字段。':
    'Danh sách tin nhắn hội thoại; mỗi tin thường có role và content.',
  '单条对话消息。': 'Một tin nhắn hội thoại.',
  '消息角色，例如 system、user 或 assistant。': 'Vai trò: system, user hoặc assistant.',
  '消息内容。': 'Nội dung tin nhắn.',
  '是否使用流式响应。': 'Có dùng phản hồi streaming.',
  '响应 ID。': 'ID phản hồi.',
  '对象类型。': 'Loại đối tượng.',
  '响应创建时间戳。': 'Thời gian tạo phản hồi.',
  '本次响应使用的模型名称。': 'Tên mô hình dùng trong phản hồi này.',
  '模型生成结果列表。': 'Danh sách kết quả sinh.',
  '单个生成结果。': 'Một kết quả sinh.',
  '结果索引。': 'Chỉ số kết quả.',
  '模型返回的消息。': 'Tin nhắn model trả về.',
  '消息角色。': 'Vai trò tin nhắn.',
  '生成结束原因。': 'Lý do kết thúc sinh.',
  'Token 使用量统计。': 'Thống kê sử dụng token.',
  '输入 Token 数。': 'Token đầu vào.',
  '输出 Token 数。': 'Token đầu ra.',
  '总 Token 数。': 'Tổng token.',
};

const id = {
  'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于':
    'BaseURL adalah alamat dasar klien kompatibel OpenAI, path tetap /v1. Biasanya untuk',
  'Responses 端点是 OpenAI 新版统一响应接口，适合需要文本生成、多模态输入、工具调用或更统一响应结构的客户端和自建应用。':
    'Endpoint Responses adalah API respons terpadu OpenAI untuk generasi teks, input multimodal, pemanggilan alat, dan struktur respons terpadu.',
  '不同模型支持的参数可能不同，例如 prompt、image、size、quality、response_format 等，请以模型文档或调用示例为准。':
    'Parameter bervariasi per model (prompt, image, size, quality, response_format, dll.). Lihat dokumentasi.',
  '向量端点用于把文本转换为 embedding，常用于知识库检索、RAG、相似度搜索、聚类、推荐和语义匹配。':
    'Endpoint embedding mengubah teks menjadi vektor untuk RAG, pencarian, clustering, rekomendasi, dan pencocokan semantik.',
  '图像端点通常用于文生图、图生图、图片编辑或图片分析类能力，适合绘图工具、自动化工作流和自建视觉应用接入。':
    'Endpoint gambar untuk text-to-image, image-to-image, edit, analisis — alat gambar dan aplikasi vision.',
  '复制可用于调用上述 API 端点的 API Key':
    'Salin API Key untuk memanggil endpoint API di atas',
  '复制带渠道路由的模型名，可将请求固定到指定渠道':
    'Salin nama model routing saluran untuk mengunci permintaan ke saluran',
  '多数只要求填写 BaseURL 的工具会自动补全 /v1/chat/completions；如果工具要求填写完整接口地址，就复制这里显示的完整 URL。':
    'Alat yang hanya butuh BaseURL biasanya menambahkan /v1/chat/completions; jika perlu URL lengkap, salin dari sini.',
  '如果你的工具明确支持 Responses API，可以优先选择这个端点；老版 OpenAI 兼容工具通常仍使用聊天补全端点。':
    'Jika mendukung Responses API, pilih endpoint ini; alat OpenAI lama biasanya chat completions.',
  '如果你的工具是知识库或工作流工具，请在 embedding 模型配置里填写这个完整端点地址和对应 API Key。':
    'Untuk basis pengetahuan/workflow, masukkan URL lengkap dan API Key di konfigurasi embedding.',
  '如果客户端只支持 OpenAI 兼容图像接口，通常需要填写完整端点地址，并确认模型名称与通道路由模型名一致。':
    'Klien image API OpenAI perlu URL lengkap dan nama model sesuai routing saluran.',
  '如果工具只提供 BaseURL 输入框，通常填写 BaseURL 即可；如果工具提供接口地址或 Endpoint 输入框，请复制这里的完整 URL。':
    'Hanya BaseURL cukup jika hanya ada field BaseURL; jika ada Endpoint, salin URL lengkap di sini.',
  '按工具要求复制 BaseURL 或完整 API 端点即可。':
    'Salin BaseURL atau endpoint API lengkap sesuai alat.',
  '按渠道展示稳定性、路由和当前价格':
    'Menampilkan stabilitas, routing, dan harga per saluran',
  '接入前建议确认客户端是否支持对应音频接口格式；如果只支持普通聊天接口，请选择聊天补全端点。':
    'Pastikan klien mendukung format audio; jika hanya chat, pilih chat completions.',
  '接入时通常传入 input 和 model，返回的向量会写入向量数据库或检索系统，例如 Milvus、Qdrant、pgvector、Elasticsearch 等。':
    'Kirim input dan model; vektor disimpan di Milvus, Qdrant, pgvector, Elasticsearch, dll.',
  '更适合 Hermes、自建脚本、自建前端/后端、HTTP 工作流节点、Postman、Apifox、curl，或明确支持 OpenAI Videos API / Sora 视频端点的工具。':
    'Cocok untuk Hermes, skrip, front/back, workflow HTTP, Postman, Apifox, curl, Videos API / Sora.',
  '用于需要填写完整接口地址的工具和应用，例如':
    'Untuk alat/aplikasi yang membutuhkan URL endpoint lengkap, mis.',
  '等工具、HTTP 工作流工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用；若工具仅支持聊天补全，请优先选择 /v1/chat/completions。':
    ' dll. Gunakan dengan nama model (langkah 2) dan API Key (langkah 3); chat-only pilih /v1/chat/completions.',
  '等工具、工作流工具或自建应用的服务地址配置。':
    ' dll. untuk konfigurasi alamat layanan.',
  '等工具、工作流编排工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用。':
    ' dll. Gunakan dengan nama model (langkah 2) dan API Key (langkah 3).',
  '等工具，也适合支持自定义 HTTP 请求的工作流工具或自建应用。':
    ' dll., juga workflow atau aplikasi dengan permintaan HTTP kustom.',
  聊天补全端点适合: 'Endpoint chat completions cocok untuk',
  '聊天补全端点，OpenAI 兼容生态中最常用的接口，通常用于 OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、工作流编排工具和自建聊天应用。':
    'Chat completions paling umum (OpenAI SDK, Cherry Studio, Chatbox, LobeChat, Dify, workflow, chat app).',
  '视频端点用于创建视频生成任务，常见流程是提交 prompt、model、尺寸或时长等参数后轮询任务结果。适合':
    'Endpoint video membuat tugas generasi — kirim prompt, model, ukuran/durasi lalu poll. Cocok untuk',
  '视频端点通常用于视频生成、视频理解、分镜或任务型视频处理，适合创作工具、自动化生产流程和异步任务场景。':
    'Generasi, pemahaman, storyboard video — alat kreatif, pipeline otomatis, tugas async.',
  '视频能力往往不是一次请求立即返回最终结果，通常需要先创建视频任务，再轮询任务状态，最后下载或读取生成的视频内容。':
    'Biasanya: buat tugas → poll status → unduh/baca video hasil.',
  '该端点表示模型支持的一类具体 API 能力，完整请求地址由 BaseURL 加端点路径组成，适合需要手动填写接口地址的客户端、工作流工具或自建应用。':
    'Endpoint = kemampuan API; URL lengkap = BaseURL + path. Untuk klien/workflow dengan URL manual.',
  '说明：如果 WorkBuddy 添加自定义模型时选择了「高级配置」中的「自定义协议」，开启后将直接使用填写的接口地址，不再自动补全 /chat/completions 路径；这种情况下接口地址填写到 /v1 即可，例如：{{url}}。':
    'WorkBuddy: Protokol Kustom (Lanjutan) memakai URL apa adanya; cukup /v1, contoh {{url}}.',
  '请求一般使用 POST，并在请求体里传入 model、messages、temperature、stream 等参数；支持流式输出时可用于打字机效果、长文本持续返回和实时助手场景。':
    'POST dengan model, messages, temperature, stream… Streaming untuk efek ketik dan asisten real-time.',
  '请求通常使用 POST，模型名称仍需填写通道路由模型名，API Key 使用本侧栏第三步复制的密钥。':
    'POST; nama model routing + API Key dari langkah 3 panel ini.',
  '调用时通常需要同时配置 API Key 和模型名称；模型名称建议使用第二步中的通道路由模型名，以便请求固定到指定渠道。':
    'Konfigurasi API Key dan nama model; gunakan nama langkah 2 untuk mengunci saluran.',
  '这类接口常需要上传音频文件或指定 voice、input、format 等参数，部分工具会要求使用 multipart/form-data。':
    'Mungkin perlu unggah audio atau voice, input, format; sebagian alat butuh multipart/form-data.',
  '音频端点通常用于语音转文字、文字转语音、翻译或音频理解，适合客服质检、会议纪要、语音助手和媒体处理工作流。':
    'STT, TTS, terjemahan, pemahaman audio — QA, notulen, asisten suara, media.',
  '建议直连本站点发起请求，关闭系统代理或 VPN，避免代理干扰模型连接与请求转发。':
    'Hubungkan langsung; matikan proxy/VPN sistem agar tidak mengganggu koneksi.',
  '要调用的模型名称。': 'Nama model yang akan dipanggil.',
  '组成对话的消息列表。每条消息通常包含 role 和 content 字段。':
    'Daftar pesan percakapan; setiap pesan biasanya punya role dan content.',
  '单条对话消息。': 'Satu pesan percakapan.',
  '消息角色，例如 system、user 或 assistant。': 'Peran: system, user, atau assistant.',
  '消息内容。': 'Isi pesan.',
  '是否使用流式响应。': 'Gunakan respons streaming.',
  '响应 ID。': 'ID respons.',
  '对象类型。': 'Tipe objek.',
  '响应创建时间戳。': 'Stempel waktu pembuatan respons.',
  '本次响应使用的模型名称。': 'Nama model dalam respons ini.',
  '模型生成结果列表。': 'Daftar hasil generasi model.',
  '单个生成结果。': 'Satu hasil generasi.',
  '结果索引。': 'Indeks hasil.',
  '模型返回的消息。': 'Pesan yang dikembalikan model.',
  '消息角色。': 'Peran pesan.',
  '生成结束原因。': 'Alasan generasi berakhir.',
  'Token 使用量统计。': 'Statistik penggunaan token.',
  '输入 Token 数。': 'Token masukan.',
  '输出 Token 数。': 'Token keluaran.',
  '总 Token 数。': 'Total token.',
};

const th = {
  'BaseURL 是 OpenAI 兼容客户端的基础地址，路径固定为 /v1。通常用于':
    'BaseURL คือที่อยู่ฐานของไคลเอนต์ที่รองรับ OpenAI โดย path คือ /v1 มักใช้กับ',
  'Responses 端点是 OpenAI 新版统一响应接口，适合需要文本生成、多模态输入、工具调用或更统一响应结构的客户端和自建应用。':
    'Responses เป็น API ตอบกลับแบบรวมของ OpenAI เหมาะกับการสร้างข้อความ อินพุตหลายรูปแบบ การเรียกเครื่องมือ และโครงสร้างตอบกลับที่เป็นมาตรฐาน',
  '不同模型支持的参数可能不同，例如 prompt、image、size、quality、response_format 等，请以模型文档或调用示例为准。':
    'พารามิเตอร์แตกต่างตามโมเดล (prompt, image, size, quality, response_format ฯลฯ) โปรดดูเอกสารหรือตัวอย่างการเรียก',
  '向量端点用于把文本转换为 embedding，常用于知识库检索、RAG、相似度搜索、聚类、推荐和语义匹配。':
    'endpoint embedding แปลงข้อความเป็นเวกเตอร์ ใช้กับ RAG การค้นหา การจัดกลุ่ม การแนะนำ และการจับคู่ความหมาย',
  '图像端点通常用于文生图、图生图、图片编辑或图片分析类能力，适合绘图工具、自动化工作流和自建视觉应用接入。':
    'endpoint รูปภาพใช้สำหรับ text-to-image, image-to-image การแก้ไขและวิเคราะห์ เหมาะกับเครื่องมือวาดและแอป vision',
  '复制可用于调用上述 API 端点的 API Key':
    'คัดลอก API Key สำหรับเรียก endpoint API ด้านบน',
  '复制带渠道路由的模型名，可将请求固定到指定渠道':
    'คัดลอกชื่อโมเดลที่มี channel routing เพื่อผูกคำขอกับช่องทาง',
  '多数只要求填写 BaseURL 的工具会自动补全 /v1/chat/completions；如果工具要求填写完整接口地址，就复制这里显示的完整 URL。':
    'เครื่องมือที่ต้องการแค่ BaseURL มักเติม /v1/chat/completions อัตโนมัติ หากต้องการ URL เต็ม ให้คัดลอกจากที่นี่',
  '如果你的工具明确支持 Responses API，可以优先选择这个端点；老版 OpenAI 兼容工具通常仍使用聊天补全端点。':
    'หากรองรับ Responses API ให้เลือก endpoint นี้ เครื่องมือ OpenAI รุ่นเก่ามักใช้ chat completions',
  '如果你的工具是知识库或工作流工具，请在 embedding 模型配置里填写这个完整端点地址和对应 API Key。':
    'สำหรับ knowledge base/workflow ให้ใส่ URL เต็มและ API Key ในการตั้งค่า embedding',
  '如果客户端只支持 OpenAI 兼容图像接口，通常需要填写完整端点地址，并确认模型名称与通道路由模型名一致。':
    'ไคลเอนต์ image API แบบ OpenAI ต้องใส่ URL เต็มและชื่อโมเดลตรงกับชื่อ routing ช่องทาง',
  '如果工具只提供 BaseURL 输入框，通常填写 BaseURL 即可；如果工具提供接口地址或 Endpoint 输入框，请复制这里的完整 URL。':
    'มีแค่ช่อง BaseURL ก็เพียงพอ หากมีช่อง Endpoint ให้คัดลอก URL เต็มจากที่นี่',
  '按工具要求复制 BaseURL 或完整 API 端点即可。':
    'คัดลอก BaseURL หรือ endpoint API เต็มตามที่เครื่องมือต้องการ',
  '按渠道展示稳定性、路由和当前价格':
    'แสดงความเสถียร เส้นทาง และราคาตามช่องทาง',
  '接入前建议确认客户端是否支持对应音频接口格式；如果只支持普通聊天接口，请选择聊天补全端点。':
    'ตรวจสอบว่าไคลเอนต์รองรับรูปแบบ audio หากรองรับแค่แชท ให้เลือก chat completions',
  '接入时通常传入 input 和 model，返回的向量会写入向量数据库或检索系统，例如 Milvus、Qdrant、pgvector、Elasticsearch 等。':
    'ส่ง input และ model โดยทั่วไป เวกเตอร์จะเก็บใน Milvus, Qdrant, pgvector, Elasticsearch ฯลฯ',
  '更适合 Hermes、自建脚本、自建前端/后端、HTTP 工作流节点、Postman、Apifox、curl，或明确支持 OpenAI Videos API / Sora 视频端点的工具。':
    'เหมาะกับ Hermes สคริปต์ front/back workflow HTTP Postman Apifox curl และเครื่องมือ Videos API / Sora',
  '用于需要填写完整接口地址的工具和应用，例如':
    'สำหรับเครื่องมือ/แอปที่ต้องใส่ URL endpoint เต็ม เช่น',
  '等工具、HTTP 工作流工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用；若工具仅支持聊天补全，请优先选择 /v1/chat/completions。':
    ' ฯลฯ ใช้ร่วมกับชื่อโมเดล (ขั้น 2) และ API Key (ขั้น 3) หากรองรับแค่แชท ให้ใช้ /v1/chat/completions',
  '等工具、工作流工具或自建应用的服务地址配置。':
    ' ฯลฯ สำหรับตั้งค่าที่อยู่บริการ',
  '等工具、工作流编排工具或自建服务。请配合第二步模型名字和第三步 API Key 一起使用。':
    ' ฯลฯ ใช้ร่วมกับชื่อโมเดล (ขั้น 2) และ API Key (ขั้น 3)',
  '等工具，也适合支持自定义 HTTP 请求的工作流工具或自建应用。':
    ' ฯลฯ รวมถึง workflow หรือแอปที่รองรับ HTTP แบบกำหนดเอง',
  聊天补全端点适合: 'endpoint chat completions เหมาะกับ',
  '聊天补全端点，OpenAI 兼容生态中最常用的接口，通常用于 OpenAI SDK、Cherry Studio、Chatbox、LobeChat、Dify、工作流编排工具和自建聊天应用。':
    'chat completions เป็น endpoint ที่ใช้มากที่สุด (OpenAI SDK, Cherry Studio, Chatbox, LobeChat, Dify, workflow, แอปแชท)',
  '视频端点用于创建视频生成任务，常见流程是提交 prompt、model、尺寸或时长等参数后轮询任务结果。适合':
    'endpoint วิดีโอสร้างงานสร้างวิดีโอ ส่ง prompt, model, ขนาด/ระยะเวลา แล้ว poll ผล เหมาะกับ',
  '视频端点通常用于视频生成、视频理解、分镜或任务型视频处理，适合创作工具、自动化生产流程和异步任务场景。':
    'ใช้สร้าง วิเคราะห์ storyboard วิดีโอ เหมาะกับเครื่องมือสร้างสรรค์ pipeline อัตโนมัติ และงาน async',
  '视频能力往往不是一次请求立即返回最终结果，通常需要先创建视频任务，再轮询任务状态，最后下载或读取生成的视频内容。':
    'มักต้องสร้างงาน → poll สถานะ → ดาวน์โหลด/อ่านวิดีโอที่สร้าง แทนการได้ผลในคำขอเดียว',
  '该端点表示模型支持的一类具体 API 能力，完整请求地址由 BaseURL 加端点路径组成，适合需要手动填写接口地址的客户端、工作流工具或自建应用。':
    'endpoint นี้คือความสามารถ API ของโมเดล URL เต็ม = BaseURL + path เหมาะกับไคลเอนต์/workflow ที่ใส่ URL เอง',
  '说明：如果 WorkBuddy 添加自定义模型时选择了「高级配置」中的「自定义协议」，开启后将直接使用填写的接口地址，不再自动补全 /chat/completions 路径；这种情况下接口地址填写到 /v1 即可，例如：{{url}}。':
    'WorkBuddy: หากเปิดโปรโตคอลกำหนดเอง (ขั้นสูง) จะใช้ URL ที่ใส่โดยตรง ใส่แค่ /v1 เช่น {{url}}',
  '请求一般使用 POST，并在请求体里传入 model、messages、temperature、stream 等参数；支持流式输出时可用于打字机效果、长文本持续返回和实时助手场景。':
    'โดยทั่วไปใช้ POST พร้อม model, messages, temperature, stream ฯลฯ streaming ใช้ได้กับเอฟเฟกต์พิมพ์และผู้ช่วย real-time',
  '请求通常使用 POST，模型名称仍需填写通道路由模型名，API Key 使用本侧栏第三步复制的密钥。':
    'ใช้ POST ชื่อโมเดล routing + API Key จากขั้น 3 ของแผงนี้',
  '调用时通常需要同时配置 API Key 和模型名称；模型名称建议使用第二步中的通道路由模型名，以便请求固定到指定渠道。':
    'ตั้งค่า API Key และชื่อโมเดล ใช้ชื่อจากขั้น 2 เพื่อผูกช่องทาง',
  '这类接口常需要上传音频文件或指定 voice、input、format 等参数，部分工具会要求使用 multipart/form-data。':
    'อาจต้องอัปโหลดไฟล์เสียงหรือระบุ voice, input, format บางเครื่องมือต้องใช้ multipart/form-data',
  '音频端点通常用于语音转文字、文字转语音、翻译或音频理解，适合客服质检、会议纪要、语音助手和媒体处理工作流。':
    'STT, TTS, แปล, เข้าใจเสียง เหมาะกับ QA บันทึกประชุม ผู้ช่วยเสียง และ workflow สื่อ',
  '建议直连本站点发起请求，关闭系统代理或 VPN，避免代理干扰模型连接与请求转发。':
    'เชื่อมต่อไซต์นี้โดยตรง ปิด proxy/VPN ของระบบเพื่อไม่ให้รบกวนการเชื่อมต่อ',
  '要调用的模型名称。': 'ชื่อโมเดลที่จะเรียก',
  '组成对话的消息列表。每条消息通常包含 role 和 content 字段。':
    'รายการข้อความในการสนทนา แต่ละข้อความมักมี role และ content',
  '单条对话消息。': 'ข้อความสนทนาหนึ่งรายการ',
  '消息角色，例如 system、user 或 assistant。': 'บทบาทข้อความ เช่น system, user หรือ assistant',
  '消息内容。': 'เนื้อหาข้อความ',
  '是否使用流式响应。': 'ใช้การตอบกลับแบบ streaming หรือไม่',
  '响应 ID。': 'ID การตอบกลับ',
  '对象类型。': 'ประเภทออบเจ็กต์',
  '响应创建时间戳。': 'เวลาที่สร้างการตอบกลับ',
  '本次响应使用的模型名称。': 'ชื่อโมเดลที่ใช้ในการตอบกลับนี้',
  '模型生成结果列表。': 'รายการผลลัพธ์ที่โมเดลสร้าง',
  '单个生成结果。': 'ผลลัพธ์การสร้างหนึ่งรายการ',
  '结果索引。': 'ดัชนีผลลัพธ์',
  '模型返回的消息。': 'ข้อความที่โมเดลส่งกลับ',
  '消息角色。': 'บทบาทข้อความ',
  '生成结束原因。': 'เหตุผลที่การสร้างสิ้นสุด',
  'Token 使用量统计。': 'สถิติการใช้ token',
  '输入 Token 数。': 'จำนวน token อินพุต',
  '输出 Token 数。': 'จำนวน token เอาต์พุต',
  '总 Token 数。': 'token รวม',
};

const ms = {
  ...id,
  '复制可用于调用上述 API 端点的 API Key':
    'Salin Kunci API untuk memanggil endpoint API di atas',
  '复制带渠道路由的模型名，可将请求固定到指定渠道':
    'Salin nama model laluan saluran untuk mengunci permintaan ke saluran',
  聊天补全端点适合: 'Endpoint chat completions sesuai untuk',
  '按渠道展示稳定性、路由和当前价格':
    'Memaparkan kestabilan, laluan dan harga semasa mengikut saluran',
};

const toZhTW = (s) =>
  s
    .replace(/端点/g, '端點')
    .replace(/复制/g, '複製')
    .replace(/说明/g, '說明')
    .replace(/视频/g, '影片')
    .replace(/图像/g, '圖像')
    .replace(/图片/g, '圖片')
    .replace(/输入/g, '輸入')
    .replace(/输出/g, '輸出')
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
    .replace(/转发/g, '轉發')
    .replace(/端点/g, '端點')
    .replace(/聊天补全/g, '聊天補全')
    .replace(/渠道/g, '渠道')
    .replace(/稳定/g, '穩定')
    .replace(/调用/g, '呼叫')
    .replace(/填写/g, '填寫')
    .replace(/建议/g, '建議')
    .replace(/创建/g, '建立')
    .replace(/下载/g, '下載')
    .replace(/读取/g, '讀取')
    .replace(/配置/g, '設定')
    .replace(/上传/g, '上傳')
    .replace(/语音/g, '語音')
    .replace(/翻译/g, '翻譯')
    .replace(/理解/g, '理解')
    .replace(/适合/g, '適合')
    .replace(/工具/g, '工具')
    .replace(/编排/g, '編排')
    .replace(/自建/g, '自建')
    .replace(/工作流/g, '工作流')
    .replace(/向量/g, '向量')
    .replace(/检索/g, '檢索')
    .replace(/相似度/g, '相似度')
    .replace(/聚类/g, '聚類')
    .replace(/推荐/g, '推薦')
    .replace(/语义/g, '語意')
    .replace(/匹配/g, '匹配')
    .replace(/文生图/g, '文生圖')
    .replace(/图生图/g, '圖生圖')
    .replace(/编辑/g, '編輯')
    .replace(/分析/g, '分析')
    .replace(/绘图/g, '繪圖')
    .replace(/自动化/g, '自動化')
    .replace(/视觉/g, '視覺')
    .replace(/接入/g, '接入')
    .replace(/多数/g, '多數')
    .replace(/自动补全/g, '自動補全')
    .replace(/完整/g, '完整')
    .replace(/显示/g, '顯示')
    .replace(/明确/g, '明確')
    .replace(/优先/g, '優先')
    .replace(/老版/g, '舊版')
    .replace(/兼容/g, '相容')
    .replace(/仍/g, '仍')
    .replace(/使用/g, '使用')
    .replace(/知识库/g, '知識庫')
    .replace(/模型配置/g, '模型設定')
    .replace(/对应/g, '對應')
    .replace(/确认/g, '確認')
    .replace(/名称/g, '名稱')
    .replace(/一致/g, '一致')
    .replace(/提供/g, '提供')
    .replace(/输入框/g, '輸入框')
    .replace(/即可/g, '即可')
    .replace(/按工具/g, '按工具')
    .replace(/展示/g, '展示')
    .replace(/路由/g, '路由')
    .replace(/当前/g, '目前')
    .replace(/价格/g, '價格')
    .replace(/接入前/g, '接入前')
    .replace(/普通/g, '一般')
    .replace(/返回/g, '返回')
    .replace(/写入/g, '寫入')
    .replace(/数据库/g, '資料庫')
    .replace(/更适合/g, '更適合')
    .replace(/脚本/g, '腳本')
    .replace(/前端/g, '前端')
    .replace(/后端/g, '後端')
    .replace(/节点/g, '節點')
    .replace(/明确支持/g, '明確支援')
    .replace(/用于需要/g, '用於需要')
    .replace(/例如/g, '例如')
    .replace(/配合/g, '配合')
    .replace(/第二步/g, '第二步')
    .replace(/第三步/g, '第三步')
    .replace(/名字/g, '名稱')
    .replace(/一起/g, '一起')
    .replace(/仅支持/g, '僅支援')
    .replace(/请优先/g, '請優先')
    .replace(/服务地址/g, '服務位址')
    .replace(/也适合/g, '也適合')
    .replace(/自定义/g, '自訂')
    .replace(/生态/g, '生態')
    .replace(/最常用/g, '最常用')
    .replace(/通常用于/g, '通常用於')
    .replace(/常见/g, '常見')
    .replace(/流程/g, '流程')
    .replace(/提交/g, '提交')
    .replace(/尺寸/g, '尺寸')
    .replace(/时长/g, '時長')
    .replace(/参数/g, '參數')
    .replace(/轮询/g, '輪詢')
    .replace(/任务/g, '任務')
    .replace(/分镜/g, '分鏡')
    .replace(/任务型/g, '任務型')
    .replace(/处理/g, '處理')
    .replace(/创作/g, '創作')
    .replace(/生产/g, '生產')
    .replace(/异步/g, '非同步')
    .replace(/场景/g, '場景')
    .replace(/往往/g, '往往')
    .replace(/一次/g, '一次')
    .replace(/立即/g, '立即')
    .replace(/最终/g, '最終')
    .replace(/需要/g, '需要')
    .replace(/先/g, '先')
    .replace(/再/g, '再')
    .replace(/最后/g, '最後')
    .replace(/表示/g, '表示')
    .replace(/一类/g, '一類')
    .replace(/具体/g, '具體')
    .replace(/能力/g, '能力')
    .replace(/地址/g, '位址')
    .replace(/路径/g, '路徑')
    .replace(/组成/g, '組成')
    .replace(/手动/g, '手動')
    .replace(/添加/g, '新增')
    .replace(/高级/g, '進階')
    .replace(/协议/g, '協定')
    .replace(/开启/g, '開啟')
    .replace(/直接/g, '直接')
    .replace(/不再/g, '不再')
    .replace(/情况下/g, '情況下')
    .replace(/一般/g, '一般')
    .replace(/并在/g, '並在')
    .replace(/请求体/g, '請求主體')
    .replace(/传入/g, '傳入')
    .replace(/支持/g, '支援')
    .replace(/流式/g, '串流')
    .replace(/可用于/g, '可用於')
    .replace(/打字机/g, '打字機')
    .replace(/效果/g, '效果')
    .replace(/长文本/g, '長文本')
    .replace(/持续/g, '持續')
    .replace(/实时/g, '即時')
    .replace(/助手/g, '助手')
    .replace(/仍需/g, '仍需')
    .replace(/同时/g, '同時')
    .replace(/以便/g, '以便')
    .replace(/固定/g, '固定')
    .replace(/指定/g, '指定')
    .replace(/这类/g, '這類')
    .replace(/常需要/g, '常需要')
    .replace(/文件/g, '檔案')
    .replace(/指定/g, '指定')
    .replace(/部分/g, '部分')
    .replace(/会要求/g, '會要求')
    .replace(/转文字/g, '轉文字')
    .replace(/文字转/g, '文字轉')
    .replace(/质检/g, '質檢')
    .replace(/会议/g, '會議')
    .replace(/纪要/g, '紀要')
    .replace(/媒体/g, '媒體')
    .replace(/发起/g, '發起')
    .replace(/系统/g, '系統')
    .replace(/代理/g, '代理')
    .replace(/避免/g, '避免')
    .replace(/要调用/g, '要呼叫')
    .replace(/每条/g, '每條')
    .replace(/通常包含/g, '通常包含')
    .replace(/字段/g, '欄位')
    .replace(/单条/g, '單條')
    .replace(/对话/g, '對話')
    .replace(/是否/g, '是否')
    .replace(/使用量/g, '使用量')
    .replace(/数/g, '數');

const zhTW = Object.fromEntries(
  usageKeys.map((key) => [key, toZhTW(key)]),
);

const localeMaps = { ja, fr, ru, vi, id, ms, th, sw: id, 'zh-TW': zhTW };

// 读取 en 作为 sw 回退（比印尼语更合适）
const enFile = JSON.parse(
  fs.readFileSync(path.join(localesDir, 'en.json'), 'utf8'),
).translation;

for (const [lang, map] of Object.entries(localeMaps)) {
  const filePath = path.join(localesDir, `${lang}.json`);
  const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  let updated = 0;
  const effectiveMap = lang === 'sw'
    ? Object.fromEntries(
        usageKeys.filter((k) => enFile[k]).map((k) => [k, enFile[k]]),
      )
    : map;
  for (const key of usageKeys) {
    if (effectiveMap[key]) {
      data.translation[key] = effectiveMap[key];
      updated++;
    }
  }
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + '\n');
  console.log(`${lang}.json: updated ${updated} usage keys`);
}

const extraLabels = {
  ja: { 端点说明: 'エンドポイント説明', 用途说明: '用途説明' },
  fr: { 端点说明: "Notes sur le point d'accès", 用途说明: "Notes d'utilisation" },
  ru: { 端点说明: 'Описание endpoint', 用途说明: 'Описание использования' },
  vi: { 端点说明: 'Mô tả endpoint', 用途说明: 'Hướng dẫn sử dụng' },
  id: { 端点说明: 'Catatan endpoint', 用途说明: 'Catatan penggunaan' },
  ms: { 端点说明: 'Nota endpoint', 用途说明: 'Nota penggunaan' },
  th: { 端点说明: 'คำอธิบาย endpoint', 用途说明: 'คำอธิบายการใช้งาน' },
  sw: { 端点说明: 'Maelezo ya endpoint', 用途说明: 'Maelezo ya matumizi' },
  'zh-TW': { 端点说明: '端點說明', 用途说明: '用途說明' },
};

for (const [lang, labels] of Object.entries(extraLabels)) {
  const filePath = path.join(localesDir, `${lang}.json`);
  const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
  for (const [key, value] of Object.entries(labels)) {
    data.translation[key] = value;
  }
  fs.writeFileSync(filePath, JSON.stringify(data, null, 2) + '\n');
  console.log(`${lang}.json: patched ${Object.keys(labels).length} label keys`);
}
