<template>
  <component :is="embedded ? 'div' : AppLayout">
  <main class="lottery-world" :class="{ 'lottery-world--embedded': embedded }">
    <div class="cave-grid"></div>
    <section v-if="loading" class="lottery-machine loading-machine">正在进入活动...</section>
    <section v-else-if="loadError" class="lottery-machine result-panel">
      <div class="pixel-face">×</div><h1>无法进入活动</h1><p>{{ loadError }}</p>
      <router-link to="/home" class="draw-button">返回首页</router-link>
    </section>
    <section v-else-if="campaign" class="lottery-layout">
      <header class="event-title">
        <p>{{ campaign.activity_type === 'registration_lottery' ? 'POOLAI // REGISTRATION' : 'POOLAI // TOKEN DROP' }}</p>
        <h1>{{ campaign.name }}</h1>
        <span>{{ campaign.description }}</span>
      </header>

      <div
        v-if="campaign.activity_type === 'registration_lottery'"
        class="registration-entry"
        :class="{ registering: drawing, registered: result?.status === 'registered', won: result?.status === 'won', lost: result?.status === 'lost', closed: registrationClosed && !result }"
      >
        <div class="registration-header">
          <div class="registration-label">
            <i aria-hidden="true"></i>
            <span>报名开奖</span>
          </div>
          <span class="registration-code">{{ registrationPhase }}</span>
        </div>
        <div class="registration-body">
          <div class="registration-emblem" aria-hidden="true">
            <i v-for="n in 4" :key="n"></i>
            <strong v-if="result?.status === 'won'">★</strong>
            <strong v-else-if="result?.status === 'lost'">×</strong>
            <strong v-else-if="result?.status === 'registered'">✓</strong>
            <strong v-else-if="registrationClosed">·</strong>
            <strong v-else>+</strong>
          </div>
          <div class="registration-copy">
            <span class="registration-kicker">{{ registrationPhase }}</span>
            <strong v-if="drawing">报名信息确认中...</strong>
            <strong v-else-if="result?.status === 'won'">恭喜你，中奖啦</strong>
            <strong v-else-if="result?.status === 'lost'">本次未中奖</strong>
            <strong v-else-if="result?.status === 'registered'">报名成功</strong>
            <strong v-else-if="registrationClosed">报名已截止</strong>
            <strong v-else>立即报名参加</strong>
            <span v-if="drawing">正在确认你的报名资格，请稍候。</span>
            <span v-else-if="result?.status === 'won'">你已进入中奖名单，恭喜你获得本次活动奖励。</span>
            <span v-else-if="result?.status === 'lost'">感谢参与，下一场活动等你来。</span>
            <span v-else-if="result?.status === 'registered'">报名已提交，开奖后会在这里显示结果。</span>
            <span v-else-if="registrationClosed">报名已截止，请等待开奖时间公布结果。</span>
            <span v-else>报名后等待统一开奖，结果将在开奖时间公布。</span>
            <div class="registration-progress" :class="{ complete: result?.status === 'registered', resolved: result?.status === 'won' || result?.status === 'lost' }" aria-hidden="true">
              <i v-for="n in 10" :key="n"></i>
            </div>
          </div>
        </div>
        <div class="registration-footer">
          <div class="registration-status" aria-label="活动进度">
            <span class="registration-status__item" :class="{ 'is-active': !registrationClosed && !result, complete: !!result }"><i aria-hidden="true"></i>报名</span>
            <i class="registration-status__connector" aria-hidden="true"></i>
            <span class="registration-status__item" :class="{ 'is-active': registrationClosed && !result, complete: result?.status === 'won' || result?.status === 'lost' }"><i aria-hidden="true"></i>开奖</span>
          </div>
          <button class="draw-button" :disabled="drawDisabled" @click="draw">
            {{ buttonText }}
          </button>
        </div>
        <p v-if="drawError" class="draw-error" role="alert">{{ drawError }}</p>
      </div>

      <div v-else class="lottery-machine" :class="{ drawing, revealed }">
        <div class="machine-lights"><i v-for="n in 12" :key="n"></i></div>
        <div class="machine-sign">LUCKY BLOCK</div>
        <div class="reel-window">
          <div v-if="drawing" class="reel-track">
            <span v-for="n in 12" :key="n">{{ n % 3 === 0 ? '¥' : n % 2 ? '?' : '◆' }}</span>
          </div>
          <div v-else-if="result" class="result-symbol" :class="result.status">
            <span>{{ result.status === 'registered' ? '✓' : result.status === 'won' ? '¥' : '·' }}</span>
            <strong v-if="result.status === 'registered'">报名成功</strong>
            <strong v-else-if="result.status === 'won'">+{{ money(result.amount) }}</strong>
            <strong v-else>再接再厉</strong>
          </div>
          <div v-else class="idle-symbol"><span>?</span><small>余额方块正在等待</small></div>
        </div>
        <div class="machine-base">
          <button class="draw-button" :disabled="drawDisabled" @click="draw">
            {{ buttonText }}
          </button>
          <div class="lever" :class="{ pulled: drawing }"><i></i></div>
        </div>
        <p v-if="drawError" class="draw-error" role="alert">{{ drawError }}</p>
        <div v-if="revealed && result?.status === 'won'" class="pixel-fireworks" aria-hidden="true">
          <div v-for="burst in 4" :key="burst" class="pixel-firework">
            <i v-for="n in 18" :key="n" :style="{ '--n': n }"></i>
          </div>
        </div>
      </div>

      <aside class="event-board">
        <div class="board-stat"><strong>{{ campaign.winner_count }}</strong><span>当前中奖人数</span></div>
        <div v-if="campaign.activity_type === 'registration_lottery'" class="board-stat"><strong>{{ campaign.winner_limit || 0 }}</strong><span>中奖名额</span></div>
        <div v-else class="board-stat"><strong>{{ remaining }}</strong><span>剩余奖品</span></div>
        <div v-if="campaign.activity_type === 'registration_lottery'" class="deadline">
          <span>活动开始</span><strong>{{ formatTime(campaign.starts_at) }}</strong>
          <span>报名截止</span><strong>{{ formatOptionalTime(campaign.registration_deadline) }}</strong>
          <span>开奖时间</span><strong>{{ formatOptionalTime(campaign.draw_at) }}</strong>
        </div>
        <div v-else class="deadline">
          <span>活动开始</span><strong>{{ formatTime(campaign.starts_at) }}</strong>
          <span>活动截止</span><strong>{{ formatTime(campaign.expires_at) }}</strong>
        </div>
        <div v-if="campaign.activity_type !== 'registration_lottery'" class="prize-preview">
          <h2>可能掉落</h2>
          <div v-for="prize in campaign.prizes" :key="prize.id">
            <span>{{ prize.name }}</span>
            <strong>¥{{ money(prize.min_amount) }} ~ ¥{{ money(prize.max_amount) }}</strong>
          </div>
        </div>
        <p v-if="campaign.min_total_recharged" class="rule rule-gate">报名门槛：累计充值满 ¥{{ money(campaign.min_total_recharged) }}。</p>
        <p class="rule">每位用户仅有一次机会。{{ campaign.activity_type === 'registration_lottery' ? '报名后等待统一开奖。' : '抽奖结果与随机余额均由服务器生成。' }}</p>
      </aside>
    </section>
  </main>
  </component>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import { drawLottery, getLottery, type LotteryDraw } from '@/api/lottery'
