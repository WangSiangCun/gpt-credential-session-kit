<template>
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <div v-else class="pool-landing">
    <header class="pool-landing__nav">
      <router-link to="/home" class="pool-landing__brand" aria-label="PoolAI home">
        <span class="pool-landing__brand-mark">
          <img :src="siteLogo || '/poolai-logo.svg'" alt="" class="pool-landing__brand-logo" />
        </span>
        <span>{{ siteName }}</span>
      </router-link>

      <nav class="pool-landing__links" aria-label="Main navigation">
        <router-link to="/home">首页</router-link>
        <router-link to="/key-usage">用量</router-link>
        <router-link v-if="showModelPlazaEntry" to="/model-plaza">{{ t('nav.modelPlaza') }}</router-link>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">文档</a>
        <router-link to="/legal">协议</router-link>
      </nav>

      <div class="pool-landing__profile-actions">
        <QQGroupButton />
        <button type="button" class="pool-landing__notice" aria-label="通知">
          <Icon name="bell" size="sm" />
        </button>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="pool-landing__avatar"
          :class="{ 'has-notification': !isAuthenticated }"
          :aria-label="isAuthenticated ? '进入控制台' : '登录'"
        >
          {{ userInitial || 'A' }}
        </router-link>
      </div>
    </header>

    <main class="pool-landing__shell">
      <div class="pool-landing__system-banner">
        <span>POOLAI ROUTING CORE / AI API 中转与算力协作</span>
        <b><i></i> 3 个协作节点在线</b>
      </div>
      <section class="pool-landing__copy">
        <h1>统一接入，<br>按量计费的<br>AI API 服务</h1>
        <p class="pool-landing__description">
          多渠道 AI API 接入、用量计费与资源调度。充值即可使用平台密钥，资源方也能贡献账号获得收益。
        </p>
        <div class="pool-landing__actions">
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="pool-landing__primary">
            立即开始 <span>></span>
          </router-link>
          <router-link to="/key-usage" class="pool-landing__secondary">
            查看用量 <span>></span>
          </router-link>
        </div>
        <div class="pool-landing__pipeline" aria-label="接入流程">
          <span><b>01</b>统一密钥</span>
          <i aria-hidden="true"></i>
          <span><b>02</b>智能路由</span>
          <i aria-hidden="true"></i>
          <span><b>03</b>用量可见</span>
        </div>
        <div class="pool-landing__checks">
          <span>注册后可直接创建 API Key</span>
          <span>适合桌面端和手机端快速上手</span>
          <span>可购买 ChatGPT，也可贡献自己的资源</span>
        </div>
      </section>

      <section class="pool-landing__relay-window" aria-label="PoolAI 动物协作节点">
        <header class="relay-header">
          <div><i></i><span>ANIMAL RELAY CREW</span></div>
          <b>LIVE</b>
        </header>
        <div class="relay-map" aria-hidden="true">
          <span class="relay-line relay-line--one"></span>
          <span class="relay-line relay-line--two"></span>
          <i class="relay-packet relay-packet--one"></i>
          <i class="relay-packet relay-packet--two"></i>
        </div>

        <div class="relay-speech relay-speech--penguin">密钥已接收，正在检查接入状态</div>
        <div class="relay-speech relay-speech--fox">已为这次请求选择更合适的线路</div>
        <div class="relay-speech relay-speech--owl">延迟和用量都已记入控制台</div>

        <div class="relay-agents">
          <article class="relay-agent relay-agent--penguin">
            <div class="pixel-penguin pixel-animal">
              <i class="animal-part body"></i><i class="animal-part belly"></i>
              <i class="animal-part eye eye--left"></i><i class="animal-part eye eye--right"></i>
              <i class="animal-part beak"></i><i class="animal-part wing wing--left"></i><i class="animal-part wing wing--right"></i>
              <i class="animal-part foot foot--left"></i><i class="animal-part foot foot--right"></i>
            </div>
            <strong>接入企鹅</strong><span>ACCESS</span>
          </article>

          <article class="relay-agent relay-agent--fox">
            <div class="pixel-fox pixel-animal">
              <i class="animal-part tail"></i><i class="animal-part body"></i><i class="animal-part head"></i>
              <i class="animal-part ear ear--left"></i><i class="animal-part ear ear--right"></i>
              <i class="animal-part muzzle"></i><i class="animal-part eye eye--left"></i><i class="animal-part eye eye--right"></i>
              <i class="animal-part nose"></i><i class="animal-part foot foot--left"></i><i class="animal-part foot foot--right"></i>
            </div>
            <strong>路由狐狸</strong><span>ROUTING</span>
          </article>

          <article class="relay-agent relay-agent--owl">
            <div class="pixel-owl pixel-animal">
              <i class="animal-part body"></i><i class="animal-part brow"></i>
              <i class="animal-part eye eye--left"></i><i class="animal-part eye eye--right"></i>
              <i class="animal-part pupil pupil--left"></i><i class="animal-part pupil pupil--right"></i>
              <i class="animal-part beak"></i><i class="animal-part wing wing--left"></i><i class="animal-part wing wing--right"></i>
              <i class="animal-part foot foot--left"></i><i class="animal-part foot foot--right"></i>
            </div>
            <strong>监控猫头鹰</strong><span>MONITOR</span>
          </article>
        </div>
      </section>

    </main>

    <section
      ref="snakeZone"
      class="pool-landing__snake-zone"
      aria-label="PoolAI 像素蛇互动区"
    >
      <span class="pool-landing__snake-grass pool-landing__snake-grass--one"></span>
      <span class="pool-landing__snake-grass pool-landing__snake-grass--two"></span>
      <span class="pool-landing__snake-rock pool-landing__snake-rock--one"></span>
      <span class="pool-landing__snake-rock pool-landing__snake-rock--two"></span>
      <div class="pool-landing__snake-hole" aria-hidden="true"><i></i><i></i></div>
      <div
        v-for="(segment, index) in visibleSnakeSegments"
        :key="index"
        class="pool-landing__snake-segment"
        :class="{ 'is-head': index === 0, 'is-wrapping': snakeWrapped }"
        :style="snakeSegmentStyle(segment, index)"
      >
        <template v-if="index === 0"><i></i><i></i><b></b></template>
      </div>
      <div v-if="snakeWrapped" class="pool-landing__snake-speech" :style="snakeSpeechStyle">来吧来吧~</div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAuthStore, useAppStore } from '@/stores'
