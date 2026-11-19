import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { buildEmbeddedUrl, detectTheme } from '../embedded-url'

describe('embedded-url', () => {
  const originalLocation = window.location

  beforeEach(() => {
    Object.defineProperty(window, 'location', {
      value: {
        origin: 'https://app.example.com',
        href: 'https://app.example.com/user/purchase',
      },
      writable: true,
      configurable: true,
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
      configurable: true,
    })
    document.documentElement.classList.remove('dark')
    vi.restoreAllMocks()
  })

  it('adds embedded query parameters without identity or source context', () => {
    const result = buildEmbeddedUrl(
      'https://pay.example.com/checkout?plan=pro',
      'dark',
      'zh-CN',
    )

    const url = new URL(result)
    expect(url.searchParams.get('plan')).toBe('pro')
    expect(url.searchParams.has('user_id')).toBe(false)
    expect(url.searchParams.has('token')).toBe(false)
    expect(url.searchParams.get('theme')).toBe('dark')
    expect(url.searchParams.get('lang')).toBe('zh-CN')
    expect(url.searchParams.get('ui_mode')).toBe('embedded')
    expect(url.searchParams.has('src_host')).toBe(false)
    expect(url.searchParams.has('src_url')).toBe(false)
  })

  it('removes saved credentials, identifiers and fragments', () => {
    const url = new URL(buildEmbeddedUrl('https://example.com/?token=secret&access_token=secret&refresh_token=secret&user_id=4&src_url=secret&src_host=secret#secret'))
    for (const key of ['token', 'access_token', 'refresh_token', 'user_id', 'src_url', 'src_host']) {
      expect(url.searchParams.has(key)).toBe(false)
    }
    expect(url.hash).toBe('')
    expect(url.toString()).not.toContain('secret')
  })

  it.each(['javascript:alert(1)', 'data:text/html,secret', 'https://user:secret@example.com/'])('rejects unsafe URL %s', (url) => {
    expect(buildEmbeddedUrl(url)).toBe('')
  })

  it('omits optional params when they are empty', () => {
    const result = buildEmbeddedUrl('https://pay.example.com/checkout', 'light')

    const url = new URL(result)
    expect(url.searchParams.get('theme')).toBe('light')
    expect(url.searchParams.get('ui_mode')).toBe('embedded')
    expect(url.searchParams.has('user_id')).toBe(false)
    expect(url.searchParams.has('token')).toBe(false)
    expect(url.searchParams.has('lang')).toBe(false)
  })

  it('rejects invalid URL input', () => {
    expect(buildEmbeddedUrl('not a url')).toBe('')
  })

  it('detects dark mode from document root class', () => {
    document.documentElement.classList.add('dark')
    expect(detectTheme()).toBe('dark')
  })
})
