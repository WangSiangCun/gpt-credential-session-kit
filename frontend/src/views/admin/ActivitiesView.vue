<template>
  <AppLayout>
  <div class="activity-page">
    <header class="page-head">
      <div>
        <p class="eyebrow">EVENT CONTROL</p>
        <h1>活动中心</h1>
        <p>创建限时抽奖，配置随机余额范围并追踪中奖用户。</p>
      </div>
      <button class="pixel-button primary" @click="openCreate">+ 新建{{ activeTab === 'registration' ? '报名开奖' : '抽奖' }}</button>
    </header>

    <div class="pixel-tabs" role="tablist" aria-label="活动类型">
      <button
        role="tab"
        :aria-selected="activeTab === 'lottery'"
        :class="{ active: activeTab === 'lottery' }"
        @click="activeTab = 'lottery'"
      >
        <span class="tab-gem"></span>抽奖
      </button>
      <button
        role="tab"
        :aria-selected="activeTab === 'registration'"
        :class="{ active: activeTab === 'registration' }"
        @click="activeTab = 'registration'"
      >
        <span class="tab-gem tab-gem-registration"></span>报名开奖
      </button>
    </div>

    <section v-if="loading" class="empty-panel">正在读取活动...</section>
    <section v-else-if="visibleCampaigns.length === 0" class="empty-panel">
      <div class="empty-chest">?</div>
      <h2>还没有{{ activeTab === 'registration' ? '报名开奖' : '抽奖' }}活动</h2>
      <p>建立活动后，系统会生成可分享链接。</p>
    </section>
    <section v-else class="campaign-grid">
      <article v-for="item in visibleCampaigns" :key="item.id" class="campaign-panel">
        <div class="campaign-top">
          <div>
            <span class="status-chip" :class="item.status">{{ statusLabel(item.status) }}</span>
            <h2>{{ item.name }}</h2>
          </div>
          <button class="icon-button" title="复制活动链接" @click="copyLink(item.token)">⧉</button>
        </div>
        <p class="description">{{ item.description || '暂无活动说明' }}</p>
        <div class="metrics">
          <div v-if="item.activity_type === 'registration_lottery'"><strong>{{ item.winner_limit || 0 }}</strong><span>中奖份数</span></div>
          <div v-else><strong>{{ item.win_rate }}%</strong><span>中奖率</span></div>
          <div><strong>{{ item.winner_count }}</strong><span>中奖人数</span></div>
          <div><strong>{{ item.draw_count }}</strong><span>参与人数</span></div>
          <div v-if="item.activity_type === 'registration_lottery'"><strong>{{ item.min_total_recharged ? `¥${money(item.min_total_recharged)}` : '无' }}</strong><span>充值门槛</span></div>
          <div v-else><strong>{{ remaining(item) }}</strong><span>剩余份数</span></div>
        </div>
        <div v-if="item.activity_type !== 'registration_lottery'" class="prize-list">
          <div v-for="prize in item.prizes" :key="prize.id" class="prize-row">
            <span>{{ prize.name }}</span>
            <strong>¥{{ money(prize.min_amount) }} ~ ¥{{ money(prize.max_amount) }}</strong>
            <small>{{ prize.remaining_quantity }}/{{ prize.total_quantity }} 份</small>
          </div>
        </div>
        <div class="campaign-footer">
          <span v-if="item.activity_type === 'registration_lottery'">
            开始 {{ formatTime(item.starts_at) }} · 报名截止 {{ formatOptionalTime(item.registration_deadline) }} · 开奖 {{ formatOptionalTime(item.draw_at) }}
          </span>
          <span v-else>{{ formatTime(item.starts_at) }} 开始 · {{ formatTime(item.expires_at) }} 截止</span>
          <div>
            <button class="pixel-button" @click="openWinners(item)">中奖名单</button>
            <button v-if="item.status === 'active'" class="pixel-button danger" @click="requestStop(item)">停止</button>
            <button v-if="isDeletable(item)" class="pixel-button danger" @click="requestDelete(item)">删除</button>
          </div>
        </div>
      </article>
    </section>

    <BaseDialog :show="showCreate" :title="`新建${activeTab === 'registration' ? '报名开奖' : '抽奖'}活动`" width="wide" @close="showCreate = false">
      <form class="create-form" @submit.prevent="createCampaign">
        <div class="form-grid">
          <label>活动名称<input v-model.trim="form.name" required maxlength="120" placeholder="夏日余额掉落" /></label>
          <label v-if="activeTab === 'lottery'">总中奖率 (%)<input v-model.number="form.win_rate" required type="number" min="0" max="100" step="0.01" /></label>
          <label v-else>中奖份数<input v-model.number="form.winner_limit" required type="number" min="1" step="1" /></label>
          <label>累计充值门槛<input v-model.number="form.min_total_recharged" type="number" min="0" step="0.01" /><small>填写 0 表示无门槛，用户端不显示。</small></label>
          <label class="wide">活动说明<textarea v-model.trim="form.description" rows="2" placeholder="展示给参与用户的简短说明"></textarea></label>
          <label>开始时间<input v-model="form.starts_at" type="datetime-local" /></label>
          <label v-if="activeTab === 'lottery'">过期时间<input v-model="form.expires_at" required type="datetime-local" /></label>
          <template v-else>
            <label>报名截止时间<input v-model="form.registration_deadline" required type="datetime-local" /></label>
            <label>开奖时间<input v-model="form.draw_at" required type="datetime-local" /></label>
          </template>
        </div>

        <div v-if="activeTab === 'lottery'" class="prize-editor-head">
          <div><h3>余额奖项</h3><p>例如：10 份、每份随机 0~1 元。</p></div>
          <button type="button" class="pixel-button" @click="addPrize">+ 添加范围</button>
        </div>
        <div v-if="activeTab === 'lottery'" class="prize-editor">
          <div v-for="(prize, index) in form.prizes" :key="index" class="prize-edit-row">
            <label>名称<input v-model.trim="prize.name" required /></label>
            <label>最小余额<input v-model.number="prize.min_amount" type="number" min="0" step="0.01" required /></label>
            <label>最大余额<input v-model.number="prize.max_amount" type="number" :min="prize.min_amount" step="0.01" required /></label>
            <label>份数<input v-model.number="prize.quantity" type="number" min="1" step="1" required /></label>
            <label>权重<input v-model.number="prize.weight" type="number" min="1" step="1" required /></label>
            <button type="button" class="remove-button" :disabled="form.prizes.length === 1" @click="form.prizes.splice(index, 1)">×</button>
          </div>
        </div>
        <p v-if="formError" class="form-error">{{ formError }}</p>
        <div class="dialog-actions">
          <button type="button" class="pixel-button" @click="showCreate = false">取消</button>
          <button class="pixel-button primary" :disabled="creating">{{ creating ? '生成中...' : '生成活动链接' }}</button>
        </div>
      </form>
    </BaseDialog>

    <BaseDialog :show="showWinners" title="中奖名单" width="wide" @close="showWinners = false">
      <div v-if="winnersLoading" class="winner-empty">正在读取名单...</div>
      <div v-else-if="winners.length === 0" class="winner-empty">暂时还没有中奖用户</div>
      <div v-else class="winner-table">
        <button v-for="winner in winners" :key="winner.id" class="winner-row" @click="showUser(winner.user_id)">
          <span class="winner-avatar">{{ (winner.username || winner.user_email).slice(0, 1).toUpperCase() }}</span>
          <span><strong>{{ winner.username || winner.user_email }}</strong><small>{{ winner.user_email }}</small></span>
          <span>{{ winner.prize_name }}</span>
          <strong class="winner-amount">+¥{{ money(winner.amount) }}</strong>
          <time>{{ formatTime(winner.created_at) }}</time>
        </button>
      </div>
    </BaseDialog>

    <ConfirmDialog
      :show="stopTarget !== null"
      title="停止抽奖活动"
      :message="`确定停止“${stopTarget?.name || ''}”吗？停止后用户将无法继续参与本次抽奖。`"
      confirm-text="确认停止"
      cancel-text="暂不停止"
      busy-text="正在停止..."
      :busy="stopping"
      danger
      variant="pixel"
      @confirm="confirmStop"
      @cancel="cancelStop"
    />

    <ConfirmDialog
      :show="deleteTarget !== null"
      title="删除历史活动"
      :message="`确定删除“${deleteTarget?.name || ''}”吗？活动将从列表和分享链接中隐藏，中奖与余额记录仍会保留。`"
      confirm-text="确认删除"
      cancel-text="暂不删除"
      busy-text="正在删除..."
      :busy="deleting"
      danger
      variant="pixel"
      @confirm="confirmDelete"
      @cancel="cancelDelete"
    />

    <UserBalanceHistoryModal
      :show="showBalance"
      :user="balanceUser"
      :hide-actions="true"
      @close="showBalance = false"
    />
  </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import UserBalanceHistoryModal from '@/components/admin/user/UserBalanceHistoryModal.vue'
