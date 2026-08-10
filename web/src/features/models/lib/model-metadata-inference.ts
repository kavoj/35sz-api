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
export type InferredModelMetadata = {
  tags: string[]
  contextLength?: number
}

const CONTEXT_RULES: Array<[RegExp, number]> = [
  [/(?:^|[-_])1m(?:[-_]|$)|gemini-(?:1\.5|2|3)/i, 1_000_000],
  [/claude-(?:3|4)/i, 200_000],
  [/gpt-4o|gpt-4\.1|gpt-5|(?:^|[-_])o[134](?:[-_]|$)/i, 128_000],
  [/(?:^|[-_])128k(?:[-_]|$)/i, 128_000],
  [/(?:^|[-_])64k(?:[-_]|$)/i, 65_536],
  [/(?:^|[-_])32k(?:[-_]|$)/i, 32_768],
  [/(?:^|[-_])16k(?:[-_]|$)/i, 16_384],
]

export function inferModelMetadata(modelName: string): InferredModelMetadata {
  const name = modelName.trim().toLowerCase()
  if (!name) return { tags: [] }

  let tags: string[]
  if (
    /seedream|dall[-_]?e|gpt-image|flux|midjourney|stable-diffusion/.test(name)
  ) {
    tags = ['image']
  } else if (/seedance|sora|kling|veo|(?:^|[-_])i2v|(?:^|[-_])t2v/.test(name)) {
    tags = ['video']
  } else if (
    /seed[-_]?asr|whisper|transcri|(?:^|[-_])asr(?:[-_]|$)/.test(name)
  ) {
    tags = ['audio', 'asr']
  } else if (/^tts(?:[-_]|$)|text[-_]?to[-_]?speech|seed[-_]?tts/.test(name)) {
    tags = ['audio', 'tts']
  } else if (/embedding|embed(?:[-_]|$)|rerank|(?:^|[-_])bge[-_]/.test(name)) {
    tags = ['embedding']
  } else {
    tags = ['chat']
    if (/gpt-4o|vision|(?:^|[-_])vl(?:[-_]|$)|glm-4v|multimodal/.test(name)) {
      tags.push('vision')
    }
    if (
      /deepseek-r\d|reasoning|thinking|(?:^|[-_])o[134](?:[-_]|$)|qwq/.test(
        name
      )
    ) {
      tags.push('reasoning')
    } else if (
      /gpt-4|gpt-5|claude-(?:3|4)|gemini-(?:1\.5|2|3)|qwen3/.test(name)
    ) {
      tags.push('function-calling')
    }
    if (/coder|codestral|code[-_]?llama/.test(name)) tags.push('code')
  }

  return {
    tags,
    contextLength: CONTEXT_RULES.find(([pattern]) => pattern.test(name))?.[1],
  }
}
