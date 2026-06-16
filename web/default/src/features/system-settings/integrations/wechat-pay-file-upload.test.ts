import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { getWechatPemFileKind } from './wechat-pay-file-upload'

describe('getWechatPemFileKind', () => {
  test('detects apiclient_cert.pem as certificate', () => {
    assert.equal(getWechatPemFileKind('apiclient_cert.pem'), 'cert')
  })

  test('detects apiclient_key.pem as private key', () => {
    assert.equal(getWechatPemFileKind('apiclient_key.pem'), 'key')
  })

  test('rejects unrelated pem files', () => {
    assert.equal(getWechatPemFileKind('other.pem'), null)
  })
})
