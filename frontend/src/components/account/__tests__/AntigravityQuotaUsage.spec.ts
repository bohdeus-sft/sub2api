import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import AntigravityQuotaUsage from '../AntigravityQuotaUsage.vue'
import UsageProgressBar from '../UsageProgressBar.vue'
import type { AntigravityQuotaBucket, AntigravityQuotaGroup } from '@/types'

vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key })
}))
enableAutoUnmount(afterEach)

function bucket(id: string, window: string, utilization: number | null, overrides: Partial<AntigravityQuotaBucket> = {}): AntigravityQuotaBucket {
  return { id, window, utilization, display_name: '', reset_time: '', disabled: false, ...overrides }
}
function group(id: string, buckets: AntigravityQuotaBucket[], source: 'summary' | 'model' = 'summary'): AntigravityQuotaGroup {
  return { id, display_name: id, source, buckets }
}

describe('AntigravityQuotaUsage', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-29T12:00:00Z'))
  })
  afterEach(() => vi.useRealTimers())

  it('shows every provider pool with separate 5-hour/weekly remaining quota and reset', () => {
    const wrapper = mount(AntigravityQuotaUsage, { props: { usage: { antigravity_quota_groups: [
      group('Gemini Models', [
        bucket('gemini-5h', '5h', 25, { reset_time: '2026-10-29T15:00:00Z' }),
        bucket('gemini-weekly', 'weekly', 90, { reset_time: '2026-11-02T12:00:00Z' })
      ]),
      group('Claude and GPT models', [bucket('3p-5h', '5h', 100), bucket('3p-weekly', 'weekly', 0)]),
      group('Gemini 4 Argon', [bucket('gemini-argon-wtu-weekly', 'weekly', 45)]),
      group('Custom pool', [bucket('custom-monthly', 'monthly', 40)])
    ] } } })
    expect(wrapper.findAllComponents(UsageProgressBar)).toHaveLength(6)
    for (const name of ['Gemini Models', 'Claude and GPT models', 'Gemini 4 Argon', 'Custom pool']) expect(wrapper.text()).toContain(name)
    const fiveHour = wrapper.get('[data-quota-bucket="gemini-5h"]')
    expect(fiveHour.text()).toContain('antigravityQuota.fiveHour')
    expect(fiveHour.text()).toContain('75%')
    expect(fiveHour.text()).toContain('3h 0m')
    expect(fiveHour.attributes('title')).toContain('gemini-5h')
    const weekly = wrapper.get('[data-quota-bucket="gemini-weekly"]')
    expect(weekly.text()).toContain('antigravityQuota.weekly')
    expect(weekly.text()).toContain('10%')
    expect(weekly.text()).toContain('4d 0h')
    expect(wrapper.get('[data-quota-bucket="3p-5h"]').text()).toContain('0%')
    expect(wrapper.get('[data-quota-bucket="3p-weekly"]').text()).toContain('100%')
    expect(wrapper.get('[data-quota-bucket="custom-monthly"]').text()).toContain('monthly')
    expect(wrapper.find('[data-test="antigravity-summary-unavailable"]').exists()).toBe(false)
  })

  it('distinguishes unknown/invalid/disabled quotas from known zero and keeps reset dates', () => {
    const wrapper = mount(AntigravityQuotaUsage, { props: { usage: { antigravity_quota_groups: [group('pool', [
      bucket('missing', 'weekly', null, { reset_time: '2026-11-01T00:00:00Z' }),
      bucket('invalid', '5h', Number.NaN), bucket('out-of-range', '5h', 101),
      bucket('disabled', 'weekly', 0, { disabled: true })
    ])] } } })
    expect(wrapper.findComponent(UsageProgressBar).exists()).toBe(false)
    expect(wrapper.get('[data-quota-bucket="missing"]').text()).toContain('antigravityQuota.unknown')
    expect(wrapper.get('time').attributes('datetime')).toBe('2026-11-01T00:00:00Z')
    expect(wrapper.get('[data-quota-bucket="invalid"]').text()).toContain('antigravityQuota.unknown')
    expect(wrapper.get('[data-quota-bucket="out-of-range"]').text()).toContain('antigravityQuota.unknown')
    expect(wrapper.get('[data-quota-bucket="disabled"]').text()).toContain('antigravityQuota.notApplicable')
    expect(wrapper.text()).not.toContain('%')
  })

  it('keeps model-specific weekly quotas including Opus in expandable details beside shared pools', async () => {
    const wrapper = mount(AntigravityQuotaUsage, { props: { usage: { antigravity_quota_groups: [
      group('Claude/GPT', [bucket('3p-weekly', 'weekly', 80)]),
      group('Claude Opus 5.5', [bucket('opus/reported', '', 30), bucket('opus/weekly', 'weekly', 40)], 'model')
    ] } } })
    const details = wrapper.get('details')
    expect(details.attributes('open')).toBeUndefined()
    expect(details.text()).toContain('antigravityQuota.modelDetailsHint')
    expect(details.text()).toContain('Claude Opus 5.5')
    expect(details.get('[data-quota-bucket="opus/reported"]').text()).toContain('antigravityQuota.reported')
    expect(details.get('[data-quota-bucket="opus/weekly"]').text()).toContain('60%')
  })

  it('shows new and custom models with an explicit missing-window notice on older API responses', () => {
    const wrapper = mount(AntigravityQuotaUsage, { props: { usage: { antigravity_quota: {
      'gemini-4-argon-high': { utilization: 20, reset_time: '' },
      'claude-opus-5-5-high': { utilization: 60, reset_time: '' },
      'custom-future-model': { utilization: 0, reset_time: '' }
    } } } })
    expect(wrapper.findAllComponents(UsageProgressBar)).toHaveLength(3)
    expect(wrapper.text()).toContain('gemini-4-argon-high')
    expect(wrapper.text()).toContain('claude-opus-5-5-high')
    expect(wrapper.text()).toContain('custom-future-model')
    expect(wrapper.text()).toContain('antigravityQuota.summaryUnavailable')
    expect(wrapper.text()).not.toContain('antigravityQuota.fiveHour')
    expect(wrapper.find('details').exists()).toBe(false)
  })
})
