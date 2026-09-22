/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
const ICON_VENDOR_NAMES: Record<string, string> = {
  OpenAI: 'OpenAI',
  Claude: 'Anthropic',
  Anthropic: 'Anthropic',
  Gemini: 'Google',
  Google: 'Google',
  Qwen: 'Alibaba',
  Alibaba: 'Alibaba',
  Doubao: 'ByteDance',
  ByteDance: 'ByteDance',
  DeepSeek: 'DeepSeek',
  Zhipu: 'Zhipu',
  Moonshot: 'Moonshot',
  Hunyuan: 'Tencent',
  Kling: 'Kling',
  Jimeng: 'Jimeng',
  Vidu: 'Vidu',
  Midjourney: 'Midjourney',
  Mistral: 'Mistral',
  Cohere: 'Cohere',
  Jina: 'Jina',
  AWS: 'AWS',
  Baidu: 'Baidu',
  Cloudflare: 'Cloudflare',
  Minimax: 'MiniMax',
  MiniMax: 'MiniMax',
  Spark: 'iFlytek',
  Tencent: 'Tencent',
  XAI: 'xAI',
}

const MODEL_VENDOR_PATTERNS: Array<[string, string]> = Object.entries({
  'text-embedding': 'OpenAI',
  'dall-e': 'OpenAI',
  chatgpt: 'OpenAI',
  whisper: 'OpenAI',
  claude: 'Anthropic',
  deepseek: 'DeepSeek',
  seedance: 'ByteDance',
  seedream: 'ByteDance',
  doubao: 'ByteDance',
  gemini: 'Google',
  gemma: 'Google',
  qwen: 'Alibaba',
  tongyi: 'Alibaba',
  moonshot: 'Moonshot',
  kimi: 'Moonshot',
  hunyuan: 'Tencent',
  minimax: 'MiniMax',
  mistral: 'Mistral',
  mixtral: 'Mistral',
  cohere: 'Cohere',
  command: 'Cohere',
  perplexity: 'Perplexity',
  sonar: 'Perplexity',
  midjourney: 'Midjourney',
  kling: 'Kling',
  jimeng: 'Jimeng',
  vidu: 'Vidu',
  baichuan: 'Baichuan',
  zhipu: 'Zhipu',
  chatglm: 'Zhipu',
  cogview: 'Zhipu',
  llama: 'Meta',
  'bge-': 'BAAI',
  'jina-': 'Jina',
  grok: 'xAI',
  sora: 'OpenAI',
  'gpt-': 'OpenAI',
  'tts-': 'OpenAI',
}).sort(([left], [right]) => right.length - left.length)

export function inferVendorName(input: {
  icon?: string
  modelName?: string
}): string | undefined {
  const iconName = input.icon?.split('.')[0]
  if (iconName && ICON_VENDOR_NAMES[iconName]) {
    return ICON_VENDOR_NAMES[iconName]
  }

  const modelName = input.modelName?.trim().toLowerCase()
  if (!modelName) return undefined

  return MODEL_VENDOR_PATTERNS.find(([pattern]) =>
    modelName.includes(pattern)
  )?.[1]
}