import lotteryAPI, { type LotteryCampaign, type LotteryWinner } from '@/api/admin/lottery'
import usersAPI from '@/api/admin/users'
import type { AdminUser } from '@/types'

const campaigns = ref<LotteryCampaign[]>([])
const activeTab = ref<'lottery' | 'registration'>('lottery')
const loading = ref(true)
const showCreate = ref(false)
const creating = ref(false)
const formError = ref('')
const showWinners = ref(false)
const winnersLoading = ref(false)
const winners = ref<LotteryWinner[]>([])
const showBalance = ref(false)
const balanceUser = ref<AdminUser | null>(null)
const stopTarget = ref<LotteryCampaign | null>(null)
const stopping = ref(false)
const deleteTarget = ref<LotteryCampaign | null>(null)
const deleting = ref(false)

const visibleCampaigns = computed(() => campaigns.value.filter((item) => (
  activeTab.value === 'registration'
    ? item.activity_type === 'registration_lottery'
    : item.activity_type !== 'registration_lottery'
)))

const localDateTime = (hours: number) => {
  const d = new Date(Date.now() + hours * 60 * 60 * 1000)
  d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
  return d.toISOString().slice(0, 16)
}
const tomorrow = () => localDateTime(24)
const form = reactive({
  name: '',
  description: '',
  activity_type: 'instant_lottery' as 'instant_lottery' | 'registration_lottery',
  win_rate: 30,
  min_total_recharged: 0,
  starts_at: '',
  expires_at: tomorrow(),
  registration_deadline: localDateTime(12),
  draw_at: localDateTime(24),
  winner_limit: 10,
  prizes: [{ name: '随机余额', min_amount: 0, max_amount: 1, quantity: 10, weight: 1 }]
})

