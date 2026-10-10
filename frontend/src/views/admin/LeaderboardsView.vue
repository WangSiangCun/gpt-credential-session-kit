<template>
  <AppLayout>
    <div class="leaderboards-page">
      <header class="leaderboards-header">
        <div>
          <div class="eyebrow"><span class="pixel-dot"></span>{{ t('leaderboards.kicker') }}</div>
          <h1>{{ t('leaderboards.title') }}</h1>
          <p>{{ t('leaderboards.description') }}</p>
        </div>
        <div class="header-signal"><span></span>{{ t('leaderboards.liveData') }}</div>
      </header>

      <section class="leaderboards-toolbar pixel-panel">
        <div class="range-presets">
          <button
            v-for="preset in presets"
            :key="preset.key"
            type="button"
            class="pixel-button"
            :class="{ active: rangePreset === preset.key }"
            @click="selectPreset(preset.key)"
          >
            {{ t(`leaderboards.presets.${preset.key}`) }}
          </button>
        </div>
        <div class="range-custom">
          <label>
            <span>{{ t('leaderboards.from') }}</span>
            <input v-model="startDate" type="date" @change="applyCustomRange" />
          </label>
          <label>
            <span>{{ t('leaderboards.to') }}</span>
            <input v-model="endDate" type="date" @change="applyCustomRange" />
          </label>
          <label class="limit-select">
            <span>{{ t('leaderboards.top') }}</span>
            <select v-model.number="limit" @change="loadActiveBoard">
              <option :value="10">10</option>
              <option :value="20">20</option>
              <option :value="50">50</option>
            </select>
          </label>
          <button type="button" class="icon-button" :title="t('leaderboards.refresh')" @click="loadActiveBoard">
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
          </button>
        </div>
      </section>

      <nav class="leaderboard-tabs" :aria-label="t('leaderboards.title')">
        <button
          v-for="board in boards"
          :key="board.key"
          type="button"
          class="leaderboard-tab"
          :class="{ active: activeBoard === board.key }"
          @click="switchBoard(board.key)"
        >
          <Icon :name="board.icon" size="sm" />
          <span>{{ t(`leaderboards.boards.${board.key}.label`) }}</span>
        </button>
      </nav>

      <section class="board-surface pixel-panel">
        <div class="board-heading">
          <div>
            <div class="board-kicker">{{ t(`leaderboards.boards.${activeBoard}.category`) }}</div>
            <h2>{{ t(`leaderboards.boards.${activeBoard}.label`) }}</h2>
            <p>{{ t(`leaderboards.boards.${activeBoard}.hint`) }}</p>
          </div>
          <div class="board-total" v-if="boardTotalLabel">
            <span>{{ t('leaderboards.periodTotal') }}</span>
            <strong>{{ boardTotalLabel }}</strong>
          </div>
        </div>

        <div v-if="loading" class="board-state"><Icon name="refresh" size="lg" class="animate-spin" /><span>{{ t('leaderboards.loading') }}</span></div>
        <div v-else-if="error" class="board-state board-error"><Icon name="exclamationTriangle" size="lg" /><span>{{ error }}</span></div>
        <div v-else-if="rows.length === 0" class="board-state"><Icon name="inbox" size="lg" /><span>{{ t('leaderboards.empty') }}</span></div>
        <div v-else class="ranking-list">
          <div class="ranking-head">
            <span>{{ t('leaderboards.rank') }}</span>
            <span>{{ t(`leaderboards.boards.${activeBoard}.subject`) }}</span>
            <span class="align-right">{{ t(`leaderboards.boards.${activeBoard}.metric`) }}</span>
          </div>
          <button
            v-for="(row, index) in rows"
            :key="row.id"
            type="button"
            class="ranking-row"
            :class="{ podium: index < 3 }"
            @click="openRow(row)"
          >
            <span class="rank-number" :class="`rank-${Math.min(index + 1, 4)}`">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="ranking-subject">
              <span class="subject-mark">{{ subjectMark(index) }}</span>
              <span class="subject-copy">
                <strong>{{ row.title }}</strong>
                <small>{{ row.subtitle }}</small>
              </span>
            </span>
            <span class="ranking-metric align-right">
              <strong>{{ row.metric }}</strong>
              <small>{{ row.secondary }}</small>
            </span>
          </button>
        </div>
      </section>
      <UserBalanceHistoryModal
        :show="showUserDetails"
        :user="selectedUser"
        :hide-actions="true"
        :show-email-action="true"
        @close="closeUserDetails"
        @email="openUserEmail"
      />
      <UserEmailModal
        :show="showUserEmail"
        :user="selectedUser"
        @close="showUserEmail = false"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UserBalanceHistoryModal from '@/components/admin/user/UserBalanceHistoryModal.vue'
