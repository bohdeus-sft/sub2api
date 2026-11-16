import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import AntigravityQuotaUsage from '../AntigravityQuotaUsage.vue'
import UsageProgressBar from '../UsageProgressBar.vue'
import type { AntigravityQuotaBucket } from '@/types'

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key })
}))
enableAutoUnmount(afterEach)

function bucket(id: string, window: string, utilization: number | null, overrides: Partial<AntigravityQuotaBucket> = {}): AntigravityQuotaBucket {
  return { id, window, utilization, display_name: '', reset_time: '', disabled: false, ...overrides }
}

describe('AntigravityQuotaUsage', () => {
  it('shows the four shared limits directly with remaining capacity and resets', () => {
    const wrapper = mount(AntigravityQuotaUsage, { props: { usage: { antigravity_quota_groups: [
      { id: 'gemini', display_name: 'Gemini Models', buckets: [
        bucket('gemini-5h', '5h', 25, { reset_time: '2026-11-17T15:00:00Z' }),
        bucket('gemini-weekly', 'weekly', 90)
      ] },
      { id: '3p', display_name: 'Claude and GPT models', buckets: [
        bucket('3p-5h', '5h', 100), bucket('3p-weekly', 'weekly', 0)
      ] }
    ] } } })
    const bars = wrapper.findAllComponents(UsageProgressBar)
    expect(bars).toHaveLength(4)
    expect(bars.map(bar => bar.props('utilization'))).toEqual([75, 10, 0, 100])
    expect(bars.every(bar => bar.props('remainingCapacity'))).toBe(true)
    expect(bars[0].props('resetsAt')).toBe('2026-11-17T15:00:00Z')
    expect(wrapper.text()).toContain('Gemini Models')
    expect(wrapper.text()).toContain('Claude and GPT models')
    expect(wrapper.find('details').exists()).toBe(false)
  })

  it('preserves unknown and disabled limits', () => {
    const wrapper = mount(AntigravityQuotaUsage, { props: { usage: { antigravity_quota_groups: [
      { id: 'gemini', display_name: 'Gemini Models', buckets: [
        bucket('gemini-5h', '5h', null), bucket('gemini-weekly', 'weekly', 100, { disabled: true })
      ] }
    ] } } })
    expect(wrapper.findAllComponents(UsageProgressBar)).toHaveLength(0)
    expect(wrapper.get('[data-quota-bucket="gemini-5h"]').text()).toContain('antigravityQuota.unknown')
    expect(wrapper.get('[data-quota-bucket="gemini-weekly"]').text()).toContain('antigravityQuota.notApplicable')
  })

  it('reports an unavailable summary without expandable controls', () => {
    const wrapper = mount(AntigravityQuotaUsage, { props: { usage: null } })
    expect(wrapper.text()).toContain('antigravityQuota.summaryUnavailable')
    expect(wrapper.find('details').exists()).toBe(false)
    expect(wrapper.findAllComponents(UsageProgressBar)).toHaveLength(0)
  })
})