function openCreate() {
  form.activity_type = activeTab.value === 'registration' ? 'registration_lottery' : 'instant_lottery'
  formError.value = ''
  showCreate.value = true
}

async function load() {
  loading.value = true
  try {
    const result = await lotteryAPI.list()
    campaigns.value = result.items
    await Promise.all(campaigns.value.map(async (item, index) => {
      campaigns.value[index] = await lotteryAPI.get(item.id)
    }))
  } finally {
    loading.value = false
  }
}

function addPrize() {
  form.prizes.push({ name: `随机余额 ${form.prizes.length + 1}`, min_amount: 1, max_amount: 5, quantity: 5, weight: 1 })
}

async function createCampaign() {
  formError.value = ''
  if (activeTab.value === 'lottery' && form.prizes.some(p => p.max_amount < p.min_amount)) {
    formError.value = '最大余额不能小于最小余额'
    return
  }
  if (activeTab.value === 'registration' && new Date(form.draw_at).getTime() <= new Date(form.registration_deadline).getTime()) {
    formError.value = '开奖时间必须晚于报名截止时间'
    return
  }
  creating.value = true
  try {
    const created = await lotteryAPI.create(activeTab.value === 'registration'
      ? {
          name: form.name,
          description: form.description,
          activity_type: 'registration_lottery',
          win_rate: 0,
          min_total_recharged: form.min_total_recharged,
          starts_at: form.starts_at ? new Date(form.starts_at).toISOString() : undefined,
          registration_deadline: new Date(form.registration_deadline).toISOString(),
          draw_at: new Date(form.draw_at).toISOString(),
          winner_limit: form.winner_limit,
        }
      : {
          name: form.name,
          description: form.description,
          activity_type: 'instant_lottery',
          win_rate: form.win_rate,
          min_total_recharged: form.min_total_recharged,
          starts_at: form.starts_at ? new Date(form.starts_at).toISOString() : undefined,
          expires_at: new Date(form.expires_at).toISOString(),
          prizes: form.prizes,
        })
    showCreate.value = false
    await load()
    await copyLink(created.token)
  } catch (error: any) {
    formError.value = error?.message || '创建失败，请检查配置'
  } finally {
    creating.value = false
  }
}

