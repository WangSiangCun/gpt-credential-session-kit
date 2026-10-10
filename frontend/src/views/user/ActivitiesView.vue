<template>
  <AppLayout>
    <main class="user-activity-page">
      <header class="activity-hero">
        <div>
          <p class="activity-kicker">POOLAI EVENT TERMINAL</p>
          <h1>活动中心</h1>
          <p>查看平台活动、活动规则与脱敏中奖名单。</p>
        </div>
        <QQGroupButton>
          <template #trigger="{ open }">
            <button
              type="button"
              class="pixel-action pixel-action--primary"
              @click="open"
            >
              <span class="action-gem" aria-hidden="true"></span>
              参加活动
            </button>
          </template>
        </QQGroupButton>
      </header>

      <div class="activity-tabs" role="tablist" aria-label="活动类型">
        <button type="button" role="tab" :aria-selected="activeTab === 'lottery'" @click="activeTab = 'lottery'">
          <span class="tab-flag" aria-hidden="true"></span>
          抽奖
        </button>
        <button type="button" role="tab" :aria-selected="activeTab === 'registration'" @click="activeTab = 'registration'">
          <span class="tab-flag tab-flag-registration" aria-hidden="true"></span>
          报名开奖
        </button>
        <span>活动链接由管理员在群内发布</span>
      </div>

      <section v-if="loading" class="activity-state">
        <div class="state-loader" aria-hidden="true"></div>
        <strong>正在读取活动...</strong>
      </section>
      <section v-else-if="loadError" class="activity-state activity-state--error">
        <div class="state-mark">!</div>
        <strong>活动读取失败</strong>
        <p>{{ loadError }}</p>
        <button type="button" class="pixel-action" @click="load">重新读取</button>
      </section>
      <section v-else-if="visibleCampaigns.length === 0" class="activity-state">
        <div class="state-mark">?</div>
        <strong>暂时没有活动</strong>
        <p>新活动发布后会显示在这里。</p>
      </section>

      <section v-else class="activity-grid" :aria-label="`${activeTab === 'registration' ? '报名开奖' : '抽奖'}活动列表`">
        <article v-for="campaign in visibleCampaigns" :key="campaign.id" class="activity-panel">
          <div class="panel-head">
            <div>
              <span class="status-chip" :class="campaign.status">{{ statusLabel(campaign.status) }}</span>
              <h2>{{ campaign.name }}</h2>
            </div>
            <span class="event-id">#{{ String(campaign.id).padStart(4, '0') }}</span>
          </div>

          <p class="activity-description">{{ campaign.description || '管理员暂未填写活动说明。' }}</p>

          <div class="activity-metrics">
            <div v-if="campaign.activity_type === 'registration_lottery'"><strong>{{ campaign.winner_limit || 0 }}</strong><span>中奖份数</span></div>
            <div v-else><strong>{{ campaign.win_rate }}%</strong><span>中奖率</span></div>
            <div><strong>{{ campaign.winner_count }}</strong><span>中奖人数</span></div>
            <div v-if="campaign.activity_type === 'registration_lottery'"><strong>{{ campaign.status === 'exhausted' ? '已开奖' : '待开奖' }}</strong><span>活动状态</span></div>
            <div v-else><strong>{{ remaining(campaign) }}</strong><span>剩余份数</span></div>
          </div>

          <div v-if="campaign.activity_type !== 'registration_lottery'" class="prize-board">
            <div v-for="prize in campaign.prizes" :key="prize.id" class="prize-line">
              <span>{{ prize.name }}</span>
              <strong>¥{{ money(prize.min_amount) }} - ¥{{ money(prize.max_amount) }}</strong>
              <small>剩余 {{ prize.remaining_quantity }} 份</small>
            </div>
          </div>

          <p v-if="campaign.min_total_recharged" class="gate-note">报名门槛：累计充值满 ¥{{ money(campaign.min_total_recharged) }}</p>

          <div class="time-board">
            <template v-if="campaign.activity_type === 'registration_lottery'">
              <span><i></i>活动开始：{{ formatTime(campaign.starts_at) }}</span>
              <span><i></i>报名截止：{{ formatOptionalTime(campaign.registration_deadline) }}</span>
              <span><i></i>开奖时间：{{ formatOptionalTime(campaign.draw_at) }}</span>
            </template>
            <template v-else>
              <span><i></i>开始：{{ formatTime(campaign.starts_at) }}</span>
              <span><i></i>截止：{{ formatTime(campaign.expires_at) }}</span>
            </template>
          </div>

          <footer class="panel-actions">
            <button type="button" class="pixel-action" @click="openWinners(campaign)">中奖名单</button>
          </footer>
        </article>
      </section>

      <PixelDialog
        :show="winnerCampaign !== null"
        :title="`${winnerCampaign?.name || ''} · 中奖名单`"
        width="wide"
        @close="closeWinners"
      >
        <div v-if="winnersLoading" class="winner-state">正在读取中奖名单...</div>
        <div v-else-if="winners.length === 0" class="winner-state">暂时还没有中奖用户</div>
        <div v-else class="winner-list">
          <div v-for="winner in winners" :key="winner.id" class="winner-line">
            <span class="winner-avatar">{{ winner.masked_account.slice(0, 1) }}</span>
            <strong>{{ winner.masked_account }}</strong>
            <span>{{ winner.prize_name }}</span>
            <b>{{ winner.amount > 0 ? `+¥${money(winner.amount)}` : '已入选' }}</b>
            <time>{{ formatTime(winner.created_at) }}</time>
          </div>
        </div>
        <p class="privacy-note">账号已由服务器脱敏，仅展示必要的中奖信息。</p>
      </PixelDialog>

    </main>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import PixelDialog from '@/components/common/PixelDialog.vue'