import type { LotteryPublicCampaign } from '@/api/lottery'

const props = withDefaults(defineProps<{
  token?: string
  embedded?: boolean
}>(), {
  token: '',
  embedded: false,
})

const emit = defineEmits<{
  drawn: [result: LotteryDraw]
}>()

const route = useRoute()
const loading = ref(true)
const drawing = ref(false)
const revealed = ref(false)
const loadError = ref('')
const drawError = ref('')
const campaign = ref<LotteryPublicCampaign | null>(null)
const result = ref<LotteryDraw | null>(null)
const token = computed(() => props.token || String(route.params.token || ''))
const clockTick = ref(Date.now())
let lotteryAudioContext: AudioContext | null = null
let winSoundPlayed = false
let rollingSoundTimer: number | null = null
let rollingSoundStep = 0
let clockTimer: number | null = null

const remaining = computed(() => campaign.value?.prizes.reduce((sum, p) => sum + p.remaining_quantity, 0) || 0)
const registrationClosed = computed(() => {
  const deadline = campaign.value?.registration_deadline
  return campaign.value?.activity_type === 'registration_lottery'
    && !!deadline
    && clockTick.value >= new Date(deadline).getTime()
})
const registrationPhase = computed(() => {
  if (result.value?.status === 'won') return '中奖结果'
  if (result.value?.status === 'lost') return '结果已公布'
  if (result.value?.status === 'registered') return '等待开奖'
  if (registrationClosed.value) return '报名已截止'
  if (campaign.value?.status === 'scheduled') return '活动未开始'
  return '报名进行中'
})
const drawDisabled = computed(() => (
  drawing.value
  || !!result.value
  || campaign.value?.status !== 'active'
  || registrationClosed.value
))
const buttonText = computed(() => {
  if (drawing.value) return campaign.value?.activity_type === 'registration_lottery' ? '正在提交报名...' : '方块翻滚中...'
  if (result.value?.status === 'registered') return '已报名'
  if (result.value?.status === 'won') return '已中奖'
  if (result.value?.status === 'lost') return '未中奖'
  if (result.value) return '已参与'
  if (registrationClosed.value) return '报名已截止，等待开奖'
  if (campaign.value?.status === 'scheduled') return '活动尚未开始'
  if (campaign.value?.status !== 'active') return '活动已结束'
  return campaign.value?.activity_type === 'registration_lottery' ? '立即报名' : '拉下拉杆'
})