import UserEmailModal from '@/components/admin/user/UserEmailModal.vue'
import { adminAPI } from '@/api/admin'
import type { AdminUser, ApiKeyUsageTrendPoint, GroupStat, ModelStat, UserBreakdownItem } from '@/types'

type BoardKey = 'spending' | 'requests' | 'tokens' | 'models' | 'groups' | 'apiKeys' | 'balances'
type BoardIcon = 'dollar' | 'chart' | 'database' | 'cpu' | 'grid' | 'key' | 'users'

interface BoardRow {
  id: string | number
  title: string
  subtitle: string
  metric: string
  secondary: string
}

const { t } = useI18n()
const activeBoard = ref<BoardKey>('spending')
const loading = ref(false)
const error = ref('')
const rows = ref<BoardRow[]>([])
const startDate = ref('')
const endDate = ref('')
const rangePreset = ref('today')
const limit = ref(20)
const totalValue = ref<number | null>(null)
const selectedUser = ref<AdminUser | null>(null)
const showUserDetails = ref(false)
const showUserEmail = ref(false)

const presets = [
  { key: 'today' },
  { key: 'sevenDays' },
  { key: 'thirtyDays' },
  { key: 'custom' }
] as const

const boards: Array<{ key: BoardKey; icon: BoardIcon }> = [
  { key: 'spending', icon: 'dollar' },
  { key: 'requests', icon: 'chart' },
  { key: 'tokens', icon: 'database' },
  { key: 'models', icon: 'cpu' },
  { key: 'groups', icon: 'grid' },
  { key: 'apiKeys', icon: 'key' },
  { key: 'balances', icon: 'users' }
]

const boardTotalLabel = computed(() => {
  if (totalValue.value == null) return ''
  if (activeBoard.value === 'spending') return `$${totalValue.value.toFixed(2)}`
  return formatCompact(totalValue.value)
})

