<template>
  <div class="pixel-auth-layout">
    <div class="pixel-auth-layout__topbar">
      <router-link to="/home" class="pixel-auth-layout__wordmark">
        <span class="pixel-auth-layout__wordmark-mark">+</span>
        <span>{{ siteName }}</span>
      </router-link>
      <span class="pixel-auth-layout__status">SYSTEM // ONLINE</span>
    </div>

    <div class="pixel-auth-layout__grid">
      <PixelAuthScene :focus="focus" :mode="mode" />

      <main class="pixel-auth-layout__panel">
        <div class="pixel-auth-layout__panel-head">
          <div class="pixel-auth-layout__logo">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" />
          </div>
          <div>
            <div class="pixel-auth-layout__eyebrow">TOKEN NETWORK</div>
            <div class="pixel-auth-layout__subtitle">{{ siteSubtitle }}</div>
          </div>
        </div>

        <div class="pixel-auth-layout__content">
          <slot />
        </div>

        <div class="pixel-auth-layout__footer">
          <slot name="footer" />
        </div>
      </main>
    </div>

    <div class="pixel-auth-layout__copyright">
      &copy; {{ currentYear }} {{ siteName }} // ALL RIGHTS RESERVED
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import PixelAuthScene from '@/components/auth/PixelAuthScene.vue'

const props = withDefaults(defineProps<{
  focus?: 'email' | 'password' | null
  mode?: 'login' | 'register'
}>(), {
  focus: null,
  mode: 'login'
})

const focus = computed(() => props.focus)

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.pixel-auth-layout {
  --pixel-cn: var(--pool-pixel-en);
  min-height: 100vh;
  overflow: hidden;
  padding: 24px clamp(16px, 4vw, 72px) 18px;
  color: #f8f0d2;
  background: #14161b;
  background-image:
    radial-gradient(circle at 8% 4%, rgba(112, 188, 219, .16), transparent 31%),
    radial-gradient(circle at 94% 93%, rgba(72, 121, 177, .15), transparent 34%);
  border: 0;
  box-shadow: none;
}

.pixel-auth-layout__topbar,
.pixel-auth-layout__copyright {
  width: min(1400px, 100%);
  margin: 0 auto;
  font: 700 10px/1 var(--pixel-cn), monospace;
  letter-spacing: .12em;
}

.pixel-auth-layout__topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 32px;
  margin-bottom: 20px;
}

.pixel-auth-layout__wordmark {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: #fff5cf;
  text-decoration: none;
}

.pixel-auth-layout__wordmark-mark {
  display: grid;
  width: 20px;
  height: 20px;
  place-items: center;
  color: #261c16;
  background: #f4ad28;
  box-shadow: 3px 3px 0 #8f5018;
  font-size: 14px;
}

.pixel-auth-layout__status,
.pixel-auth-layout__eyebrow {
  color: #a7bb74;
}

.pixel-auth-layout__grid {
  display: grid;
  grid-template-columns: minmax(0, 1.28fr) minmax(380px, .72fr);
  width: min(1400px, 100%);
  min-height: min(760px, calc(100vh - 112px));
  margin: 0 auto;
  gap: 20px;
}

.pixel-auth-layout__panel {
  display: flex;
  min-width: 0;
  flex-direction: column;
  padding: clamp(24px, 4vw, 52px);
  border: 1px solid rgba(255,255,255,.13);
  border-radius: 14px;
  background: rgba(30,32,40,.55);
  box-shadow: inset 0 1px 0 rgba(255,255,255,.07), 0 18px 48px rgba(0,0,0,.25);
  backdrop-filter: blur(18px) saturate(125%);
}

.pixel-auth-layout__panel-head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-bottom: 26px;
  border-bottom: 1px solid rgba(255, 240, 196, .12);
}

.pixel-auth-layout__logo {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  overflow: hidden;
  border: 3px solid #f0aa25;
  background: #3a2d24;
  image-rendering: pixelated;
}

.pixel-auth-layout__logo img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.pixel-auth-layout__eyebrow {
  margin-bottom: 6px;
  font: 700 10px/1 var(--pixel-cn), monospace;
  letter-spacing: .1em;
}

.pixel-auth-layout__subtitle {
  overflow: hidden;
  color: #b9b0a5;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pixel-auth-layout__content {
  flex: 1;
  padding: 30px 0;
}

.pixel-auth-layout__footer {
  padding-top: 22px;
  border-top: 1px solid rgba(255, 240, 196, .12);
  font-size: 12px;
  text-align: center;
}

.pixel-auth-layout__copyright {
  padding-top: 18px;
  color: #766f6c;
  font-size: 8px;
  text-align: right;
}

:deep(.pixel-auth-layout__panel h2) {
  color: #fff5d6;
  font: 700 clamp(22px, 3vw, 32px)/1.2 Manrope, "SF Pro Display", "PingFang SC", "Microsoft YaHei", sans-serif;
  letter-spacing: 0;
}

:deep(.pixel-auth-layout__panel p),
:deep(.pixel-auth-layout__panel label),
:deep(.pixel-auth-layout__panel span) {
  color: #b9b0a5;
}

:deep(.pixel-auth-layout__panel .input-label) {
  color: #ded2bb;
  font: 700 11px/1.4 Manrope, "SF Pro Text", "PingFang SC", "Microsoft YaHei", sans-serif;
  letter-spacing: .03em;
  text-transform: uppercase;
}

:deep(.pixel-auth-layout__panel .input),
:deep(.pixel-auth-layout__panel .btn-secondary) {
  border: 1px solid rgba(255,255,255,.13);
  border-radius: 10px;
  background: rgba(12,14,19,.46);
  color: #fffbe6;
  box-shadow: inset 0 1px 3px rgba(0,0,0,.22);
}

:deep(.pixel-auth-layout__panel .input:focus) {
  border-color: rgba(131,207,223,.75);
  box-shadow: 0 0 0 3px rgba(131,207,223,.15), inset 0 1px 3px rgba(0,0,0,.22);
  outline: none;
  --tw-ring-color: transparent;
}

:deep(.pixel-auth-layout__panel .btn-primary) {
  min-height: 46px;
  border: 1px solid rgba(191,242,199,.46) !important;
  border-radius: 10px !important;
  background: #67bb74 !important;
  color: #13200b !important;
  box-shadow: inset 0 1px rgba(255,255,255,.42), 0 8px 22px rgba(103,187,116,.2) !important;
  font: 700 11px/1 var(--pixel-cn), monospace;
  text-transform: uppercase;
}

:deep(.pixel-auth-layout__panel .btn-primary:hover) {
  background: #83cb8e !important;
}

:deep(.pixel-auth-layout__panel a) {
  color: #a7c76a;
}

@media (max-width: 980px) {
  .pixel-auth-layout {
    padding: 16px 12px;
  }

  .pixel-auth-layout__topbar {
    margin-bottom: 12px;
  }

  .pixel-auth-layout__grid {
    grid-template-columns: 1fr;
    min-height: 0;
  }

  .pixel-auth-layout__panel {
    padding: 24px 20px;
  }

  .pixel-auth-layout__copyright {
    text-align: center;
  }
}
</style>
