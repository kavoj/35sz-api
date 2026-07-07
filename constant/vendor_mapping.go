package constant

import "strings"

// VendorSpec 描述一个供应商的初始信息，用于新建/幂等 upsert。
// Name 唯一，作为主键；DisplayName 面向 UI；Icon 使用 @lobehub/icons 的 key。
type VendorSpec struct {
	Name        string
	DisplayName string
	Icon        string
}

// ChannelTypeToVendor 由 channel type 反查 vendor 规格。
// 与前端 web/default/src/features/channels/lib/channel-utils.ts:getChannelTypeIcon 保持同步。
// 语义：这个渠道所服务的模型，其"原始供应商"是谁。
var ChannelTypeToVendor = map[int]VendorSpec{
	ChannelTypeOpenAI:         {Name: "OpenAI", DisplayName: "OpenAI", Icon: "OpenAI.Color"},
	ChannelTypeAzure:          {Name: "Azure", DisplayName: "Azure OpenAI", Icon: "Azure.Color"},
	ChannelTypeOhMyGPT:        {Name: "OpenAI", DisplayName: "OpenAI", Icon: "OpenAI.Color"},
	ChannelTypeCustom:         {Name: "OpenAI", DisplayName: "OpenAI", Icon: "OpenAI.Color"},
	ChannelTypeAdvancedCustom: {Name: "OpenAI", DisplayName: "OpenAI", Icon: "OpenAI.Color"},
	ChannelTypeAnthropic:      {Name: "Anthropic", DisplayName: "Anthropic", Icon: "Claude.Color"},
	ChannelTypeGemini:         {Name: "Google", DisplayName: "Google", Icon: "Gemini.Color"},
	ChannelTypePaLM:           {Name: "Google", DisplayName: "Google", Icon: "Google.Color"},
	ChannelTypeVertexAi:       {Name: "Google", DisplayName: "Google", Icon: "Gemini.Color"},
	ChannelTypeAws:            {Name: "AWS", DisplayName: "AWS Bedrock", Icon: "Aws.Color"},
	ChannelCloudflare:         {Name: "Cloudflare", DisplayName: "Cloudflare", Icon: "Cloudflare.Color"},
	ChannelTypeBaidu:          {Name: "Baidu", DisplayName: "百度 / Baidu", Icon: "Baidu.Color"},
	ChannelTypeBaiduV2:        {Name: "Baidu", DisplayName: "百度 / Baidu", Icon: "Baidu.Color"},
	ChannelTypeZhipu:          {Name: "Zhipu", DisplayName: "智谱 AI / Zhipu", Icon: "Zhipu.Color"},
	ChannelTypeZhipu_v4:       {Name: "Zhipu", DisplayName: "智谱 AI / Zhipu", Icon: "Zhipu.Color"},
	ChannelTypeAli:            {Name: "Alibaba", DisplayName: "阿里 / Qwen", Icon: "Qwen.Color"},
	ChannelTypeXunfei:         {Name: "iFlytek", DisplayName: "讯飞 / Spark", Icon: "Spark.Color"},
	ChannelTypeTencent:        {Name: "Tencent", DisplayName: "腾讯 / Hunyuan", Icon: "Hunyuan.Color"},
	ChannelType360:            {Name: "Ai360", DisplayName: "360 智脑", Icon: "Ai360.Color"},
	ChannelTypeMoonshot:       {Name: "Moonshot", DisplayName: "月之暗面 / Kimi", Icon: "Moonshot.Color"},
	ChannelTypeLingYiWanWu:    {Name: "LingYiWanWu", DisplayName: "零一万物 / Yi", Icon: "Yi.Color"},
	ChannelTypeMiniMax:        {Name: "MiniMax", DisplayName: "MiniMax", Icon: "Minimax.Color"},
	ChannelTypeVolcEngine:     {Name: "ByteDance", DisplayName: "字节跳动 / Doubao", Icon: "Doubao.Color"},
	ChannelTypeDoubaoVideo:    {Name: "ByteDance", DisplayName: "字节跳动 / Doubao", Icon: "Doubao.Color"},
	ChannelTypeOllama:         {Name: "Ollama", DisplayName: "Ollama", Icon: "Ollama.Color"},
	ChannelTypePerplexity:     {Name: "Perplexity", DisplayName: "Perplexity", Icon: "Perplexity.Color"},
	ChannelTypeCohere:         {Name: "Cohere", DisplayName: "Cohere", Icon: "Cohere.Color"},
	ChannelTypeMistral:        {Name: "Mistral", DisplayName: "Mistral", Icon: "Mistral.Color"},
	ChannelTypeDeepSeek:       {Name: "DeepSeek", DisplayName: "DeepSeek", Icon: "DeepSeek.Color"},
	ChannelTypeXai:            {Name: "xAI", DisplayName: "xAI", Icon: "XAI.Color"},
	ChannelTypeCoze:           {Name: "Coze", DisplayName: "Coze", Icon: "Coze.Color"},
	ChannelTypeSiliconFlow:    {Name: "SiliconFlow", DisplayName: "SiliconFlow", Icon: "SiliconCloud.Color"},
	ChannelTypeOpenRouter:     {Name: "OpenRouter", DisplayName: "OpenRouter", Icon: "OpenRouter.Color"},
	ChannelTypeMidjourney:     {Name: "Midjourney", DisplayName: "Midjourney", Icon: "Midjourney.Color"},
	ChannelTypeMidjourneyPlus: {Name: "Midjourney", DisplayName: "Midjourney", Icon: "Midjourney.Color"},
	ChannelTypeKling:          {Name: "Kling", DisplayName: "可灵 / Kling", Icon: "Kling.Color"},
	ChannelTypeJimeng:         {Name: "Jimeng", DisplayName: "即梦 / Jimeng", Icon: "Jimeng.Color"},
	ChannelTypeVidu:           {Name: "Vidu", DisplayName: "Vidu", Icon: "Vidu.Color"},
	ChannelTypeSunoAPI:        {Name: "Suno", DisplayName: "Suno", Icon: "Suno.Color"},
	ChannelTypeSora:           {Name: "OpenAI", DisplayName: "OpenAI", Icon: "OpenAI.Color"},
	ChannelTypeReplicate:      {Name: "Replicate", DisplayName: "Replicate", Icon: "Replicate.Color"},
	ChannelTypeDify:           {Name: "Dify", DisplayName: "Dify", Icon: "Dify.Color"},
	ChannelTypeJina:           {Name: "Jina", DisplayName: "Jina", Icon: "Jina.Color"},
	ChannelTypeFastGPT:        {Name: "FastGPT", DisplayName: "FastGPT", Icon: "FastGPT.Color"},
	ChannelTypeXinference:     {Name: "Xinference", DisplayName: "Xinference", Icon: "Xinference.Color"},
	ChannelTypeCodex:          {Name: "OpenAI", DisplayName: "OpenAI", Icon: "OpenAI.Color"},
}