async function load() {
  try {
    const state = await getLottery(token.value)
    campaign.value = state.campaign
    result.value = state.my_draw
    revealed.value = !!state.my_draw
  } catch (e: any) {
    loadError.value = e?.message || '活动不存在或暂时不可用'
  } finally {
    loading.value = false
  }
}

async function draw() {
  const isRegistration = campaign.value?.activity_type === 'registration_lottery'
  if (!isRegistration) {
    prepareLotteryAudio()
    playLeverSound()
    startRollingSound()
  }
  drawing.value = true
  drawError.value = ''
  try {
    const [drawResult] = await Promise.all([
      drawLottery(token.value),
      new Promise(resolve => setTimeout(resolve, isRegistration ? 650 : 1900))
    ])
    if (!isRegistration) stopRollingSound()
    result.value = drawResult
    revealed.value = true
    emit('drawn', drawResult)
    if (!isRegistration && drawResult.status === 'won') {
      playWinSoundOnce()
      if (campaign.value) campaign.value.winner_count++
    }
  } catch (e: any) {
    drawError.value = e?.message || '抽奖失败，请稍后再试'
  } finally {
    if (!isRegistration) stopRollingSound()
    drawing.value = false
  }
}

function prepareLotteryAudio() {
  if (lotteryAudioContext) {
    if (lotteryAudioContext.state === 'suspended') void lotteryAudioContext.resume()
    return lotteryAudioContext
  }

  const AudioContextConstructor =
    window.AudioContext ||
    (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
  if (!AudioContextConstructor) return null

  lotteryAudioContext = new AudioContextConstructor()
  if (lotteryAudioContext.state === 'suspended') void lotteryAudioContext.resume()
  return lotteryAudioContext
}

function schedulePixelTone(
  audioContext: AudioContext,
  frequency: number,
  startAt: number,
  duration: number,
  volume: number,
  type: OscillatorNode['type'] = 'square',
  endFrequency = frequency
) {
  const oscillator = audioContext.createOscillator()
  const gain = audioContext.createGain()

  oscillator.type = type
  oscillator.frequency.setValueAtTime(frequency, startAt)
  if (endFrequency !== frequency) {
    oscillator.frequency.exponentialRampToValueAtTime(Math.max(1, endFrequency), startAt + duration)
  }
  gain.gain.setValueAtTime(0.0001, startAt)
  gain.gain.exponentialRampToValueAtTime(volume, startAt + Math.min(0.012, duration / 3))
  gain.gain.exponentialRampToValueAtTime(0.0001, startAt + duration)

  oscillator.connect(gain)
  gain.connect(audioContext.destination)
  oscillator.start(startAt)
  oscillator.stop(startAt + duration + 0.02)
}

function playLeverSound() {
  const audioContext = prepareLotteryAudio()
  if (!audioContext) return

  void audioContext.resume().then(() => {
    const startAt = audioContext.currentTime + 0.01
    schedulePixelTone(audioContext, 150, startAt, 0.18, 0.07, 'sawtooth', 58)
    schedulePixelTone(audioContext, 720, startAt + 0.015, 0.045, 0.035, 'square', 260)
    schedulePixelTone(audioContext, 92, startAt + 0.14, 0.11, 0.065, 'square', 70)
  }).catch(() => {})
}

function playRollingTick() {
  const audioContext = prepareLotteryAudio()
  if (!audioContext || audioContext.state !== 'running') return

  const frequencies = [196, 247, 220, 294]
  const frequency = frequencies[rollingSoundStep % frequencies.length]
  const startAt = audioContext.currentTime + 0.005
  schedulePixelTone(audioContext, frequency, startAt, 0.055, 0.026, 'square', frequency * 0.88)
  if (rollingSoundStep % 4 === 0) {
    schedulePixelTone(audioContext, frequency / 2, startAt, 0.075, 0.018, 'triangle')
  }
  rollingSoundStep++
}

function startRollingSound() {
  stopRollingSound()
  rollingSoundStep = 0
  rollingSoundTimer = window.setInterval(playRollingTick, 115)
}

function stopRollingSound() {
  if (rollingSoundTimer !== null) {
    window.clearInterval(rollingSoundTimer)
    rollingSoundTimer = null
  }
}

function playWinSoundOnce() {
  if (winSoundPlayed) return
  const audioContext = prepareLotteryAudio()
  if (!audioContext) return
  winSoundPlayed = true

  void audioContext.resume().then(() => {
    const startAt = audioContext.currentTime + 0.03
    const melody = [
      { frequency: 523.25, offset: 0, duration: 0.11 },
      { frequency: 659.25, offset: 0.1, duration: 0.11 },
      { frequency: 783.99, offset: 0.2, duration: 0.13 },
      { frequency: 1046.5, offset: 0.31, duration: 0.2 },
      { frequency: 783.99, offset: 0.5, duration: 0.1 },
      { frequency: 1046.5, offset: 0.59, duration: 0.12 },
      { frequency: 1318.51, offset: 0.7, duration: 0.4 }
    ]

    melody.forEach(({ frequency, offset, duration }, index) => {
      schedulePixelTone(
        audioContext,
        frequency,
        startAt + offset,
        duration,
        index === melody.length - 1 ? 0.075 : 0.052,
        index < 6 ? 'square' : 'triangle'
      )
    })

    ;[1046.5, 1318.51, 1567.98].forEach((frequency, index) => {
      schedulePixelTone(audioContext, frequency, startAt + 0.72, 0.46, 0.032, 'triangle')
      schedulePixelTone(audioContext, frequency * 2, startAt + 0.36 + index * 0.1, 0.07, 0.022, 'square')
    })
  }).catch(() => {
    winSoundPlayed = false
  })
}

const money = (value: number) => Number(value || 0).toFixed(2)
const formatTime = (value: string) => new Date(value).toLocaleString('zh-CN', { hour12: false })
const formatOptionalTime = (value?: string | null) => value ? formatTime(value) : '-'
onMounted(() => {
  clockTimer = window.setInterval(() => {
    clockTick.value = Date.now()
  }, 1000)
  void load()
})
onBeforeUnmount(() => {
  if (clockTimer !== null) {
    window.clearInterval(clockTimer)
    clockTimer = null
  }
  stopRollingSound()
  if (lotteryAudioContext) void lotteryAudioContext.close()
  lotteryAudioContext = null
})
</script>

<style scoped>
.lottery-world{position:relative;min-height:calc(100vh - 90px);overflow:hidden;background:#0d1015;color:#eef4f8;padding:34px;isolation:isolate}
.lottery-world--embedded{min-height:0;padding:12px 8px 20px;background:#10141a}
.lottery-world--embedded .lottery-layout{max-width:980px}
.lottery-world--embedded .event-title h1{font-size:26px}
.cave-grid{position:absolute;inset:0;z-index:-2;background-color:#11151b;background-image:linear-gradient(90deg,transparent 31px,#202833 32px),linear-gradient(transparent 31px,#202833 32px);background-size:32px 32px;opacity:.68}
.lottery-world:after{content:"";position:absolute;inset:0;z-index:-1;background:linear-gradient(90deg,rgba(8,10,14,.78),transparent 35%,transparent 65%,rgba(8,10,14,.72))}
.lottery-layout{max-width:1100px;margin:auto;display:grid;grid-template-columns:minmax(420px,1.35fr) minmax(270px,.65fr);gap:30px;align-items:center}.event-title{grid-column:1/-1;text-align:center}.event-title p,.event-title h1,.machine-sign,.draw-button,.event-board strong,.prize-preview h2{font-family:"PixelFont","ArkPixel",monospace}.event-title p{color:#80d47d;font-size:12px}.event-title h1{font-size:34px;color:#fff;margin:9px}.event-title span{color:#a5b1c0}
.registration-entry{position:relative;border:5px solid #4d5d6b;background:#151d25;background-image:linear-gradient(90deg,rgba(111,197,121,.05) 1px,transparent 1px),linear-gradient(rgba(111,197,121,.04) 1px,transparent 1px);background-size:24px 24px;box-shadow:8px 8px 0 #050608,inset 0 0 0 4px #263541;padding:22px;margin-top:12px;overflow:hidden}.registration-entry:before{content:"";position:absolute;left:22px;right:22px;top:20px;height:4px;background:#263641;box-shadow:0 8px 0 #10161c}.registration-entry:after{content:"";position:absolute;right:22px;top:20px;width:42px;height:42px;border-top:4px solid #6fc579;border-right:4px solid #6fc579;opacity:.85}.registration-header{position:relative;z-index:1;display:flex;align-items:center;justify-content:space-between;padding:14px 0 18px;border-bottom:3px solid #30414c}.registration-label{display:flex;align-items:center;gap:12px;color:#f5f7f8;font:700 16px/1 "PixelFont","ArkPixel",monospace}.registration-label i{width:13px;height:13px;background:#78cf7c;box-shadow:3px 3px #2b5535}.registration-code{color:#8292a0;font:700 11px/1 "PixelFont","ArkPixel",monospace;letter-spacing:1px}.registration-body{position:relative;z-index:1;display:flex;align-items:center;gap:24px;padding:34px 10px 30px}.registration-emblem{position:relative;display:grid;flex:none;width:116px;height:116px;place-items:center;border:4px solid #6fc579;background:#1d2b32;box-shadow:6px 6px 0 #0a1014,inset 0 0 0 4px #2c4a4b}.registration-emblem i{position:absolute;width:15px;height:15px;background:#6fc579;box-shadow:3px 3px #294b32}.registration-emblem i:nth-child(1){left:12px;top:12px}.registration-emblem i:nth-child(2){right:12px;top:12px}.registration-emblem i:nth-child(3){left:12px;bottom:12px}.registration-emblem i:nth-child(4){right:12px;bottom:12px}.registration-emblem strong{color:#f4d776;font:700 27px/1 "PixelFont","ArkPixel",monospace;letter-spacing:1px}.registration-entry.registering .registration-emblem{animation:registration-pulse .7s steps(3) infinite}.registration-entry.registered .registration-emblem{border-color:#e2c663;background:#303a2c;box-shadow:6px 6px 0 #16190f,inset 0 0 0 4px #62552a}.registration-entry.registered .registration-emblem i{background:#e2c663;box-shadow:3px 3px #655321}.registration-copy{display:grid;align-content:center;gap:10px;min-width:0;flex:1}.registration-kicker{color:#7ed07e;font:700 11px/1 "PixelFont","ArkPixel",monospace;letter-spacing:1px;text-transform:uppercase}.registration-copy strong{color:#f6f8fa;font:700 25px/1.3 "PixelFont","ArkPixel",monospace}.registration-copy span{color:#aab6c0;font-size:14px;line-height:1.75}.registration-progress{display:flex;gap:5px;height:14px;border:3px solid #3c5260;background:#1a252c;padding:2px;margin-top:8px}.registration-progress i{flex:1;background:#385149;box-shadow:2px 0 0 #172229}.registration-progress i:nth-child(-n+4){background:#5f9d68}.registration-entry.registering .registration-progress i{animation:registration-progress .65s steps(4) infinite}.registration-entry.registered .registration-progress i{background:#c4a840}.registration-footer{position:relative;z-index:1;display:flex;align-items:center;justify-content:space-between;gap:18px;border-top:3px solid #30414c;padding:18px 0 0}.registration-status{display:flex;align-items:center;gap:10px}.registration-status__item{display:flex;align-items:center;gap:7px;color:#758592;font:700 11px/1 "PixelFont","ArkPixel",monospace;white-space:nowrap}.registration-status__item i{width:8px;height:8px;background:#40515c;box-shadow:2px 2px #1b272e}.registration-status__item.is-active{color:#d9e7db}.registration-status__item.is-active i,.registration-status__item.complete i{background:#78cf7c;box-shadow:2px 2px #2b5535}.registration-status__connector{width:30px;height:3px;background:#3c5159}.registration-entry .draw-button{background:#4b8652;border-color:#244d2a}.registration-entry.registered .draw-button{background:#9b8332;border-color:#5a4b1d}.registration-entry .draw-button:disabled{filter:none;opacity:.58}
.lottery-layout{grid-template-columns:minmax(0,1.35fr) minmax(250px,.65fr)}.registration-entry.registered .registration-emblem{border-color:#78cf7c;background:#263a31;box-shadow:6px 6px 0 #0d1712,inset 0 0 0 4px #31553b}.registration-entry.registered .registration-emblem i{background:#78cf7c;box-shadow:3px 3px #2b5535}.registration-entry.registered .registration-progress i{background:#5f9d68}.registration-entry.registered .draw-button{background:#315e67;border-color:#1d3d45}.registration-entry.won{border-color:#a68a3f;box-shadow:8px 8px 0 #050608,inset 0 0 0 4px #4f4528}.registration-entry.won:after{border-color:#e2c663}.registration-entry.won .registration-label i,.registration-entry.won .registration-emblem i{background:#e2c663;box-shadow:3px 3px #655321}.registration-entry.won .registration-emblem{border-color:#e2c663;background:#3c3625;box-shadow:6px 6px 0 #17130a,inset 0 0 0 4px #65552b}.registration-entry.won .registration-kicker,.registration-entry.won .registration-copy strong{color:#f1d477}.registration-entry.won .registration-progress i{background:#c4a840}.registration-entry.won .draw-button{background:#9b8332;border-color:#5a4b1d}.registration-entry.lost{border-color:#6b5962;box-shadow:8px 8px 0 #050608,inset 0 0 0 4px #3b3038}.registration-entry.lost:after{border-color:#b8858e}.registration-entry.lost .registration-label i,.registration-entry.lost .registration-emblem i{background:#b8858e;box-shadow:3px 3px #51343b}.registration-entry.lost .registration-emblem{border-color:#b8858e;background:#30272d;box-shadow:6px 6px 0 #120d11,inset 0 0 0 4px #4f3840}.registration-entry.lost .registration-kicker,.registration-entry.lost .registration-copy strong{color:#e0a9b0}.registration-entry.lost .registration-progress i{background:#765963}.registration-entry.lost .draw-button{background:#5d4b54;border-color:#3c3036}.registration-entry.resolved .registration-progress i{opacity:1}
.lottery-machine{position:relative;border:5px solid #657386;background:#1b212a;box-shadow:8px 8px 0 #050608,inset 0 0 0 4px #303947;padding:24px;margin-top:12px}.machine-lights{display:flex;justify-content:space-between;margin-bottom:14px}.machine-lights i{width:9px;height:9px;background:#6fc579;box-shadow:2px 2px #23472a}.drawing .machine-lights i{animation:blink .3s steps(1) infinite}.machine-lights i:nth-child(even){animation-delay:.15s}.machine-sign{text-align:center;background:#12161c;border:3px solid #3e4a5b;color:#f0d77b;padding:11px;font-size:19px;box-shadow:4px 4px #090b0e}.reel-window{height:260px;margin:20px 10px;border:6px solid #0a0c10;background:#e9dec3;box-shadow:inset 7px 7px #b8aa8d;display:grid;place-items:center;overflow:hidden;color:#191b1f}.idle-symbol,.result-symbol{display:grid;place-items:center}.idle-symbol span,.result-symbol>span{font:700 100px/1 "PixelFont","ArkPixel",monospace}.idle-symbol small{font:700 13px monospace;margin-top:16px}.reel-track{display:flex;flex-direction:column;gap:18px;font:700 70px/1 "PixelFont","ArkPixel",monospace;animation:reel .4s linear infinite}.result-symbol strong{font-size:30px;color:#274629;margin-top:8px}.result-symbol.lost strong{color:#555f6c}.machine-base{display:flex;justify-content:center;position:relative}.draw-button{display:inline-block;border:4px solid #273827;background:#56834c;color:#fff;padding:15px 28px;box-shadow:5px 5px #070908;font-size:16px;text-decoration:none}.draw-button:not(:disabled):hover{transform:translate(-2px,-2px);box-shadow:7px 7px #070908}.draw-button:disabled{filter:grayscale(.6);opacity:.7}.lever{position:absolute;right:-47px;bottom:16px;width:35px;height:75px;border:5px solid #11161c;background:#566171;transform-origin:bottom;transition:transform .35s steps(4)}.lever i{position:absolute;width:28px;height:28px;left:-1px;top:-22px;background:#79ca75;border:4px solid #244027}.lever.pulled{transform:rotate(140deg)}
.event-board{border:4px solid #4b5869;background:#151a21;box-shadow:6px 6px #050608;padding:20px}.board-stat{display:flex;justify-content:space-between;align-items:baseline;padding:15px 0;border-bottom:3px solid #303a47}.board-stat strong{font-size:28px;color:#84d581}.board-stat span{color:#8f9aaa}.deadline{display:grid;gap:6px;padding:17px 0}.deadline span{color:#7f8b99}.deadline strong{font-size:13px}.prize-preview{border-top:3px solid #303a47;padding-top:16px}.prize-preview h2{font-size:15px;color:#f1d982}.prize-preview div{display:grid;gap:4px;padding:10px 0;border-bottom:2px solid #29323d}.prize-preview div span{color:#a5afbc}.prize-preview div strong{color:#84d581}.rule{color:#687586;font-size:12px;line-height:1.7;margin-top:15px}.loading-machine,.result-panel{max-width:560px;margin:80px auto;text-align:center}.result-panel h1{font-family:"PixelFont","ArkPixel",monospace}.pixel-face{font:700 70px monospace}
.pixel-fireworks{position:absolute;inset:-30px;z-index:5;overflow:visible;pointer-events:none}.pixel-firework{position:absolute;left:18%;top:30%;width:1px;height:1px}.pixel-firework:nth-child(2){left:80%;top:22%}.pixel-firework:nth-child(3){left:28%;top:76%}.pixel-firework:nth-child(4){left:76%;top:70%}.pixel-firework i{--angle:calc(var(--n)*20deg);--distance:clamp(72px,10vw,138px);position:absolute;left:0;top:0;width:8px;height:8px;background:#79d27d;box-shadow:3px 3px 0 rgba(4,6,8,.5);opacity:0;animation:firework-particle 2.2s steps(9) infinite;transform:rotate(var(--angle)) translateX(0) scale(0)}.pixel-firework:nth-child(2) i{animation-delay:-.55s}.pixel-firework:nth-child(3) i{animation-delay:-1.1s}.pixel-firework:nth-child(4) i{animation-delay:-1.65s}.pixel-firework i:nth-child(3n){background:#f2d56d}.pixel-firework i:nth-child(3n+1){background:#a993e8}.pixel-firework i:nth-child(4n){width:6px;height:6px;background:#e98585}
@keyframes reel{to{transform:translateY(-176px)}}@keyframes blink{50%{background:#f1d665;box-shadow:2px 2px #6b5720}}@keyframes registration-pulse{50%{transform:translateY(-3px);box-shadow:6px 9px 0 #0a1014,inset 0 0 0 4px #2c4a4b}}@keyframes registration-progress{0%{filter:brightness(.75)}50%{filter:brightness(1.35)}100%{filter:brightness(.85)}}@keyframes firework-particle{0%,12%{opacity:0;transform:rotate(var(--angle)) translateX(0) scale(0)}18%{opacity:1;transform:rotate(var(--angle)) translateX(10px) scale(1)}66%{opacity:1}82%,100%{opacity:0;transform:rotate(var(--angle)) translateX(var(--distance)) scale(1)}}
.draw-error{margin:14px auto 0;border:3px solid #8c4d57;background:#321e25;padding:10px 12px;color:#ffb9c0;text-align:center;font:700 12px/1.5 "PixelFont","ArkPixel",monospace}
.rule-gate{color:#e7d38b}
@media(max-width:820px){.lottery-world{padding:20px 14px}.lottery-layout{grid-template-columns:1fr}.event-title h1{font-size:25px}.registration-body{flex-direction:column;align-items:flex-start;gap:22px;padding:28px 4px 24px}.registration-emblem{width:88px;height:88px}.registration-emblem i{width:11px;height:11px}.registration-emblem i:nth-child(1){left:9px;top:9px}.registration-emblem i:nth-child(2){right:9px;top:9px}.registration-emblem i:nth-child(3){left:9px;bottom:9px}.registration-emblem i:nth-child(4){right:9px;bottom:9px}.registration-emblem strong{font-size:22px}.registration-copy strong{font-size:20px}.registration-footer{align-items:stretch;flex-direction:column}.registration-status{justify-content:flex-start}.registration-entry .draw-button{width:100%}.reel-window{height:220px}.lever{right:-20px}.event-board{margin-top:5px}}
.registration-entry.closed{border-color:#8a733d;box-shadow:8px 8px 0 #050608,inset 0 0 0 4px #4e432a}.registration-entry.closed:after{border-color:#d0ae5a}.registration-entry.closed .registration-label i{background:#d0ae5a;box-shadow:3px 3px #665223}.registration-entry.closed .registration-emblem{border-color:#8e7a45;background:#292b27;box-shadow:6px 6px 0 #15140d,inset 0 0 0 4px #55492d}.registration-entry.closed .registration-emblem i{background:#d0ae5a;box-shadow:3px 3px #665223}.registration-entry.closed .registration-kicker,.registration-entry.closed .registration-copy strong{color:#e0c878}
</style>