async function copyLink(token: string) {
  await navigator.clipboard.writeText(`${location.origin}/lottery/${token}`)
}

function requestStop(item: LotteryCampaign) {
  stopTarget.value = item
}

function cancelStop() {
  if (!stopping.value) stopTarget.value = null
}

async function confirmStop() {
  const target = stopTarget.value
  if (!target || stopping.value) return

  stopping.value = true
  try {
    await lotteryAPI.stop(target.id)
    stopTarget.value = null
    await load()
  } finally {
    stopping.value = false
  }
}

function isDeletable(item: LotteryCampaign) {
  return ['stopped', 'exhausted', 'expired'].includes(item.status) || new Date(item.expires_at).getTime() <= Date.now()
}

function requestDelete(item: LotteryCampaign) {
  deleteTarget.value = item
}

function cancelDelete() {
  if (!deleting.value) deleteTarget.value = null
}

async function confirmDelete() {
  const target = deleteTarget.value
  if (!target || deleting.value) return

  deleting.value = true
  try {
    await lotteryAPI.remove(target.id)
    deleteTarget.value = null
    await load()
  } finally {
    deleting.value = false
  }
}

async function openWinners(item: LotteryCampaign) {
  showWinners.value = true
  winnersLoading.value = true
  try {
    winners.value = (await lotteryAPI.winners(item.id, 1, 100)).items
  } finally {
    winnersLoading.value = false
  }
}

async function showUser(userId: number) {
  balanceUser.value = await usersAPI.getById(userId)
  showBalance.value = true
}

const remaining = (item: LotteryCampaign) => item.prizes.reduce((sum, p) => sum + p.remaining_quantity, 0)
const money = (value: number) => Number(value || 0).toFixed(2)
const formatTime = (value: string) => new Date(value).toLocaleString('zh-CN', { hour12: false })
const formatOptionalTime = (value?: string | null) => value ? formatTime(value) : '-'
const statusLabel = (status: string) => ({ active: '进行中', scheduled: '未开始', stopped: '已停止', exhausted: '已抽完', expired: '已过期' }[status] || status)
onMounted(load)
</script>

