import { describe, expect, test } from 'vitest'
import {
  SUPPORTED_UPLOAD_EXTENSIONS,
  isSupportedUploadExtension,
  normalizeExtensionToken,
} from '../src/admin/uploadExtensions'

describe('upload extension allowlist helpers (issue #408)', () => {
  test('canonical set covers the six supported image types', () => {
    expect(SUPPORTED_UPLOAD_EXTENSIONS).toEqual(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp'])
  })

  test('normalizeExtensionToken tolerates dotless and upper-case input', () => {
    expect(normalizeExtensionToken('png')).toBe('.png')
    expect(normalizeExtensionToken('.PNG')).toBe('.png')
    expect(normalizeExtensionToken('  .JPG  ')).toBe('.jpg')
    expect(normalizeExtensionToken('')).toBe('')
  })

  test('isSupportedUploadExtension accepts canonical variants only', () => {
    for (const ext of ['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.JPG', 'png']) {
      expect(isSupportedUploadExtension(ext)).toBe(true)
    }
    for (const ext of ['.svg', '.html', '.htm', '.xml', '.js', '.css', '.json', '.pdf', 'avatar.png.exe', '']) {
      expect(isSupportedUploadExtension(ext)).toBe(false)
    }
  })
})