function dateString(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function selectPreset(key: string): void {
  rangePreset.value = key
  const end = new Date()
  const start = new Date(end)
  if (key === 'sevenDays') start.setDate(start.getDate() - 6)
  if (key === 'thirtyDays') start.setDate(start.getDate() - 29)
  if (key === 'custom') return
  startDate.value = dateString(start)
  endDate.value = dateString(end)
  loadActiveBoard()
}

function applyCustomRange(): void {
  rangePreset.value = 'custom'
  loadActiveBoard()
}

function switchBoard(board: BoardKey): void {
  activeBoard.value = board
  loadActiveBoard()
}

function formatCompact(value: number): string {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(1)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return value.toLocaleString()
}

function formatMetric(value: number, kind: 'money' | 'count' | 'tokens'): string {
  return kind === 'money' ? `$${value.toFixed(2)}` : kind === 'tokens' ? formatCompact(value) : value.toLocaleString()
}

function subjectMark(index: number): string {
  return index === 0 ? '◆' : index === 1 ? '◇' : index === 2 ? '▣' : '·'
}

function mapUserRows(items: UserBreakdownItem[], metric: 'actual_cost' | 'requests' | 'total_tokens'): BoardRow[] {
  const kind = metric === 'actual_cost' ? 'money' : metric === 'total_tokens' ? 'tokens' : 'count'
  return items.slice(0, limit.value).map((item) => ({
    id: item.user_id,
    title: item.email || `#${item.user_id}`,
    subtitle: `ID ${item.user_id}`,
    metric: formatMetric(item[metric], kind),
    secondary: `${formatCompact(item.requests)} ${t('leaderboards.secondary.requests')}`
  }))
}

async function loadUserBoard(metric: 'actual_cost' | 'requests' | 'total_tokens'): Promise<void> {
  const response = await adminAPI.dashboard.getUserBreakdown({
    start_date: startDate.value,
    end_date: endDate.value,
    limit: limit.value,
    sort_by: metric === 'actual_cost' ? 'actual_cost' : metric === 'total_tokens' ? 'total_tokens' : 'requests'
  })
  rows.value = mapUserRows(response.users, metric)
  totalValue.value = response.users.reduce((sum, item) => sum + item[metric], 0)
}

function mapModelRows(items: ModelStat[]): BoardRow[] {
  return items.slice().sort((a, b) => b.actual_cost - a.actual_cost).slice(0, limit.value).map((item) => ({
    id: item.model,
    title: item.model,
    subtitle: `${formatCompact(item.requests)} ${t('leaderboards.secondary.requests')}`,
    metric: `$${item.actual_cost.toFixed(2)}`,
    secondary: `${formatCompact(item.total_tokens)} ${t('leaderboards.secondary.tokens')}`
  }))
}

function mapGroupRows(items: GroupStat[]): BoardRow[] {
  return items.slice().sort((a, b) => b.actual_cost - a.actual_cost).slice(0, limit.value).map((item) => ({
    id: item.group_id,
    title: item.group_name || `#${item.group_id}`,
    subtitle: `ID ${item.group_id}`,
    metric: `$${item.actual_cost.toFixed(2)}`,
    secondary: `${formatCompact(item.requests)} ${t('leaderboards.secondary.requests')}`
  }))
}

function mapApiKeyRows(items: ApiKeyUsageTrendPoint[]): BoardRow[] {
  const grouped = new Map<number, { name: string; requests: number; tokens: number }>()
  for (const item of items) {
    const current = grouped.get(item.api_key_id) || { name: item.key_name, requests: 0, tokens: 0 }
    current.requests += item.requests
    current.tokens += item.tokens
    grouped.set(item.api_key_id, current)
  }
  return [...grouped.entries()].sort((a, b) => b[1].requests - a[1].requests).slice(0, limit.value).map(([id, item]) => ({
    id,
    title: item.name || `#${id}`,
    subtitle: `ID ${id}`,
    metric: formatCompact(item.requests),
    secondary: `${formatCompact(item.tokens)} ${t('leaderboards.secondary.tokens')}`
  }))
}

function mapBalanceRows(items: AdminUser[]): BoardRow[] {
  return items.slice(0, limit.value).map((user) => ({
    id: user.id,
    title: user.email || user.username || `#${user.id}`,
    subtitle: user.username || `ID ${user.id}`,
    metric: `$${user.balance.toFixed(2)}`,
    secondary: user.status
  }))
}

async function loadActiveBoard(): Promise<void> {
  if (rangePreset.value === 'custom' && (!startDate.value || !endDate.value)) return
  loading.value = true
  error.value = ''
  totalValue.value = null
  try {
    if (activeBoard.value === 'spending') await loadUserBoard('actual_cost')
    else if (activeBoard.value === 'requests') await loadUserBoard('requests')
    else if (activeBoard.value === 'tokens') await loadUserBoard('total_tokens')
    else if (activeBoard.value === 'models') {
      const response = await adminAPI.dashboard.getModelStats({ start_date: startDate.value, end_date: endDate.value })
      rows.value = mapModelRows(response.models)
      totalValue.value = response.models.reduce((sum, item) => sum + item.actual_cost, 0)
    } else if (activeBoard.value === 'groups') {
      const response = await adminAPI.dashboard.getGroupStats({ start_date: startDate.value, end_date: endDate.value })
      rows.value = mapGroupRows(response.groups)
      totalValue.value = response.groups.reduce((sum, item) => sum + item.actual_cost, 0)
    } else if (activeBoard.value === 'apiKeys') {
      const response = await adminAPI.dashboard.getApiKeyUsageTrend({ start_date: startDate.value, end_date: endDate.value, limit: limit.value })
      rows.value = mapApiKeyRows(response.trend)
      totalValue.value = response.trend.reduce((sum, item) => sum + item.requests, 0)
    } else {
      const response = await adminAPI.users.list(1, limit.value, { sort_by: 'balance', sort_order: 'desc' })
      rows.value = mapBalanceRows(response.items)
    }
  } catch (cause) {
    console.error('Failed to load leaderboard:', cause)
    error.value = t('leaderboards.loadFailed')
    rows.value = []
  } finally {
    loading.value = false
  }
}

async function openRow(row: BoardRow): Promise<void> {
  if (!['spending', 'requests', 'tokens', 'balances'].includes(activeBoard.value)) return
  const userID = Number(row.id)
  if (!Number.isSafeInteger(userID) || userID <= 0) return

  try {
    selectedUser.value = await adminAPI.users.getById(userID)
    showUserDetails.value = true
  } catch (cause) {
    console.error('Failed to load leaderboard user:', cause)
    error.value = t('leaderboards.loadFailed')
  }
}

function closeUserDetails(): void {
  showUserDetails.value = false
  selectedUser.value = null
}

function openUserEmail(): void {
  showUserDetails.value = false
  showUserEmail.value = true
}

onMounted(() => {
  selectPreset('today')
})
</script>

<style scoped>
.leaderboards-page { max-width: 1500px; margin: 0 auto; padding: 18px 20px 34px; color: #e5e7eb; }
.leaderboards-header { display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; margin-bottom: 16px; }
.eyebrow, .board-kicker { display: flex; align-items: center; gap: 7px; color: #a7f3b2; font-family: var(--pool-pixel-cn, monospace); font-size: 11px; text-transform: uppercase; }
.pixel-dot { width: 8px; height: 8px; background: #6ec177; box-shadow: 2px 2px 0 #080a0f; }
.leaderboards-header h1 { margin: 7px 0 3px; color: #f5f3ff; font-family: var(--pool-pixel-cn, monospace); font-size: 25px; }
.leaderboards-header p { margin: 0; color: #9ca3af; font-size: 13px; }
.header-signal { display: inline-flex; align-items: center; gap: 7px; border: 2px solid #4c5b71; padding: 7px 9px; color: #b7c4d7; font-family: var(--pool-pixel-cn, monospace); font-size: 11px; box-shadow: 3px 3px 0 #080a0f; }
.header-signal span { width: 7px; height: 7px; background: #6ec177; animation: signal-blink 1.8s steps(2, end) infinite; }
.pixel-panel { border: 3px solid #4b4370; background: #1b202d; box-shadow: 5px 5px 0 #080a0f; }
.leaderboards-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 10px 12px; }
.range-presets, .range-custom { display: flex; flex-wrap: wrap; align-items: end; gap: 6px; }
.pixel-button, .icon-button { border: 2px solid #46516b; padding: 7px 10px; color: #b9c1d0; background: #121824; font-family: var(--pool-pixel-cn, monospace); font-size: 11px; box-shadow: 2px 2px 0 #080a0f; cursor: pointer; }
.pixel-button:hover, .icon-button:hover, .pixel-button.active { border-color: #a78bfa; color: #fff; background: #30274a; transform: translate(-1px, -1px); }
.range-custom label { display: grid; gap: 3px; color: #8d96a6; font-size: 10px; }
.range-custom input, .range-custom select { height: 29px; border: 2px solid #46516b; padding: 3px 6px; color: #dbe3ef; background: #121824; font-size: 11px; }
.leaderboard-tabs { display: flex; flex-wrap: wrap; gap: 6px; margin: 15px 0 12px; }
.leaderboard-tab { display: inline-flex; align-items: center; gap: 7px; border: 3px solid #343c52; padding: 9px 11px; color: #9da8ba; background: #151b27; font-family: var(--pool-pixel-cn, monospace); font-size: 11px; box-shadow: 3px 3px 0 #080a0f; cursor: pointer; }
.leaderboard-tab:hover, .leaderboard-tab.active { border-color: #8b5cf6; color: #fff; background: #2b2643; transform: translate(-1px, -1px); }
.board-surface { overflow: hidden; }
.board-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; padding: 16px 18px; border-bottom: 3px solid #3a3458; background: linear-gradient(90deg, #282543, #1b202d); }
.board-heading h2 { margin: 5px 0 3px; color: #f5f3ff; font-family: var(--pool-pixel-cn, monospace); font-size: 18px; }
.board-heading p { margin: 0; color: #9ca3af; font-size: 12px; }
.board-total { display: grid; gap: 4px; min-width: 120px; border: 2px solid #46516b; padding: 8px 10px; text-align: right; background: #121824; box-shadow: 3px 3px 0 #080a0f; }
.board-total span { color: #8d96a6; font-size: 10px; }
.board-total strong { color: #a7f3b2; font-family: var(--pool-pixel-cn, monospace); font-size: 17px; }
.ranking-list { padding: 8px 12px 14px; }
.ranking-head, .ranking-row { display: grid; grid-template-columns: 64px minmax(0, 1fr) minmax(150px, .4fr); align-items: center; gap: 12px; }
.ranking-head { padding: 8px 10px; color: #7f899b; font-family: var(--pool-pixel-cn, monospace); font-size: 10px; text-transform: uppercase; }
.ranking-row { width: 100%; border: 2px solid transparent; border-bottom-color: #2a3242; padding: 10px; color: #d7deea; background: #171d29; text-align: left; cursor: pointer; }
.ranking-row:nth-child(odd) { background: #1a2130; }
.ranking-row:hover { border-color: #8067ba; background: #24233a; }
.rank-number { color: #8490a4; font-family: var(--pool-pixel-cn, monospace); font-size: 12px; }
.rank-1 { color: #f5c451; } .rank-2 { color: #cbd5e1; } .rank-3 { color: #d69b6d; }
.ranking-subject { display: flex; min-width: 0; align-items: center; gap: 10px; }
.subject-mark { display: grid; width: 25px; height: 25px; place-items: center; border: 2px solid #4b4370; color: #c4b5fd; background: #292542; font-size: 12px; }
.subject-copy { display: grid; min-width: 0; gap: 3px; }
.subject-copy strong { overflow: hidden; color: #f4f6fa; font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
.subject-copy small, .ranking-metric small { color: #8d96a6; font-size: 10px; }
.ranking-metric { display: grid; gap: 3px; }
.ranking-metric strong { color: #a7f3b2; font-family: var(--pool-pixel-cn, monospace); font-size: 14px; }
.align-right { text-align: right; }
.board-state { display: grid; min-height: 260px; place-content: center; justify-items: center; gap: 10px; color: #9ca3af; font-family: var(--pool-pixel-cn, monospace); font-size: 12px; }
.board-error { color: #fca5a5; }
@keyframes signal-blink { 0%, 42% { opacity: 1; } 43%, 100% { opacity: .35; } }
@media (max-width: 800px) {
  .leaderboards-page { padding: 12px 10px 25px; }
  .leaderboards-header, .leaderboards-toolbar { align-items: stretch; flex-direction: column; }
  .range-custom { align-items: stretch; }
  .range-custom label { flex: 1; }
  .ranking-head, .ranking-row { grid-template-columns: 42px minmax(0, 1fr) minmax(100px, .45fr); gap: 7px; }
}
</style>
