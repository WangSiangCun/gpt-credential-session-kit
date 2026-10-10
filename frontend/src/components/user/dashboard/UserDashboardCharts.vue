<template>
  <div class="space-y-6">
    <!-- Date Range Filter -->
    <div class="card dashboard-date-filter-card p-4">
      <div class="flex flex-wrap items-center gap-4">
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('dashboard.timeRange') }}:</span>
          <DateRangePicker :start-date="startDate" :end-date="endDate" @update:startDate="$emit('update:startDate', $event)" @update:endDate="$emit('update:endDate', $event)" @change="$emit('dateRangeChange', $event)" />
        </div>
        <button @click="$emit('refresh')" :disabled="loading" class="btn btn-secondary">
          {{ t('common.refresh') }}
        </button>
        <div class="ml-auto flex items-center gap-2">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('dashboard.granularity') }}:</span>
          <div class="w-28">
            <Select :model-value="granularity" :options="[{value:'day', label:t('dashboard.day')}, {value:'hour', label:t('dashboard.hour')}]" @update:model-value="$emit('update:granularity', $event)" @change="$emit('granularityChange')" />
          </div>
        </div>
      </div>
    </div>

    <!-- Charts Grid -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
      <!-- Model Distribution Chart -->
      <ModelDistributionChart
        v-model:metric="modelMetric"
        :model-stats="models"
        :loading="loading"
        :show-source-toggle="false"
        :show-metric-toggle="true"
        :enable-breakdown="false"
        :show-account-cost="false"
        :start-date="startDate"
        :end-date="endDate"
      />

      <!-- Group Distribution Chart -->
      <GroupDistributionChart
        v-model:metric="groupMetric"
        :group-stats="groups"
        :loading="loading"
        :show-metric-toggle="true"
        :enable-breakdown="false"
        :show-account-cost="false"
        :start-date="startDate"
        :end-date="endDate"
      />

      <!-- Endpoint Distribution Chart -->
      <EndpointDistributionChart
        v-model:metric="endpointMetric"
        :endpoint-stats="endpoints"
        :loading="loading"
        :show-source-toggle="false"
        :show-metric-toggle="true"
        :enable-breakdown="false"
        :title="t('usage.endpointDistribution')"
        :start-date="startDate"
        :end-date="endDate"
      />

      <!-- Token Usage Trend Chart -->
      <TokenUsageTrend :trend-data="trend" :loading="loading" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import GroupDistributionChart from '@/components/charts/GroupDistributionChart.vue'
import EndpointDistributionChart from '@/components/charts/EndpointDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { EndpointStat, GroupStat, TrendDataPoint, ModelStat } from '@/types'

defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[], models: ModelStat[], groups: GroupStat[], endpoints: EndpointStat[] }>()
defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()
const modelMetric = ref<'tokens' | 'actual_cost'>('tokens')
const groupMetric = ref<'tokens' | 'actual_cost'>('tokens')
const endpointMetric = ref<'tokens' | 'actual_cost'>('tokens')
</script>
