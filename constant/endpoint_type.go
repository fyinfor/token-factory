package constant

type EndpointType string

const (
	EndpointTypeOpenAI                EndpointType = "openai"
	EndpointTypeOpenAIResponse        EndpointType = "openai-response"
	EndpointTypeOpenAIResponseCompact EndpointType = "openai-response-compact"
	EndpointTypeAnthropic             EndpointType = "anthropic"
	EndpointTypeGemini                EndpointType = "gemini"
	EndpointTypeJinaRerank            EndpointType = "jina-rerank"
	EndpointTypeImageGeneration       EndpointType = "image-generation"
	EndpointTypeEmbeddings            EndpointType = "embeddings"
	EndpointTypeOpenAIVideo           EndpointType = "openai-video"
	// EndpointTypeOpenAIVideoGW points to the OpenAI-compatible video gateway
	// (currently Hidream/Seedance MaaS or ARK-compatible upstream). The value
	// "hidream-video" is kept as-is for backward compatibility with existing
	// channel/endpoint configurations stored in the database.
	EndpointTypeOpenAIVideoGW EndpointType = "hidream-video"
	// EndpointTypeTokenFactoryVideo is the unified task video entry on TokenFactory
	// (POST /v1/video/generations). Use this when testing TokenFactoryOpen (60) channels
	// against an upstream TokenFactory instance — not the external Hidream /v1/videos/generations path.
	EndpointTypeTokenFactoryVideo EndpointType = "tokenfactory-video"
	// EndpointTypeVideoGenerator points to providers exposing
	// /video/generations style APIs (formerly /videogenerator/generate).
	EndpointTypeVideoGenerator EndpointType = "videogenerator"
	// EndpointTypeTencentCloudVODVideo is Tencent Cloud VOD AIGC video (TC3 API).
	// Client body matches OpenAI-videogenerator-style gateway fields; upstream uses JSON API 3.0.
	EndpointTypeTencentCloudVODVideo EndpointType = "tencentcloud-vod-video"
	// EndpointTypeTencentCloudVODImage is Tencent Cloud VOD AIGC image (TC3 API).
	EndpointTypeTencentCloudVODImage EndpointType = "tencentcloud-vod-image"
	// EndpointTypeAliVideo is Alibaba DashScope video-synthesis (async task API).
	EndpointTypeAliVideo EndpointType = "ali-video"
	// EndpointTypeSeedanceVideo is VolcEngine contents generations (Seedance 2.0 async API).
	// Client entry: POST /v1/video/generations 或 POST /api/v3/contents/generations/tasks;
	// upstream: POST /api/v3/contents/generations/tasks.
	EndpointTypeSeedanceVideo EndpointType = "seedance-video"
	// EndpointTypeMiniMaxH3Video is MiniMax Hailuo-03 / H3 video generation V2.
	// Client entry: POST /v1/video/generations; upstream: POST {baseUrl}/video_generation
	// where baseUrl is typically https://api.minimaxi.com/v2.
	EndpointTypeMiniMaxH3Video EndpointType = "minimax-h3-video"
	// EndpointTypeAliASRSync is Alibaba DashScope ASR sync transcription.
	// Client entry: POST /v1/audio/transcriptions.
	// AliASRSync channel: upstream multimodal-generation.
	// AliASRAsync channel: same client path, internally async submit + poll then return the result.
	EndpointTypeAliASRSync EndpointType = "ali-asr-sync"
	// EndpointTypeAliASRAsync is Alibaba DashScope ASR async file transcription.
	// Client entry: POST /v1/audio/transcriptions/async; upstream: asr/transcription + tasks poll.
	EndpointTypeAliASRAsync EndpointType = "ali-asr-async"
	//EndpointTypeMidjourney     EndpointType = "midjourney-proxy"
	//EndpointTypeSuno           EndpointType = "suno-proxy"
	//EndpointTypeKling          EndpointType = "kling"
	//EndpointTypeJimeng         EndpointType = "jimeng"
)
