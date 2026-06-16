export type WechatPemFileKind = 'cert' | 'key'

export function getWechatPemFileKind(fileName: string): WechatPemFileKind | null {
  const normalized = fileName.trim().toLowerCase()
  if (normalized === 'apiclient_cert.pem') return 'cert'
  if (normalized === 'apiclient_key.pem') return 'key'
  return null
}
