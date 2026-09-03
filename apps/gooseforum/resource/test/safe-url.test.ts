import { describe, expect, test } from 'vitest'
import { safeUrl } from '../src/runtime/safe-url'

describe('safeUrl site-link policy', () => {
  test('keeps relative site paths and http(s)', () => {
    expect(safeUrl('/sponsors', 'site-link')).toBe('/sponsors')
    expect(safeUrl('sponsors', 'site-link')).toBe('sponsors')
    expect(safeUrl('/wiki/a?b=c#frag', 'site-link')).toBe('/wiki/a?b=c#frag')
    expect(safeUrl('https://example.com/path?q=1', 'site-link')).toBe('https://example.com/path?q=1')
    expect(safeUrl('http://example.com', 'site-link')).toBe('http://example.com')
  })

  test('empty and whitespace degrade to empty', () => {
    expect(safeUrl('', 'site-link')).toBe('')
    expect(safeUrl('   ', 'site-link')).toBe('')
    expect(safeUrl(undefined, 'site-link')).toBe('')
    expect(safeUrl(null, 'site-link')).toBe('')
  })

  test('trimmed values are returned trimmed', () => {
    expect(safeUrl('  /sponsors  ', 'site-link')).toBe('/sponsors')
  })

  test('rejects dangerous schemes, protocol-relative and obfuscation', () => {
    const dangerous = [
      'javascript:alert(1)',
      'JaVaScRiPt:alert(1)',
      'data:text/html;base64,PHNjcmlwdD4=',
      'vbscript:msgbox(1)',
      'file:///etc/passwd',
      '//evil.example.com/path',
      '///evil.example.com/path',
      'java\nscript:alert(1)',
      'java\tscript:alert(1)',
      'java\rscript:alert(1)',
      'jav\u0000ascript:alert(1)',
      'jav&#x61;script:alert(1)',
      'jav&#97;script:alert(1)',
      'javascript&#58;alert(1)',
      'java&#x09;script:alert(1)',
      '&#106;&#97;&#118;&#97;&#115;&#99;&#114;&#105;&#112;&#116;&#58;alert(1)',
      'https://',
      'http://',
    ]
    for (const raw of dangerous) {
      expect(safeUrl(raw, 'site-link'), raw).toBe('')
    }
  })
})

describe('safeUrl external policy', () => {
  test('keeps absolute http(s) only', () => {
    expect(safeUrl('https://example.com', 'external')).toBe('https://example.com')
    expect(safeUrl('http://example.com/path', 'external')).toBe('http://example.com/path')
  })

  test('rejects relative paths, mailto and dangerous schemes', () => {
    for (const raw of ['/sponsors', 'sponsors', '//example.com', 'mailto:a@example.com', 'javascript:alert(1)', 'ftp://example.com']) {
      expect(safeUrl(raw, 'external'), raw).toBe('')
    }
  })
})

describe('safeUrl image and contact policies', () => {
  test('image keeps relative and http(s)', () => {
    expect(safeUrl('/static/pic/logo.webp', 'image')).toBe('/static/pic/logo.webp')
    expect(safeUrl('https://cdn.example.com/a.png', 'image')).toBe('https://cdn.example.com/a.png')
    expect(safeUrl('//cdn.example.com/a.png', 'image')).toBe('')
    expect(safeUrl('file:///tmp/a.png', 'image')).toBe('')
  })

  test('contact additionally allows mailto', () => {
    expect(safeUrl('mailto:contact@example.com', 'contact')).toBe('mailto:contact@example.com')
    expect(safeUrl('https://example.com/contact', 'contact')).toBe('https://example.com/contact')
    expect(safeUrl('mailto:', 'contact')).toBe('')
    expect(safeUrl('javascript:alert(1)', 'contact')).toBe('')
  })
})

describe('safeUrl length guard', () => {
  test('overlong values degrade to empty', () => {
    const long = `https://example.com/${'a'.repeat(3000)}`
    expect(safeUrl(long, 'site-link')).toBe('')
    expect(safeUrl(long, 'external')).toBe('')
  })
})