import QQGroupButton from '@/components/common/QQGroupButton.vue'
import {
  listLotteries,
  listLotteryWinners,
  type LotteryPublicCampaign,
  type LotteryPublicWinner,
} from '@/api/lottery'

const campaigns = ref<LotteryPublicCampaign[]>([])
const activeTab = ref<'lottery' | 'registration'>('lottery')
const loading = ref(true)
const loadError = ref('')
const winnerCampaign = ref<LotteryPublicCampaign | null>(null)
const winners = ref<LotteryPublicWinner[]>([])
const winnersLoading = ref(false)

const visibleCampaigns = computed(() => campaigns.value.filter((campaign) => (
  activeTab.value === 'registration'
    ? campaign.activity_type === 'registration_lottery'
    : campaign.activity_type !== 'registration_lottery'
)))

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    campaigns.value = (await listLotteries()).items
  } catch (error: any) {
    loadError.value = error?.message || '请稍后再试'
  } finally {
    loading.value = false
  }
}

async function openWinners(campaign: LotteryPublicCampaign) {
  winnerCampaign.value = campaign
  winners.value = []
  winnersLoading.value = true
  try {
    winners.value = (await listLotteryWinners(campaign.id)).items
  } finally {
    winnersLoading.value = false
  }
}

function closeWinners() {
  winnerCampaign.value = null
  winners.value = []
}

const remaining = (campaign: LotteryPublicCampaign) => (
  campaign.prizes.reduce((sum, prize) => sum + prize.remaining_quantity, 0)
)
const money = (value: number) => Number(value || 0).toFixed(2)
const formatTime = (value: string) => new Date(value).toLocaleString('zh-CN', { hour12: false })
const formatOptionalTime = (value?: string | null) => value ? formatTime(value) : '-'
const statusLabel = (status: LotteryPublicCampaign['status']) => ({
  active: '进行中',
  scheduled: '未开始',
  stopped: '已停止',
  exhausted: '已抽完',
  expired: '已过期',
}[status])

onMounted(load)
</script>

