import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

const here = dirname(fileURLToPath(import.meta.url))
const read = (path: string) => readFileSync(resolve(here, path), 'utf8')

describe('Prompt Audit integration surface', () => {
  it('keeps content-retaining admin pages out of the personal deployment router', () => {
    const router = read('../../../router/index.ts')
    for (const path of ['/admin/prompt-audit', '/admin/risk-control', '/admin/plugins']) {
      expect(router).not.toContain(`path: '${path}'`)
    }
  })

  it('keeps content-retaining features out of the personal deployment sidebar', () => {
    const sidebar = read('../../../components/layout/AppSidebar.vue')
    for (const path of ['/admin/security-audit', '/admin/risk-control', '/admin/prompt-audit', '/admin/plugins']) {
      expect(sidebar).not.toContain(`path: '${path}'`)
    }
  })

  it('keeps Prompt Audit locale trees symmetric and all operational controls named', () => {
    expect(Object.keys(zh.admin.promptAudit)).toEqual(Object.keys(en.admin.promptAudit))
    expect(zh.nav.securityAudit).toBeTruthy()
    expect(en.nav.securityAudit).toBeTruthy()
    const endpoint = read('../components/EndpointPool.vue')
    const events = read('../components/EventWorkspace.vue')
    expect(endpoint).toContain('aria-label')
    expect(events).toContain('aria-label')
    expect(events).toContain('overflow-x-auto')
    expect(events).toContain('sm:grid-cols-2')
  })
})
