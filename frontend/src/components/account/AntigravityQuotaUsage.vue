<template>
  <div class="max-w-xs space-y-2" data-test="antigravity-quotas">
    <div v-if="groups.length" class="text-[10px] text-gray-500 dark:text-gray-400">
      {{ t('admin.accounts.antigravityQuota.remaining') }}
    </div>
    <div v-for="group in groups" :key="group.id" class="space-y-1" :data-quota-group="group.id">
      <div class="break-words text-[10px] font-medium text-gray-700 dark:text-gray-300">
        {{ group.display_name }}
      </div>
      <div v-for="bucket in group.buckets" :key="bucket.id" :title="bucketTitle(bucket)" :data-quota-bucket="bucket.id">
        <UsageProgressBar
          v-if="remainingPercent(bucket) !== null && !bucket.disabled"
          :label="windowLabel(bucket)"
          :utilization="remainingPercent(bucket) ?? 0"
          :resets-at="bucket.reset_time || null"
          color="indigo"
          remaining-capacity
          label-width="auto"
        />
        <div v-else class="text-[10px] text-gray-500 dark:text-gray-400">
          {{ windowLabel(bucket) }} · {{ bucket.disabled ? t('admin.accounts.antigravityQuota.notApplicable') : t('admin.accounts.antigravityQuota.unknown') }}
          <time v-if="bucket.reset_time" :datetime="bucket.reset_time"> · {{ resetLabel(bucket.reset_time) }}</time>
        </div>
      </div>
    </div>
    <p v-if="!summaryGroups.length" class="whitespace-normal text-[10px] text-gray-500 dark:text-gray-400" data-test="antigravity-summary-unavailable">
      {{ t('admin.accounts.antigravityQuota.summaryUnavailable') }}
    </p>
    <details v-if="summaryGroups.length && modelGroups.length" class="text-[10px] text-gray-500 dark:text-gray-400">
      <summary class="cursor-pointer">{{ t('admin.accounts.antigravityQuota.modelDetails') }}</summary>
      <p class="my-1 whitespace-normal">{{ t('admin.accounts.antigravityQuota.modelDetailsHint') }}</p>
      <div v-for="group in modelGroups" :key="group.id" class="my-2 space-y-1" :title="group.id">
        <div class="break-words">{{ group.display_name }}</div>
        <div v-for="bucket in group.buckets" :key="bucket.id" :title="bucketTitle(bucket)" :data-quota-bucket="bucket.id">
          <UsageProgressBar
            v-if="remainingPercent(bucket) !== null && !bucket.disabled"
            :label="windowLabel(bucket)"
            :utilization="remainingPercent(bucket) ?? 0"
            :resets-at="bucket.reset_time || null"
            color="indigo"
            remaining-capacity
            label-width="auto"
          />
          <div v-else>
            {{ windowLabel(bucket) }} · {{ bucket.disabled ? t('admin.accounts.antigravityQuota.notApplicable') : t('admin.accounts.antigravityQuota.unknown') }}
            <time v-if="bucket.reset_time" :datetime="bucket.reset_time"> · {{ resetLabel(bucket.reset_time) }}</time>
          </div>
        </div>
      </div>
    </details>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountUsageInfo, AntigravityQuotaBucket, AntigravityQuotaGroup } from '@/types'
import UsageProgressBar from './UsageProgressBar.vue'

const props = defineProps<{ usage: AccountUsageInfo | null }>()
const { t } = useI18n()
const summaryGroups = computed(() => props.usage?.antigravity_quota_groups?.filter((group) => group.source === 'summary') ?? [])
const modelGroups = computed<AntigravityQuotaGroup[]>(() => {
  const reported = props.usage?.antigravity_quota_groups?.filter((group) => group.source === 'model') ?? []
  if (reported.length) return reported
  // Compatibility with older servers; every reported model remains visible.
  return Object.entries(props.usage?.antigravity_quota ?? {}).sort(([a], [b]) => a.localeCompare(b)).map(([id, quota]) => ({
    id, display_name: id, source: 'model',
    buckets: [{ id: `${id}/quota`, display_name: '', window: '', utilization: quota.utilization, reset_time: quota.reset_time, disabled: false }]
  }))
})
const groups = computed(() => summaryGroups.value.length ? summaryGroups.value : modelGroups.value)

function remainingPercent(bucket: AntigravityQuotaBucket): number | null {
  const used = bucket.utilization
  return typeof used === 'number' && Number.isFinite(used) && used >= 0 && used <= 100 ? 100 - used : null
}

function windowLabel(bucket: AntigravityQuotaBucket): string {
  if (bucket.window === '5h') return t('admin.accounts.antigravityQuota.fiveHour')
  if (bucket.window === 'weekly') return t('admin.accounts.antigravityQuota.weekly')
  return bucket.display_name || bucket.window || t('admin.accounts.antigravityQuota.reported')
}

function resetLabel(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleString()
}

function bucketTitle(bucket: AntigravityQuotaBucket): string {
  return [bucket.id, bucket.display_name, resetLabel(bucket.reset_time)].filter(Boolean).join(' · ')
}
</script>