<style scoped>
.user-activity-page{min-height:100%;color:#edf3f8}.activity-hero{display:flex;align-items:flex-end;justify-content:space-between;gap:24px;margin-bottom:22px;padding:22px 24px;border:4px solid #465364;background:#151a21;box-shadow:6px 6px 0 #07090c,inset 3px 3px 0 rgba(255,255,255,.035)}
.activity-kicker{margin:0;color:#78cf7c!important;font:700 11px/1 'PoolPixelCN','PoolPixelEN',monospace}.activity-hero h1{margin:10px 0 7px;color:#fff;font:700 28px/1.25 'PoolPixelCN','PoolPixelEN',monospace;text-shadow:2px 2px #080a0d}.activity-hero p{margin:0;color:#99a6b6}
.activity-tabs{display:flex;align-items:center;gap:0;margin-bottom:18px;border:3px solid #3d4858;background:#13171d;box-shadow:4px 4px 0 #080a0d}.activity-tabs button{display:flex;align-items:center;gap:10px;border:0;border-right:3px solid #3d4858;background:#242b35;padding:12px 22px;color:#fff;font:700 13px/1 'PoolPixelCN','PoolPixelEN',monospace}.activity-tabs button[aria-selected="true"]{background:#4b3b70}.activity-tabs>span{margin-left:20px;color:#697687;font-size:12px}.tab-flag{width:12px;height:12px;background:#75cb79;box-shadow:4px 0 #315837,0 4px #315837}.tab-flag-registration{background:#d2ae55;box-shadow:4px 0 #6d5424,0 4px #6d5424}
.activity-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(390px,1fr));gap:20px}.activity-panel{position:relative;border:4px solid #475465;background:#171c23;padding:20px;box-shadow:6px 6px 0 #07090c,inset 3px 3px 0 rgba(255,255,255,.03)}.activity-panel:before{position:absolute;top:-4px;right:25px;width:13px;height:8px;background:#7c5bbd;box-shadow:3px 3px #35254f;content:''}
.panel-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px}.panel-head h2{margin:10px 0 0;color:#fff;font:700 19px/1.4 'PoolPixelCN','PoolPixelEN',monospace}.event-id{color:#647183;font:700 11px/1 monospace}.status-chip{display:inline-block;border:2px solid #596678;padding:4px 8px;color:#b6c0ce;font:700 10px/1 'PoolPixelCN','PoolPixelEN',monospace}.status-chip.active{border-color:#4f8c55;color:#87dd8c}.status-chip.scheduled{border-color:#806b36;color:#f1d67f}.status-chip.stopped,.status-chip.expired{border-color:#79464a;color:#eaa0a4}.status-chip.exhausted{border-color:#66518b;color:#c5afea}
.activity-description{min-height:44px;margin:14px 0;color:#98a4b3;line-height:1.65}.activity-metrics{display:grid;grid-template-columns:repeat(3,1fr);border:3px solid #35404d;background:#0f1318}.activity-metrics div{padding:13px 12px;border-right:3px solid #35404d}.activity-metrics div:last-child{border:0}.activity-metrics strong{display:block;color:#f5f7fa;font:700 19px/1 'PoolPixelCN','PoolPixelEN',monospace}.activity-metrics span{display:block;margin-top:7px;color:#748193;font-size:11px}
.prize-board{margin-top:14px;border-top:3px solid #35404d}.prize-line{display:grid;grid-template-columns:minmax(0,1fr) auto auto;align-items:center;gap:14px;padding:11px 3px;border-bottom:3px solid #303a46}.prize-line span{overflow:hidden;color:#b8c2cd;text-overflow:ellipsis;white-space:nowrap}.prize-line strong{color:#8ad88f;font:700 12px/1.4 'PoolPixelCN','PoolPixelEN',monospace}.prize-line small{color:#758294}.time-board{display:grid;gap:7px;padding:14px 2px;color:#788697;font-size:11px}.time-board span{display:flex;align-items:center;gap:8px}.time-board i{width:6px;height:6px;background:#7857b8;box-shadow:2px 2px #302047}
.gate-note{margin:12px 0 0;border-left:4px solid #d2ae55;background:#27231a;padding:9px 11px;color:#e7d38b;font-size:11px}
.panel-actions{display:flex;justify-content:flex-end;gap:10px;border-top:3px solid #35404d;padding-top:14px}.pixel-action{display:inline-flex;min-height:38px;align-items:center;justify-content:center;gap:10px;border:3px solid #536174;border-radius:0;background:#252c35;padding:8px 14px;color:#e8edf3;font:700 12px/1 'PoolPixelCN','PoolPixelEN',monospace;box-shadow:3px 3px 0 #07090c}.pixel-action:hover:not(:disabled){transform:translate(-1px,-1px);box-shadow:4px 4px 0 #07090c}.pixel-action:disabled{cursor:not-allowed;opacity:.45}.pixel-action--primary{border-color:#79cb72;background:#527e49;color:#fff}.action-gem{width:8px;height:8px;background:#9bea8f;box-shadow:3px 0 #315e31,0 3px #315e31}
.activity-state{display:grid;min-height:300px;place-items:center;align-content:center;gap:12px;border:4px solid #465364;background:#151a21;padding:48px;text-align:center;box-shadow:6px 6px 0 #07090c}.activity-state strong{color:#fff;font:700 17px/1.4 'PoolPixelCN','PoolPixelEN',monospace}.activity-state p{margin:0;color:#8794a5}.state-mark,.state-loader{display:grid;width:54px;height:48px;place-items:center;border:4px solid #607085;background:#2a333f;color:#fff;font:700 22px monospace;box-shadow:5px 5px #07090c}.state-loader{animation:state-pulse 1s steps(2) infinite}.state-loader:after{content:'...'}.activity-state--error .state-mark{border-color:#8a5055;background:#3b2328}@keyframes state-pulse{50%{filter:brightness(1.35)}}
.winner-list{border:3px solid #3b4654}.winner-line{display:grid;grid-template-columns:38px minmax(120px,1fr) minmax(120px,1fr) 100px 170px;align-items:center;gap:12px;border-bottom:3px solid #303a46;padding:11px;color:#dce4ed}.winner-line:last-child{border:0}.winner-line b{color:#8ddd91;font-family:'PoolPixelCN','PoolPixelEN',monospace}.winner-line time{color:#7f8c9d;font-size:11px}.winner-avatar{display:grid;width:34px;height:34px;place-items:center;border:3px solid #596779;background:#2b3440;color:#fff;font-family:'PoolPixelCN','PoolPixelEN',monospace}.winner-state{padding:52px;text-align:center;color:#8e9bac;font-family:'PoolPixelCN','PoolPixelEN',monospace}.privacy-note{margin:14px 0 0;border-left:5px solid #78c971;background:#1c222b;padding:11px 13px;color:#8e9bac;font-size:11px}
@media(max-width:760px){.activity-hero{align-items:flex-start;flex-direction:column}.activity-tabs>span{display:none}.activity-grid{grid-template-columns:1fr}.prize-line{grid-template-columns:1fr auto}.prize-line small{grid-column:1/-1}.winner-line{grid-template-columns:36px 1fr auto}.winner-line>span:nth-child(3),.winner-line time{display:none}}
</style>