import QQGroupButton from '@/components/common/QQGroupButton.vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const currentYear = computed(() => new Date().getFullYear())
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const showModelPlazaEntry = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (authStore.isAuthenticated || appStore.cachedPublicSettings?.model_plaza_require_auth !== true))
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'PoolAI')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const isHomeContentUrl = computed(() => /^https?:\/\//i.test(homeContent.value.trim()))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const userInitial = computed(() => authStore.user?.email?.charAt(0).toUpperCase() || '')

type SnakePoint = { x: number; y: number; angle: number }

const snakeZone = ref<HTMLElement | null>(null)
const snakeActive = ref(false)
const snakeWrapped = ref(false)
const snakeTarget = ref({ x: 76, y: 154 })
const snakeSegments = ref<SnakePoint[]>(Array.from({ length: 18 }, () => ({ x: 76, y: 154, angle: 0 })))
const visibleSnakeSegments = computed(() => (snakeActive.value || snakeWrapped.value ? snakeSegments.value : snakeSegments.value.slice(0, 1)))
const snakeSpeechStyle = computed(() => ({
  left: `${Math.round(snakeTarget.value.x)}px`,
  top: `${Math.round(snakeTarget.value.y - 66)}px`,
}))
let snakeFrame = 0
let snakeLastMovedAt = 0

const followSnake = (event: MouseEvent) => {
  const bounds = snakeZone.value?.getBoundingClientRect()
  if (!bounds) return

  snakeTarget.value = {
    x: Math.max(20, Math.min(bounds.width - 20, event.clientX - bounds.left)),
    y: event.clientY - bounds.top,
  }
  snakeActive.value = true
  snakeLastMovedAt = Date.now()
}

const snakeSegmentStyle = (segment: SnakePoint, index: number) => ({
  transform: `translate(${Math.round(segment.x)}px, ${Math.round(segment.y)}px) rotate(${Math.round(segment.angle)}deg)`,
  zIndex: snakeSegments.value.length - index,
})

const animateSnake = () => {
  const now = Date.now()
  const segmentLength = 16
  const head = snakeSegments.value[0]
  const target = snakeTarget.value
  const dx = target.x - head.x
  const dy = target.y - head.y
  const distance = Math.hypot(dx, dy)

  if (snakeActive.value && now - snakeLastMovedAt > 3200) {
    snakeActive.value = false
    snakeWrapped.value = false
    snakeTarget.value = { x: 76, y: 154 }
  }

  if (snakeActive.value && distance < 16) snakeWrapped.value = true
  if (snakeWrapped.value && distance > 68) snakeWrapped.value = false

  if (snakeWrapped.value) {
    const phase = now / 260
    snakeSegments.value = snakeSegments.value.map((_, index) => {
      const ring = 25 + (index % 5) * 5
      const angle = phase + index * 0.78
      return {
        x: target.x + Math.cos(angle) * ring,
        y: target.y + Math.sin(angle) * ring * 0.68,
        angle: (angle * 180) / Math.PI + 90,
      }
    })
  } else {
    const next = snakeSegments.value.map((segment) => ({ ...segment }))
    const speed = snakeActive.value ? 2.15 : 1.15
    const direction = Math.atan2(dy, dx)
    next[0].x += Math.cos(direction) * Math.min(speed, distance)
    next[0].y += Math.sin(direction) * Math.min(speed, distance)
    next[0].angle = (direction * 180) / Math.PI + 90

    for (let index = 1; index < next.length; index += 1) {
      const lead = next[index - 1]
      const current = next[index]
      const offsetX = lead.x - current.x
      const offsetY = lead.y - current.y
      const gap = Math.max(Math.hypot(offsetX, offsetY), 0.001)
      current.x = lead.x - (offsetX / gap) * segmentLength
      current.y = lead.y - (offsetY / gap) * segmentLength
      current.angle = (Math.atan2(offsetY, offsetX) * 180) / Math.PI + 90
    }
    snakeSegments.value = next
  }

  snakeFrame = requestAnimationFrame(animateSnake)
}

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()
  window.addEventListener('mousemove', followSnake, { passive: true })
  snakeFrame = requestAnimationFrame(animateSnake)
})

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', followSnake)
  cancelAnimationFrame(snakeFrame)
})
</script>

<style scoped>
.pool-landing {
  --landing-line: #211a2e;
  --landing-paper: #fff0c9;
  --landing-purple: #271b49;
  --landing-purple-light: #38245f;
  min-height: 100vh;
  overflow: hidden;
  color: var(--landing-paper);
  background: #cfba8e;
  font-family: Inter, Manrope, "PingFang SC", "Microsoft YaHei", sans-serif;
}