// iconToVendor 由 Lobe icon key 反查 vendor 规格。
// 派生自 ChannelTypeToVendor，保持"渠道→图标→供应商"三者一致的单一数据源。
// 同一个 icon 可能对应多个 channel type（如 OpenAI 兼容渠道），此处取任意其一即可，
// 因为其 vendor 相同。
var iconToVendor = func() map[string]VendorSpec {
	m := make(map[string]VendorSpec, len(ChannelTypeToVendor))
	for _, spec := range ChannelTypeToVendor {
		if spec.Icon == "" {
			continue
		}
		if _, ok := m[spec.Icon]; !ok {
			m[spec.Icon] = spec
		}
	}
	return m
}()

// LookupVendorByIcon 根据 icon key 返回 vendor 规格。
// 支持大小写差异与去掉 ".Color" 后缀的粗匹配。
func LookupVendorByIcon(icon string) (VendorSpec, bool) {
	if icon == "" {
		return VendorSpec{}, false
	}
	if v, ok := iconToVendor[icon]; ok {
		return v, true
	}
	// 尝试去掉 .Color / .Avatar 后缀，然后再补上 .Color 匹配
	base := icon
	if i := strings.Index(base, "."); i > 0 {
		base = base[:i]
	}
	if v, ok := iconToVendor[base+".Color"]; ok {
		return v, true
	}
	return VendorSpec{}, false
}

// LookupVendorByChannelType 根据 channel type 返回 vendor 规格。
func LookupVendorByChannelType(channelType int) (VendorSpec, bool) {
	v, ok := ChannelTypeToVendor[channelType]
	return v, ok
}

// namePatternToVendor 模型名前缀/子串 → vendor 名。
// 顺序不敏感；命中越具体越好，通过 InferVendorNameByModelName 按串长降序遍历以减少歧义。
var namePatternToVendor = map[string]string{
	"gpt":            "OpenAI",
	"dall-e":         "OpenAI",
	"text-davinci":   "OpenAI",
	"text-curie":     "OpenAI",
	"text-babbage":   "OpenAI",
	"text-ada":       "OpenAI",
	"o1":             "OpenAI",
	"o3":             "OpenAI",
	"o4":             "OpenAI",
	"sora":           "OpenAI",
	"whisper":        "OpenAI",
	"tts":            "OpenAI",
	"claude":         "Anthropic",
	"gemini":         "Google",
	"gemma":          "Google",
	"palm":           "Google",
	"text-bison":     "Google",
	"chat-bison":     "Google",
	"deepseek":       "DeepSeek",
	"glm":            "Zhipu",
	"chatglm":        "Zhipu",
	"cogview":        "Zhipu",
	"zhipu":          "Zhipu",
	"moonshot":       "Moonshot",
	"kimi":           "Moonshot",
	"qwen":           "Alibaba",
	"tongyi":         "Alibaba",
	"doubao":         "ByteDance",
	"seed":           "ByteDance",
	"seedance":       "ByteDance",
	"seedream":       "ByteDance",
	"hunyuan":        "Tencent",
	"baichuan":       "Baichuan",
	"llama":          "Meta",
	"mistral":        "Mistral",
	"mixtral":        "Mistral",
	"minimax":        "MiniMax",
	"abab":           "MiniMax",
	"yi":             "LingYiWanWu",
	"spark":          "iFlytek",
	"ernie":          "Baidu",
	"wenxin":         "Baidu",
	"grok":           "xAI",
	"cohere":         "Cohere",
	"command":        "Cohere",
	"perplexity":     "Perplexity",
	"sonar":          "Perplexity",
	"midjourney":     "Midjourney",
	"suno":           "Suno",
	"kling":          "Kling",
	"jimeng":         "Jimeng",
	"vidu":           "Vidu",
	"replicate":      "Replicate",
}

// InferVendorNameByModelName 从模型名推断 vendor.Name。
// 匹配策略：先按 pattern 长度降序（更具体的优先），子串命中即返回。
// 若无命中返回空串。
func InferVendorNameByModelName(modelName string) string {
	if modelName == "" {
		return ""
	}
	name := strings.ToLower(modelName)
	// 排序无法在包级 var 里预排；此处线性扫描并挑最长命中即可
	var best string
	var bestLen int
	for pattern, vendor := range namePatternToVendor {
		if strings.Contains(name, pattern) && len(pattern) > bestLen {
			best = vendor
			bestLen = len(pattern)
		}
	}
	return best
}