<style scoped>
.activity-page { min-height: 100%; color: #eef3f8; }
.page-head { display:flex; align-items:flex-end; justify-content:space-between; gap:24px; margin-bottom:22px; }
.eyebrow { color:#75ca79!important; font:700 11px/1 "PixelFont","ArkPixel",monospace; letter-spacing:0; }
h1,h2,h3,strong,.pixel-button,.pixel-tabs { font-family:"PixelFont","ArkPixel",monospace; }
h1 { margin:8px 0 6px; font-size:28px; color:#fff; } .page-head p{color:#96a2b4}
.pixel-tabs { border:3px solid #3d4858; box-shadow:4px 4px 0 #090b0f; background:#171b22; margin-bottom:18px; }
.pixel-tabs button { padding:12px 22px; color:#fff; border-right:3px solid #3d4858; background:#242a34; }
.pixel-tabs button[aria-selected="true"] { background:#4b3b70; color:#fff; }
.tab-gem { display:inline-block;width:8px;height:8px;background:#72c776;margin-right:9px;box-shadow:3px 0 #335c38,0 3px #335c38; }
.tab-gem-registration { background:#d2ae55; box-shadow:3px 0 #6d5424,0 3px #6d5424; }
.campaign-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(410px,1fr));gap:18px}
.campaign-panel,.empty-panel{background:#171b22;border:3px solid #485466;box-shadow:5px 5px 0 #080a0e;padding:20px}
.campaign-top{display:flex;justify-content:space-between;align-items:flex-start}.campaign-top h2{font-size:19px;margin-top:10px;color:white}
.status-chip{padding:4px 8px;border:2px solid #526073;color:#b5c0cf;font:700 11px/1 monospace}.status-chip.active{color:#82d986;border-color:#4b8851}.status-chip.expired,.status-chip.stopped{color:#e89b9b;border-color:#804747}
.icon-button{width:36px;height:34px;border:2px solid #546174;background:#252b35;color:#dfe7f0;font-size:20px;box-shadow:3px 3px #080a0e}
.description{color:#929eae;min-height:42px;margin:15px 0}.metrics{display:grid;grid-template-columns:repeat(4,1fr);border:2px solid #3c4655;background:#101318}
.metrics div{padding:12px;border-right:2px solid #3c4655}.metrics div:last-child{border:0}.metrics strong{display:block;color:#fff;font-size:18px}.metrics span{font-size:11px;color:#778495}
.prize-list{margin:14px 0;border-top:2px solid #343e4c}.prize-row{display:grid;grid-template-columns:1fr auto auto;gap:14px;padding:10px 4px;border-bottom:2px solid #343e4c;align-items:center}.prize-row strong{color:#91df95}.prize-row small{color:#8592a2}
.campaign-footer{display:flex;align-items:center;justify-content:space-between;gap:12px;color:#778495;font-size:12px}.campaign-footer>div{display:flex;gap:8px}
.pixel-button{border:2px solid #556174;background:#242a34;color:#e7edf5;padding:9px 14px;box-shadow:3px 3px 0 #080a0e;font-size:12px}.pixel-button:hover{transform:translate(-1px,-1px);box-shadow:4px 4px #080a0e}.pixel-button.primary{background:#557e45;border-color:#82c96c;color:#fff}.pixel-button.danger{border-color:#8d5050;color:#ffb2b2}.pixel-button:disabled{opacity:.5}
.empty-panel{text-align:center;padding:70px}.empty-chest{margin:auto;width:54px;height:46px;border:4px solid #758297;background:#2d3541;display:grid;place-items:center;font:700 22px monospace;box-shadow:6px 6px #080a0e}.empty-panel h2{margin:20px 0 8px}
.create-form{color:#dce5ef}.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:14px}.form-grid .wide{grid-column:1/-1}
label{display:grid;gap:7px;color:#aeb9c7;font:700 12px/1.3 "PixelFont","ArkPixel",monospace}label small{color:#7f8b9b;font:400 11px/1.4 system-ui,sans-serif}input,textarea{border:2px solid #4a5668!important;border-radius:0!important;background:#11151b!important;color:white!important;padding:10px 11px!important;outline:none}input:focus,textarea:focus{border-color:#78be68!important}
.prize-editor-head{display:flex;justify-content:space-between;align-items:end;margin:22px 0 10px}.prize-editor-head h3{color:white}.prize-editor-head p{color:#7f8b9b;font-size:12px}
.prize-editor{display:grid;gap:10px}.prize-edit-row{display:grid;grid-template-columns:1.3fr repeat(4,1fr) 34px;gap:9px;align-items:end;padding:12px;border:2px solid #3d4857;background:#151920}.remove-button{height:39px;border:2px solid #744848;color:#ffaaaa;background:#2c1d20;font-size:20px}
.dialog-actions{display:flex;justify-content:flex-end;gap:10px;margin-top:20px}.form-error{color:#ffaaaa;margin-top:12px}
.winner-table{border:2px solid #3d4857}.winner-row{width:100%;display:grid;grid-template-columns:40px 1fr 1fr 100px 150px;align-items:center;gap:12px;padding:11px;border-bottom:2px solid #303946;color:#dfe7f1;text-align:left}.winner-row:hover{background:#242b35}.winner-avatar{width:34px;height:34px;display:grid;place-items:center;border:2px solid #5c6a7d;background:#303846}.winner-row span:nth-child(2){display:grid}.winner-row small,.winner-row time{color:#8190a2}.winner-amount{color:#8bdc8f}.winner-empty{padding:50px;text-align:center;color:#8491a1}
@media(max-width:760px){.page-head{align-items:flex-start;flex-direction:column}.campaign-grid{grid-template-columns:1fr}.metrics{grid-template-columns:1fr 1fr}.metrics div:nth-child(2){border-right:0}.prize-edit-row{grid-template-columns:1fr 1fr}.remove-button{grid-column:2}.winner-row{grid-template-columns:36px 1fr auto}.winner-row>span:nth-child(3),.winner-row time{display:none}}
</style>
