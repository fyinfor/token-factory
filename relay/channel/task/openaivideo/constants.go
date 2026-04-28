package openaivideo

// ModelList enumerates the known video model IDs exposed through the OpenAI
// Video gateway (currently sourced from the Hidream/Seedance upstream).
//
// Upstream model IDs are usually opaque hashes (e.g. Video-a4lzrja7); the
// human-readable aliases (Seedance2.0 / Seedance2.0-fast) are kept so users
// can route them through model_mapping if they prefer the friendly name.
var ModelList = []string{
	"Seedance2.0",
	"Seedance2.0-fast",
	"Video-a4lzrja7",
}

var ChannelName = "openai-video"
