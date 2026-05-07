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
	// EndpointTypeVideoGenerator points to providers exposing
	// /videogenerator/generate style APIs.
	EndpointTypeVideoGenerator EndpointType = "videogenerator"
	// EndpointTypeTencentCloudVODVideo is Tencent Cloud VOD AIGC video (TC3 API).
	// Client body matches OpenAI-videogenerator-style gateway fields; upstream uses JSON API 3.0.
	EndpointTypeTencentCloudVODVideo EndpointType = "tencentcloud-vod-video"
	// EndpointTypeTencentCloudVODImage is Tencent Cloud VOD AIGC image (TC3 API).
	EndpointTypeTencentCloudVODImage EndpointType = "tencentcloud-vod-image"
	//EndpointTypeMidjourney     EndpointType = "midjourney-proxy"
	//EndpointTypeSuno           EndpointType = "suno-proxy"
	//EndpointTypeKling          EndpointType = "kling"
	//EndpointTypeJimeng         EndpointType = "jimeng"
)