.pool-landing__nav {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  min-height: 94px;
  padding: 0 clamp(28px, 4vw, 72px);
  border-bottom: 5px solid #315529;
  background-color: #70a856;
  background-image:
    linear-gradient(rgba(255,255,255,.14) 50%, transparent 50%),
    linear-gradient(90deg, rgba(42,88,37,.2) 50%, transparent 50%);
  background-size: 14px 14px;
  box-shadow: 0 5px 0 rgba(54,67,36,.34);
}

.pool-landing__brand,
.pool-landing__links,
.pool-landing__avatar,
.pool-landing__actions,
.pool-landing__checks { display: flex; align-items: center; }
.pool-landing__brand {
  gap: 14px;
  color: #fff7dd;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace;
  font-size: 16px;
  text-decoration: none;
}
.pool-landing__brand-mark {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  overflow: hidden;
  border: 3px solid #fff0bd;
  background: #284d35;
  box-shadow: 3px 3px 0 #1e3526;
}
.pool-landing__brand-logo { display: block; width: 100%; height: 100%; image-rendering: pixelated; }
.pool-landing__links { gap: 42px; }
.pool-landing__links a {
  color: #fff8df;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace;
  font-size: 14px;
  text-decoration: none;
}
.pool-landing__links a:hover { color: #f4d77b; }
.pool-landing__avatar {
  position: relative;
  justify-self: end;
  display: grid;
  width: 54px;
  height: 54px;
  place-items: center;
  border: 4px solid #231a30;
  background: #7a47c7;
  box-shadow: 4px 4px 0 #3c255f;
  color: #fff8df;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace;
  text-decoration: none;
  animation: avatar-float 3s steps(3) infinite;
}
.pool-landing__avatar:hover { transform: scale(1.08); outline: 2px solid #a5dc78; outline-offset: 3px; filter: brightness(1.12); }
.pool-landing__avatar.has-notification::after {
  position: absolute;
  top: -7px;
  right: -7px;
  width: 10px;
  height: 10px;
  border: 2px solid #231a30;
  background: #ec665f;
  content: '';
  animation: avatar-notification 4s steps(4) infinite;
}
@keyframes avatar-float { 0%, 72%, 100% { transform: translateY(0); } 84% { transform: translateY(-4px); } }
@keyframes avatar-notification { 0%, 74%, 100% { transform: translateX(0); } 78% { transform: translateX(-3px); } 82% { transform: translateX(3px); } 86% { transform: translateX(-2px); } }

.pool-landing__shell {
  display: grid;
  grid-template-columns: minmax(420px, .9fr) minmax(480px, 1.1fr);
  gap: clamp(32px, 4vw, 64px);
  width: min(1900px, calc(100% - 80px));
  min-height: calc(100vh - 160px);
  margin: 46px auto 0;
  padding: clamp(44px, 4vw, 70px) clamp(34px, 3vw, 54px) 72px;
  border: 5px solid var(--landing-line);
  background-color: var(--landing-purple);
  background-image:
    linear-gradient(rgba(154,124,214,.09) 1px, transparent 1px),
    linear-gradient(90deg, rgba(154,124,214,.09) 1px, transparent 1px),
    linear-gradient(135deg, transparent 53%, rgba(119,79,180,.23) 53% 100%);
  background-size: 22px 22px, 22px 22px, 100% 100%;
  box-shadow: 8px 8px 0 rgba(63,48,37,.68);
}
.pool-landing__copy { align-self: center; max-width: 640px; }
.pool-landing__eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  margin: 0 0 38px;
  padding: 10px 14px;
  border: 3px solid #8267b1;
  background: #211736;
  color: #ffedbd;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace;
  font-size: 12px;
}
.pool-landing__eyebrow i { width: 9px; height: 9px; background: #8ece55; box-shadow: 2px 2px #477d31; }
.pool-landing__copy h1 {
  margin: 0;
  color: #fff1c6;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace;
  font-size: clamp(38px, 4.5vw, 76px);
  font-weight: 700;
  line-height: 1.35;
  letter-spacing: 0;
  text-shadow: 4px 4px 0 #160f28;
}
.pool-landing__description {
  max-width: 610px;
  margin: 28px 0 0;
  color: #ddd1ef;
  font-size: 17px;
  line-height: 1.75;
}
.pool-landing__actions { gap: 18px; margin-top: 52px; }
.pool-landing__actions a {
  display: inline-flex;
  align-items: center;
  gap: 16px;
  min-height: 58px;
  padding: 0 26px;
  border: 4px solid var(--landing-line);
  box-shadow: 5px 5px 0 rgba(16,11,26,.78);
  color: #fff9e8;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace;
  font-size: 14px;
  text-decoration: none;
}
.pool-landing__primary { background-color: #8d50de; background-image: repeating-linear-gradient(90deg, transparent 0 7px, rgba(255,255,255,.11) 7px 9px); }
.pool-landing__secondary { background: #ead3a1; color: #332637 !important; }
.pool-landing__actions a:hover { transform: translate(-1px,-1px); filter: brightness(1.08); }
.pool-landing__checks { flex-wrap: wrap; gap: 12px; margin-top: 28px; }
.pool-landing__checks span {
  padding: 10px 13px 10px 31px;
  border: 2px solid #735a9d;
  background: rgba(45,31,79,.82);
  color: #f5e8cf;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace;
  font-size: 11px;
  line-height: 1.2;
  position: relative;
}
.pool-landing__checks span::before { position: absolute; left: 11px; content: 'o'; color: #91cf5a; font-family: sans-serif; font-weight: 900; }

.pool-landing__cat-window {
  position: relative;
  align-self: start;
  min-height: 438px;
  overflow: hidden;
  border: 5px solid #160f26;
  background: #201735;
  box-shadow: 7px 7px 0 rgba(13,9,21,.78), inset 0 0 0 4px #382657;
}
.pool-landing__speech {
  position: absolute;
  z-index: 6;
  padding: 17px 22px;
  border: 4px solid #211a2e;
  background: #ffe49c;
  box-shadow: 4px 4px 0 rgba(11,8,19,.78);
  color: #2b2232;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace;
  font-size: 11px;
  opacity: 0;
  pointer-events: none;
  animation: speech-cycle 16s steps(1) infinite;
}
.pool-landing__speech::after { position: absolute; right: 28%; bottom: -12px; width: 18px; height: 18px; border-right: 4px solid #211a2e; border-bottom: 4px solid #211a2e; background: #ffe49c; content: ''; transform: rotate(45deg); }
.pool-landing__speech--cat { top: 28px; right: 58px; animation-delay: 0s; }
.pool-landing__speech--orange { bottom: 126px; left: 24px; animation-delay: -4s; }
.pool-landing__speech--key { right: 70px; bottom: 136px; animation-delay: -8s; }
.pool-landing__speech--node { right: 18px; bottom: 176px; animation-delay: -12s; }
@keyframes speech-cycle { 0%, 23% { opacity: 1; } 25%, 100% { opacity: 0; } }
.pool-landing__cat-scene { width: 100%; min-height: 438px; border: 0; box-shadow: none; background: transparent; }
.pool-landing__cat-window :deep(.pixel-auth-scene__grid),
.pool-landing__cat-window :deep(.pixel-auth-scene__brand),
.pool-landing__cat-window :deep(.pixel-auth-scene__copy) { display: none; }
.pool-landing__cat-window :deep(.pixel-auth-scene) { background: transparent; }
.pool-landing__cat-window :deep(.pixel-auth-cat) { bottom: 20px; transform: translateX(-50%) scale(.72); transform-origin: bottom center; animation: landing-cat-bob 3.4s steps(4) infinite; }
.pool-landing__cat-window :deep(.pixel-auth-cat__eyes b) { animation: landing-cat-eyes 3.4s steps(4) infinite; }
.pool-landing__cat-window :deep(.pixel-auth-crew) { inset: 0; z-index: 5; display: block; transform: none; }
.pool-landing__cat-window :deep(.pixel-auth-creature) { position: absolute; margin: 0; transform-origin: center bottom; }
.pool-landing__cat-window :deep(.pixel-auth-creature--orange) { bottom: 30px; left: 10%; transform: scale(.72); animation: landing-orange-sway 3.2s steps(4) infinite; }
.pool-landing__cat-window :deep(.pixel-auth-creature--black) { bottom: 76px; left: 66%; transform: scale(.68); animation: landing-key-turn 5s steps(8) infinite; }
.pool-landing__cat-window :deep(.pixel-auth-creature--purple) { right: 7%; bottom: 24px; left: auto; transform: scale(.72); animation: landing-node-bob 3.7s steps(4) infinite; }
@keyframes landing-cat-bob { 0%, 100% { transform: translateX(-50%) scale(.72) translateY(0); } 50% { transform: translateX(-50%) scale(.72) translateY(-7px); } }
@keyframes landing-cat-eyes { 0%, 25% { transform: translate(-3px, 0); } 50%, 70% { transform: translate(4px, 1px); } 100% { transform: translate(-3px, 0); } }
@keyframes landing-orange-sway { 0%, 100% { transform: scale(.72) rotate(-3deg); } 50% { transform: scale(.72) rotate(4deg) translateX(5px); } }
@keyframes landing-key-turn { 0%, 100% { transform: scale(.68) rotate(-4deg); } 50% { transform: scale(.68) rotate(5deg) translateY(-8px); } }
@keyframes landing-node-bob { 0%, 100% { transform: scale(.72) translateY(0); } 50% { transform: scale(.72) translateY(-11px); } }
.pool-landing__spark { position: absolute; z-index: 5; width: 12px; height: 12px; background: #8ece55; box-shadow: 14px 14px #537f3f; }
.pool-landing__spark--one { right: 28px; top: 27px; }
.pool-landing__spark--two { bottom: 35px; left: 45px; background: #d19631; box-shadow: 12px 12px #805d28; }

@media (max-width: 1060px) {
  .pool-landing__shell { grid-template-columns: 1fr; width: min(900px, calc(100% - 32px)); }
  .pool-landing__cat-window { max-width: 720px; width: 100%; justify-self: center; }
}
@media (max-width: 640px) {
  .pool-landing__nav { grid-template-columns: 1fr auto; min-height: 72px; padding: 0 18px; }
  .pool-landing__links { display: none; }
  .pool-landing__brand { font-size: 12px; }
  .pool-landing__brand-mark { width: 30px; height: 30px; }
  .pool-landing__avatar { width: 42px; height: 42px; }
  .pool-landing__shell { margin-top: 20px; padding: 46px 24px; }
  .pool-landing__copy h1 { font-size: 36px; }
  .pool-landing__description { font-size: 15px; }
  .pool-landing__speech { top: 18px; right: 14px; left: 14px; font-size: 10px; }
  .pool-landing__cat-window { min-height: 360px; }
  .pool-landing__cat-scene { min-height: 360px; }
  .pool-landing__cat-window :deep(.pixel-auth-cat) { transform: translateX(-50%) scale(.62); }
  .pool-landing__actions { align-items: stretch; flex-direction: column; }
}

/* System-home layer: aligns public product information with the dark console. */
.pool-landing {
  --system-bg: #12141a;
  --system-panel: #1e1b2e;
  --system-panel-alt: #20242c;
  --system-line: #394150;
  --system-purple: #8b5cf6;
  min-height: 100vh;
  overflow: auto;
  background-color: var(--system-bg);
  background-image: linear-gradient(rgba(115,94,163,.06) 1px, transparent 1px), linear-gradient(90deg, rgba(115,94,163,.06) 1px, transparent 1px);
  background-size: 22px 22px;
}
.pool-landing__nav {
  position: sticky;
  top: 0;
  z-index: 20;
  min-height: 72px;
  border-bottom: 1px solid #3d354d;
  background-color: #16121f;
  background-image: linear-gradient(90deg, #292044, #1c182c 54%, #101216);
  background-size: 100% 100%;
  box-shadow: 0 2px 0 rgba(0,0,0,.45);
}
.pool-landing__brand,
.pool-landing__brand span { font-family: 'PoolPixelCN', 'PoolPixelEN', monospace !important; }
.pool-landing__brand-mark { border-color: #715fa1; background: #252137; box-shadow: 2px 2px #0d0d13; }
.pool-landing__links { gap: 32px; }
.pool-landing__links a { color: #c4bfd4; font-family: Inter, Manrope, sans-serif !important; font-size: 13px; }
.pool-landing__links a:hover { color: #c7b5ff; }
.pool-landing__profile-actions { display: flex; align-items: center; justify-self: end; }
.pool-landing__notice {
  justify-self: end;
  display: grid;
  width: 34px;
  height: 34px;
  margin-right: 12px;
  place-items: center;
  border: 1px solid #4a4560;
  background: #242331;
  color: #c8c8d6;
  box-shadow: 2px 2px #0e1014;
}
.pool-landing__avatar { width: 42px; height: 42px; border-width: 2px; background: #6c4ab1; box-shadow: 2px 2px #231832; }
.pool-landing__shell {
  grid-template-columns: minmax(0, 1fr) minmax(430px, .9fr);
  gap: 28px;
  width: min(1360px, calc(100% - 48px));
  min-height: 0;
  margin: 28px auto 0;
  padding: 0;
  border: 0;
  background: transparent;
  box-shadow: none;
}
.pool-landing__system-banner {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 52px;
  padding: 0 18px;
  border: 1px solid var(--system-line);
  background: #1c2027;
  box-shadow: 2px 2px #0d0f13;
  color: #edeaf6;
  font-family: 'PoolPixelCN', 'PoolPixelEN', monospace !important;
  font-size: 12px;
}
.pool-landing__system-banner b { display: inline-flex; gap: 8px; align-items: center; color: #a9e5ae; font-family: Inter, sans-serif !important; font-size: 12px; font-weight: 600; }
.pool-landing__system-banner i { width: 7px; height: 7px; background: #6ec177; box-shadow: 1px 1px #3a8143; }
.pool-landing__copy {
  align-self: stretch;
  max-width: none;
  padding: clamp(30px, 4vw, 58px);
  border: 1px solid var(--system-line);
  background: var(--system-panel);
  box-shadow: 3px 3px #0e1014;
}
.pool-landing__eyebrow { margin-bottom: 28px; border-color: #4b4169; background: #252038; color: #c8b9fa; }
.pool-landing__copy h1 { font-size: clamp(34px, 4vw, 60px); line-height: 1.28; text-shadow: 3px 3px #111019; }
.pool-landing__description { max-width: 540px; color: #b8bdc9; font-family: Inter, Manrope, sans-serif !important; font-size: 15px; line-height: 1.8; }
.pool-landing__actions { margin-top: 34px; }
.pool-landing__actions a { min-height: 46px; border-width: 2px; box-shadow: 2px 2px #0e1014; }
.pool-landing__primary { background-color: var(--system-purple); background-image: none; }
.pool-landing__secondary { border-color: #48515b !important; background: #2a3038; color: #e2e5ea !important; }
.pool-landing__checks { display: none; }
.pool-landing__cat-window {
  align-self: stretch;
  min-height: 390px;
  border: 1px solid #444054;
  background-color: #1a1828;
  background-image:
    linear-gradient(rgba(139,92,246,.09) 1px, transparent 1px),
    linear-gradient(90deg, rgba(139,92,246,.09) 1px, transparent 1px),
    linear-gradient(135deg, rgba(70,51,108,.34) 0 18%, transparent 18% 76%, rgba(36,29,55,.72) 76% 100%);
  background-size: 22px 22px, 22px 22px, 100% 100%;
  box-shadow: 3px 3px #0e1014, inset 0 0 0 2px #29253a;
}
.pool-landing__cat-scene { min-height: 390px; }
.pool-landing__cat-window :deep(.pixel-auth-scene) {
  background-color: transparent;
  background-image:
    linear-gradient(rgba(174,148,233,.045) 1px, transparent 1px),
    linear-gradient(90deg, rgba(174,148,233,.045) 1px, transparent 1px);
  background-size: 11px 11px;
}
.pool-landing__speech {
  max-width: min(250px, calc(100% - 34px));
  border-width: 2px;
  box-shadow: 2px 2px #0d0d12;
  font-size: 10px;
  line-height: 1.55;
}
.pool-landing__speech--cat { top: 22px; right: auto; left: 50%; transform: translateX(-50%); }
.pool-landing__speech--cat::after { right: calc(50% - 9px); }
.pool-landing__speech--orange { top: 43%; bottom: auto; left: 3%; }
.pool-landing__speech--orange::after { right: auto; left: 30px; }
.pool-landing__speech--key { top: 14%; right: 16%; bottom: auto; }
.pool-landing__speech--key::after { right: auto; left: 104px; }
.pool-landing__speech--node { top: 24%; right: 1%; bottom: auto; }
.pool-landing__speech--node::after { right: 30px; }
.pool-landing__metrics {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}
.pool-landing__metrics article,
.pool-landing__features a {
  position: relative;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--system-line);
  background: var(--system-panel-alt);
  box-shadow: 2px 2px #0d0f13;
}
.pool-landing__metrics article { min-height: 118px; padding: 18px; }
.pool-landing__metrics span { color: #aeb3c0; font-family: Inter, sans-serif !important; font-size: 12px; }
.pool-landing__metrics strong { margin-top: 10px; color: #f0ecff; font-family: 'PoolPixelCN', 'PoolPixelEN', monospace !important; font-size: 24px; }
.pool-landing__metrics small { margin-top: auto; color: #6ec177; font-family: Inter, sans-serif !important; font-size: 11px; }
.pool-landing__features {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  width: min(1360px, calc(100% - 48px));
  margin: 18px auto 42px;
}
.pool-landing__features a { min-height: 118px; padding: 20px; color: #e9ebef; text-decoration: none; }
.pool-landing__features b { font-family: 'PoolPixelCN', 'PoolPixelEN', monospace !important; font-size: 13px; }
.pool-landing__features span { max-width: 25ch; margin-top: 10px; color: #aeb3c0; font-family: Inter, sans-serif !important; font-size: 12px; line-height: 1.5; }
.pool-landing__features i { position: absolute; right: 18px; bottom: 16px; color: #a78bfa; font-family: 'PoolPixelCN', 'PoolPixelEN', monospace !important; font-style: normal; }
.pool-landing__features a:hover { border-color: #74629f; background: #282539; transform: translate(-1px,-1px); }
.pool-landing__snake-zone {
  position: relative;
  width: min(1360px, calc(100% - 48px));
  height: 210px;
  margin: 18px auto 42px;
  overflow: visible;
  z-index: 10;
  border: 1px solid #454054;
  background-color: #171721;
  background-image:
    linear-gradient(rgba(127,98,184,.07) 1px, transparent 1px),
    linear-gradient(90deg, rgba(127,98,184,.07) 1px, transparent 1px),
    linear-gradient(180deg, transparent 0 67%, #233126 67% 76%, #4a382a 76% 100%);
  background-size: 20px 20px, 20px 20px, 100% 100%;
  box-shadow: 3px 3px #0c0e12, inset 0 0 0 2px #211e30;
  cursor: crosshair;
}
.pool-landing__snake-zone::before,
.pool-landing__snake-zone::after { position: absolute; right: 0; bottom: 0; width: 132px; height: 48px; background-image: repeating-linear-gradient(90deg, #4a382a 0 17px, #513d2d 17px 34px); content: ''; opacity: .9; }
.pool-landing__snake-zone::after { right: auto; left: 178px; width: 94px; opacity: .66; }
.pool-landing__snake-hole { position: absolute; z-index: 1; bottom: 21px; left: 35px; width: 88px; height: 47px; background: #282019; box-shadow: inset 0 8px #16131a, 0 0 0 5px #4f3829, -12px 0 #31442b, 12px -7px #3d5432; }
.pool-landing__snake-hole i { position: absolute; width: 11px; height: 9px; background: #725039; }
.pool-landing__snake-hole i:first-child { top: -12px; left: 8px; }
.pool-landing__snake-hole i:last-child { top: -19px; right: 13px; background: #426134; }
.pool-landing__snake-grass,
.pool-landing__snake-rock { position: absolute; z-index: 1; display: block; }
.pool-landing__snake-grass { width: 13px; height: 24px; background: #53793f; box-shadow: 12px 7px #38572f, 24px 0 #628a4a; }
.pool-landing__snake-grass--one { bottom: 62px; left: 144px; }
.pool-landing__snake-grass--two { right: 100px; bottom: 68px; transform: scale(.75); }
.pool-landing__snake-rock { width: 19px; height: 14px; background: #4b4658; box-shadow: 14px 7px #363343; }
.pool-landing__snake-rock--one { right: 235px; bottom: 33px; }
.pool-landing__snake-rock--two { right: 45px; bottom: 60px; transform: scale(.7); }
.pool-landing__snake-segment { position: absolute; top: -10px; left: -10px; width: 20px; height: 20px; transform-origin: center; background: #7250bd; box-shadow: inset 4px 4px #9775e5, inset -4px -4px #4b337d; transition: filter .12s linear; will-change: transform; }
.pool-landing__snake-segment:nth-of-type(2n) { background: #6846af; }
.pool-landing__snake-segment.is-head { width: 36px; height: 32px; top: -16px; left: -18px; background: #8460d2; box-shadow: inset 5px 5px #a887f1, inset -5px -5px #513789; }
.pool-landing__snake-segment.is-head::before { position: absolute; top: -7px; left: 6px; width: 8px; height: 8px; background: #8460d2; box-shadow: 16px 0 #8460d2; content: ''; }
.pool-landing__snake-segment.is-head i { position: absolute; top: 8px; width: 7px; height: 8px; background: #fff4cf; box-shadow: 1px 1px #332942; }
.pool-landing__snake-segment.is-head i:first-child { left: 7px; }
.pool-landing__snake-segment.is-head i:nth-child(2) { right: 7px; }
.pool-landing__snake-segment.is-head i::after { position: absolute; bottom: 1px; left: 2px; width: 3px; height: 4px; background: #26212d; content: ''; animation: snake-blink 4s steps(1) infinite; }
.pool-landing__snake-segment.is-head b { position: absolute; bottom: 5px; left: 15px; width: 6px; height: 3px; background: #e6c1e9; }
.pool-landing__snake-segment.is-wrapping { filter: brightness(1.16); }
@keyframes snake-blink { 0%, 89%, 100% { height: 4px; } 92%, 96% { height: 1px; } }
.pool-landing__snake-speech { position: absolute; z-index: 30; padding: 12px 18px; border: 2px solid #201927; background: #ffe49c; box-shadow: 2px 2px #0d0d12; color: #2d2434; font-family: 'PoolPixelCN', 'PoolPixelEN', monospace !important; font-size: 11px; pointer-events: none; transform: translateX(-50%); animation: snake-speech-in .18s steps(2); }
.pool-landing__snake-speech::after { position: absolute; bottom: -8px; left: calc(50% - 7px); width: 12px; height: 12px; border-right: 2px solid #201927; border-bottom: 2px solid #201927; background: #ffe49c; content: ''; transform: rotate(45deg); }
@keyframes snake-speech-in { from { opacity: 0; transform: translate(-50%, 5px); } to { opacity: 1; transform: translate(-50%, 0); } }

/* Homepage relay scene: deliberately distinct from the login cat crew. */
.pool-landing__copy{position:relative;overflow:hidden}.pool-landing__copy:after{position:absolute;right:-2px;bottom:-2px;width:54px;height:54px;border-top:4px solid #654f91;border-left:4px solid #654f91;background:#171522;content:'';clip-path:polygon(100% 0,100% 100%,0 100%)}
.pool-landing__pipeline{display:flex;align-items:center;gap:10px;margin-top:31px;color:#c8ced7;font:700 10px/1.4 'PoolPixelCN','PoolPixelEN',monospace}.pool-landing__pipeline span{display:flex;align-items:center;gap:7px}.pool-landing__pipeline b{display:grid;width:25px;height:25px;place-items:center;border:2px solid #5c4b7c;background:#282239;color:#a991dc;font-size:8px}.pool-landing__pipeline i{width:24px;height:2px;background:#475260;box-shadow:8px 0 #6b4ca4}
.pool-landing__relay-window{position:relative;align-self:stretch;min-height:440px;overflow:hidden;border:4px solid #424c5a;background-color:#11161c;background-image:linear-gradient(rgba(119,92,174,.08) 2px,transparent 2px),linear-gradient(90deg,rgba(119,92,174,.08) 2px,transparent 2px),linear-gradient(180deg,transparent 0 75%,#18221c 75% 81%,#252126 81% 100%);background-size:24px 24px,24px 24px,100% 100%;box-shadow:6px 6px 0 #07090c,inset 3px 3px 0 #202733}
.relay-header{position:absolute;z-index:8;top:0;right:0;left:0;display:flex;min-height:52px;align-items:center;justify-content:space-between;border-bottom:3px solid #3d4755;background:#181e26;padding:0 17px}.relay-header div{display:flex;align-items:center;gap:9px}.relay-header i{width:9px;height:9px;background:#77ce79;box-shadow:3px 3px #315936}.relay-header span,.relay-header b{color:#dce4ec;font:700 10px/1 'PoolPixelCN','PoolPixelEN',monospace}.relay-header b{border:2px solid #477c4c;background:#1c2a21;padding:5px 7px;color:#8be190}
.relay-map{position:absolute;z-index:1;inset:52px 0 0;pointer-events:none}.relay-line{position:absolute;top:59%;height:4px;background:#394653;box-shadow:0 4px #0b0e12}.relay-line:after{position:absolute;top:-6px;width:12px;height:12px;border:3px solid #58697a;background:#161b22;content:''}.relay-line--one{left:21%;width:29%}.relay-line--one:after{right:-5px}.relay-line--two{right:20%;width:27%}.relay-line--two:after{left:-5px}.relay-packet{position:absolute;top:calc(59% - 3px);z-index:2;width:9px;height:9px;background:#9c75db;box-shadow:2px 2px #3d295c;animation:packet-run-one 3.2s steps(10) infinite}.relay-packet--one{left:22%}.relay-packet--two{right:22%;background:#78ce7a;box-shadow:2px 2px #315936;animation-name:packet-run-two;animation-delay:-1.2s}@keyframes packet-run-one{to{transform:translateX(175px)}}@keyframes packet-run-two{to{transform:translateX(-165px)}}
.relay-speech{position:absolute;z-index:10;bottom:205px;max-width:220px;border:3px solid #202630;background:#f5dda0;padding:12px 14px;color:#25222b;font:700 10px/1.65 'PoolPixelCN','PoolPixelEN',monospace;box-shadow:4px 4px #07090c;opacity:0;animation:relay-speech-cycle 12s steps(1) infinite}.relay-speech:after{position:absolute;bottom:-10px;width:14px;height:14px;border-right:3px solid #202630;border-bottom:3px solid #202630;background:#f5dda0;content:'';transform:rotate(45deg)}.relay-speech--penguin{left:2%;animation-delay:0s}.relay-speech--penguin:after{left:38%}.relay-speech--fox{left:50%;transform:translateX(-50%);animation-delay:4s}.relay-speech--fox:after{left:calc(50% - 7px)}.relay-speech--owl{right:2%;animation-delay:8s}.relay-speech--owl:after{right:38%}@keyframes relay-speech-cycle{0%,28%{opacity:1}31%,100%{opacity:0}}
.relay-agents{position:absolute;z-index:5;right:0;bottom:35px;left:0;display:grid;grid-template-columns:repeat(3,1fr);align-items:end;padding:0 20px}.relay-agent{display:grid;justify-items:center;gap:5px;animation:agent-idle 3s steps(4) infinite}.relay-agent--fox{animation-delay:-1s}.relay-agent--owl{animation-delay:-2s}.relay-agent strong{color:#f0e7cf;font:700 10px/1.2 'PoolPixelCN','PoolPixelEN',monospace}.relay-agent>span{color:#667484;font:700 8px/1 monospace}@keyframes agent-idle{50%{transform:translateY(-5px)}}
.pixel-animal{position:relative;width:96px;height:128px;image-rendering:pixelated}.animal-part{position:absolute;display:block}
.pixel-penguin .body{top:19px;left:20px;width:56px;height:85px;background:#252d37;box-shadow:inset 7px 0 #3c4754,inset -7px 0 #10151b,0 6px #11161b}.pixel-penguin .belly{top:47px;left:32px;width:32px;height:50px;background:#e9e5d5;box-shadow:inset 5px 0 #f6f1dc}.pixel-penguin .eye{top:34px;width:8px;height:10px;background:#f7f1d9;box-shadow:inset 3px -5px #101319}.pixel-penguin .eye--left{left:31px}.pixel-penguin .eye--right{right:31px}.pixel-penguin .beak{top:49px;left:43px;width:12px;height:7px;background:#e7a54b;box-shadow:3px 3px #8a5525}.pixel-penguin .wing{top:50px;width:13px;height:40px;background:#1a2028}.pixel-penguin .wing--left{left:8px;transform:rotate(8deg)}.pixel-penguin .wing--right{right:8px;transform:rotate(-8deg)}.pixel-penguin .foot{top:106px;width:22px;height:8px;background:#e4a149}.pixel-penguin .foot--left{left:16px}.pixel-penguin .foot--right{right:16px}
.pixel-fox .tail{top:68px;left:2px;width:42px;height:22px;background:#cc6f3c;box-shadow:-8px 8px #e9d1aa,0 7px #7e3e28}.pixel-fox .body{top:60px;left:29px;width:52px;height:47px;background:#c96837;box-shadow:inset 6px 0 #dc7b45,inset -6px 0 #8b432b}.pixel-fox .head{top:22px;left:23px;width:58px;height:48px;background:#d6723e;box-shadow:inset 7px 0 #e5854b,inset -7px 0 #93462c}.pixel-fox .ear{top:5px;width:21px;height:25px;background:#c76538;box-shadow:inset 5px 5px #4b2b31}.pixel-fox .ear--left{left:23px}.pixel-fox .ear--right{right:15px}.pixel-fox .muzzle{top:48px;left:35px;width:36px;height:22px;background:#ead2ad;box-shadow:inset 5px 0 #f4dfbd}.pixel-fox .eye{top:37px;width:7px;height:8px;background:#25202a}.pixel-fox .eye--left{left:35px}.pixel-fox .eye--right{right:26px}.pixel-fox .nose{top:54px;left:49px;width:9px;height:7px;background:#25202a}.pixel-fox .foot{top:104px;width:18px;height:10px;background:#6e3829}.pixel-fox .foot--left{left:27px}.pixel-fox .foot--right{right:16px}
.pixel-owl .body{top:27px;left:17px;width:65px;height:80px;background:#67516d;box-shadow:inset 7px 0 #7d6583,inset -7px 0 #46364e,0 7px #2e2734}.pixel-owl .brow{top:14px;left:17px;width:65px;height:25px;background:#59455f;box-shadow:-7px -7px #59455f,7px -7px #59455f}.pixel-owl .eye{top:35px;width:25px;height:25px;background:#eee1b8;box-shadow:inset 4px 4px #fff1c8}.pixel-owl .eye--left{left:22px}.pixel-owl .eye--right{right:19px}.pixel-owl .pupil{top:43px;width:9px;height:11px;background:#1c2027}.pixel-owl .pupil--left{left:31px}.pixel-owl .pupil--right{right:27px}.pixel-owl .beak{top:61px;left:45px;width:10px;height:10px;background:#d9a04f;box-shadow:0 5px #885b2d}.pixel-owl .wing{top:65px;width:14px;height:34px;background:#4c3c55}.pixel-owl .wing--left{left:6px}.pixel-owl .wing--right{right:4px}.pixel-owl .foot{top:108px;width:18px;height:7px;background:#bd8644}.pixel-owl .foot--left{left:23px}.pixel-owl .foot--right{right:19px}
@media (max-width: 900px) {
  .pool-landing__shell { grid-template-columns: 1fr; }
  .pool-landing__metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .pool-landing__relay-window{min-height:450px}
}
@media (max-width: 640px) {
  .pool-landing__notice { display: none; }
  .pool-landing__metrics,
  .pool-landing__features { grid-template-columns: 1fr; }
  .pool-landing__shell,
  .pool-landing__features { width: min(100% - 24px, 1360px); }
  .pool-landing__system-banner span{max-width:65%;font-size:9px;line-height:1.5}.pool-landing__system-banner b{font-size:9px!important}.pool-landing__pipeline{flex-wrap:wrap}.pool-landing__pipeline i{width:14px}.relay-agents{padding:0 2px}.pixel-animal{transform:scale(.78);transform-origin:bottom center}.relay-agent{gap:0}.relay-agent strong{font-size:8px}.relay-speech{right:12px!important;bottom:172px;left:12px!important;max-width:none;transform:none!important}.relay-speech--penguin:after{right:auto!important;left:calc(16.667% - 7px)!important}.relay-speech--fox:after{right:auto!important;left:calc(50% - 7px)!important}.relay-speech--owl:after{right:auto!important;left:calc(83.333% - 7px)!important}
}
</style>
