<template>
  <aside
    class="pixel-auth-scene"
    :class="{ 'is-peeking': focus === 'email', 'is-avoiding': focus === 'password' }"
    :style="sceneStyle"
    aria-hidden="true"
    @mousemove="followPointer"
    @mouseleave="resetLook"
  >
    <div class="pixel-auth-scene__grid"></div>
    <div class="pixel-auth-scene__brand">POOLAI / {{ mode === 'register' ? 'ONBOARD' : 'ACCESS' }}</div>
    <div class="pixel-auth-scene__copy">
      <strong>{{ mode === 'register' ? '创建你的 Token 世界' : '欢迎回到 Token 世界' }}</strong>
      <span>安全接入 · 实时可见 · 自由调度</span>
    </div>

    <div class="pixel-auth-crew">
      <div class="pixel-auth-creature pixel-auth-creature--orange">
        <span class="pixel-auth-mark">T</span>
        <span class="pixel-auth-eyes"><i></i><i></i></span>
        <b></b>
      </div>
      <div class="pixel-auth-creature pixel-auth-creature--black">
        <span class="pixel-auth-mark">K</span>
        <span class="pixel-auth-eyes"><i></i><i></i></span>
        <b></b>
      </div>
      <div class="pixel-auth-creature pixel-auth-creature--purple">
        <span class="pixel-auth-mark">N</span>
        <span class="pixel-auth-eyes"><i></i><i></i></span>
        <b></b>
      </div>
    </div>

    <div class="pixel-auth-cat">
      <span class="pixel-auth-cat__token">TOKEN</span>
      <div class="pixel-auth-cat__neck"></div>
      <div class="pixel-auth-cat__head">
        <i class="pixel-auth-cat__ear pixel-auth-cat__ear--left"></i>
        <i class="pixel-auth-cat__ear pixel-auth-cat__ear--right"></i>
        <span class="pixel-auth-cat__eyes"><i><b></b></i><i><b></b></i></span>
        <span class="pixel-auth-cat__muzzle"><i></i><i></i></span>
        <span class="pixel-auth-cat__whiskers pixel-auth-cat__whiskers--left"><i></i><i></i><i></i></span>
        <span class="pixel-auth-cat__whiskers pixel-auth-cat__whiskers--right"><i></i><i></i><i></i></span>
      </div>
      <div class="pixel-auth-cat__body"><i></i><i></i></div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, toRef } from 'vue'

type SceneFocus = 'email' | 'password' | null

const props = withDefaults(defineProps<{
  focus?: SceneFocus
  mode?: 'login' | 'register'
}>(), {
  focus: null,
  mode: 'login'
})

const lookX = ref(0)
const lookY = ref(0)

const sceneStyle = computed(() => ({
  '--scene-look-x': `${lookX.value}px`,
  '--scene-look-y': `${lookY.value}px`
}) as Record<string, string>)

function followPointer(event: MouseEvent): void {
  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()
  lookX.value = Math.max(-5, Math.min(5, ((event.clientX - rect.left) / rect.width - 0.5) * 11))
  lookY.value = Math.max(-4, Math.min(4, ((event.clientY - rect.top) / rect.height - 0.45) * 9))
}

function resetLook(): void {
  lookX.value = 0
  lookY.value = 0
}

const focus = toRef(props, 'focus')
const mode = toRef(props, 'mode')
</script>

<style scoped>
.pixel-auth-scene {
  --pixel-cn: var(--pool-pixel-en);
  position: relative;
  min-height: 680px;
  overflow: hidden;
  isolation: isolate;
  color: #30251b;
  background-color: #c57b15;
  background-image:
    linear-gradient(90deg, rgba(75, 51, 16, .18) 2px, transparent 2px),
    linear-gradient(rgba(255, 213, 103, .16) 2px, transparent 2px),
    linear-gradient(135deg, #f7b91a 0 48%, #e88a10 48% 100%);
  background-size: 24px 24px, 24px 24px, 100% 100%;
  border: 4px solid #4b3520;
  box-shadow: 6px 6px 0 rgba(54, 36, 17, .4), inset 4px 4px 0 rgba(255, 241, 154, .24);
}

.pixel-auth-scene__grid {
  position: absolute;
  inset: 0;
  z-index: -1;
  opacity: .24;
  background-image: linear-gradient(rgba(95, 62, 6, .25) 2px, transparent 2px), linear-gradient(90deg, rgba(95, 62, 6, .25) 2px, transparent 2px);
  background-size: 42px 42px;
}

.pixel-auth-scene__brand,
.pixel-auth-scene__copy {
  position: absolute;
  z-index: 4;
  left: 46px;
  font-family: var(--pixel-cn), monospace;
  text-shadow: 2px 2px 0 rgba(255, 235, 146, .65);
}

.pixel-auth-scene__brand {
  top: 42px;
  color: #563610;
  font-size: 11px;
  font-weight: 900;
  letter-spacing: .18em;
}

.pixel-auth-scene__copy {
  top: 80px;
  display: grid;
  gap: 10px;
}

.pixel-auth-scene__copy strong {
  max-width: 12ch;
  font-size: clamp(30px, 3.3vw, 52px);
  line-height: 1.12;
}

/* Keep the legacy scene readable even when an older checkout contains broken localized text. */
.pixel-auth-scene__copy strong {
  font-size: 0;
}

.pixel-auth-scene__copy strong::after {
  content: 'TOKEN WORLD ONLINE';
  display: block;
  font-size: clamp(30px, 3.3vw, 52px);
  line-height: 1.12;
}

.pixel-auth-scene__copy span {
  color: #6a4318;
  font-size: 12px;
  font-weight: 800;
}

.pixel-auth-scene__copy span {
  font-size: 0;
}

.pixel-auth-scene__copy span::after {
  content: 'SECURE ACCESS / LIVE CONTROL / FREE ROUTING';
  display: block;
  font-size: 12px;
}

.pixel-auth-cat {
  position: absolute;
  z-index: 3;
  bottom: 92px;
  left: 50%;
  width: 240px;
  height: 320px;
  transform: translateX(-50%);
  filter: none;
  transition: transform .28s steps(3);
}

.pixel-auth-cat__body {
  position: absolute;
  bottom: 0;
  left: 42px;
  width: 156px;
  height: 130px;
  background: #e6a51d;
  box-shadow: 8px 8px 0 #9c5f12, inset 8px 8px 0 #f8ca3c;
  clip-path: polygon(18px 0, calc(100% - 18px) 0, calc(100% - 18px) 12px, 100% 12px, 100% 100%, 0 100%, 0 12px, 18px 12px);
}

.pixel-auth-cat__body i {
  position: absolute;
  bottom: -8px;
  width: 48px;
  height: 34px;
  background: #ffe58c;
  box-shadow: 3px 3px 0 #b77c22;
}

.pixel-auth-cat__body i:first-child { left: 11px; }
.pixel-auth-cat__body i:last-child { right: 11px; }

.pixel-auth-cat__neck {
  position: absolute;
  z-index: -1;
  bottom: 112px;
  left: 87px;
  width: 68px;
  height: 54px;
  background: #e6a51d;
  transform-origin: bottom;
  transition: height .34s steps(3), transform .34s steps(3);
}

.pixel-auth-cat__head {
  position: absolute;
  top: 15px;
  left: 18px;
  width: 204px;
  height: 166px;
  background: #ffd43b;
  box-shadow: 8px 8px 0 #bd7111, inset 8px 8px 0 #ffe87e;
  clip-path: polygon(12px 0, calc(100% - 12px) 0, calc(100% - 12px) 12px, 100% 12px, 100% calc(100% - 12px), calc(100% - 12px) calc(100% - 12px), calc(100% - 12px) 100%, 12px 100%, 12px calc(100% - 12px), 0 calc(100% - 12px), 0 12px, 12px 12px);
  transition: transform .32s steps(3);
}

.pixel-auth-cat__ear {
  position: absolute;
  top: -30px;
  width: 65px;
  height: 78px;
  background: #e8a51c;
  box-shadow: 5px 5px 0 #a26013;
  transform: skewY(-18deg);
}

.pixel-auth-cat__ear::after {
  position: absolute;
  inset: 16px 17px 18px;
  background: #f3983e;
  content: '';
}

.pixel-auth-cat__ear--left { left: 10px; }
.pixel-auth-cat__ear--right { right: 10px; transform: scaleX(-1) skewY(-18deg); }

.pixel-auth-cat__eyes {
  position: absolute;
  top: 58px;
  left: 42px;
  display: flex;
  gap: 26px;
}

.pixel-auth-cat__eyes > i {
  display: grid;
  width: 42px;
  height: 50px;
  place-items: center;
  background: #fffdf1;
  box-shadow: 4px 4px 0 #b9a55b;
}

.pixel-auth-cat__eyes b,
.pixel-auth-eyes i::after {
  display: block;
  width: 16px;
  height: 20px;
  background: #29251f;
  content: '';
  transform: translate(var(--scene-look-x), var(--scene-look-y));
  transition: transform .1s ease-out;
}

.pixel-auth-cat__muzzle {
  position: absolute;
  bottom: 24px;
  left: 77px;
  display: flex;
  gap: 12px;
}

.pixel-auth-cat__muzzle i {
  width: 21px;
  height: 6px;
  background: #725016;
}

.pixel-auth-cat__whiskers {
  position: absolute;
  top: 111px;
  display: grid;
  width: 56px;
  gap: 8px;
  transition: transform .3s steps(3);
}

.pixel-auth-cat__whiskers i { height: 3px; background: #725016; }
.pixel-auth-cat__whiskers--left { left: -38px; transform: rotate(-9deg); }
.pixel-auth-cat__whiskers--right { right: -38px; transform: rotate(9deg); }

.pixel-auth-cat__token {
  position: absolute;
  z-index: 5;
  bottom: 39px;
  left: 50%;
  padding: 5px 7px;
  transform: translateX(-50%);
  border: 3px solid #342f27;
  background: #5c8e35;
  box-shadow: 3px 3px 0 #342f27;
  color: #fff9db;
  font: 800 8px/1 var(--pixel-cn), monospace;
  letter-spacing: .07em;
  animation: pixel-auth-token-bob 2.4s steps(3) infinite;
}

.pixel-auth-crew {
  position: absolute;
  right: 3%;
  left: 18%;
  bottom: 76px;
  z-index: 4;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
}

.pixel-auth-creature {
  position: relative;
  display: grid;
  place-items: center;
  border: 5px solid #241a22;
  box-shadow: 6px 6px 0 rgba(57, 40, 12, .45), inset 5px 5px 0 rgba(255, 255, 255, .13);
  color: #fff7e7;
  transition: transform .28s steps(3);
  clip-path: polygon(10px 0, calc(100% - 10px) 0, calc(100% - 10px) 10px, 100% 10px, 100% calc(100% - 10px), calc(100% - 10px) calc(100% - 10px), calc(100% - 10px) 100%, 10px 100%, 10px calc(100% - 10px), 0 calc(100% - 10px), 0 10px, 10px 10px);
  animation: pixel-auth-float 3.2s steps(4) infinite;
}

.pixel-auth-creature--orange { width: 79px; height: 79px; background: #e96813; }
.pixel-auth-creature--black { width: 57px; height: 92px; background: #38343a; animation-delay: -.8s; }
.pixel-auth-creature--purple { width: 70px; height: 141px; background: #7137bd; animation-delay: -1.6s; }

.pixel-auth-mark { position: absolute; top: 8px; color: #fff9df; font: 800 11px var(--pixel-cn), monospace; }
.pixel-auth-eyes { display: flex; gap: 10px; align-items: center; margin-top: -10px; }
.pixel-auth-eyes i { position: relative; width: 13px; height: 17px; background: #fff; }
.pixel-auth-eyes i::after { width: 7px; height: 7px; margin: 5px auto; }
.pixel-auth-creature b { position: absolute; bottom: 17px; width: 19px; height: 7px; background: #fff; }
.pixel-auth-creature--purple b { bottom: 26px; }

.is-peeking .pixel-auth-cat { transform: translateX(-50%) translateY(-20px); }
.is-peeking .pixel-auth-cat__neck { height: 104px; transform: scaleX(.9); }
.is-peeking .pixel-auth-cat__head { transform: translateY(-38px) rotate(-2deg); }
.is-peeking .pixel-auth-cat__whiskers--left { transform: translateY(-30px) rotate(-15deg); }
.is-peeking .pixel-auth-cat__whiskers--right { transform: translateY(-30px) rotate(15deg); }
.is-peeking .pixel-auth-creature b { width: 13px; height: 13px; }
.is-peeking .pixel-auth-cat__eyes b,
.is-peeking .pixel-auth-eyes i::after { transform: translate(7px, 3px); }
.is-peeking .pixel-auth-creature b { border-radius: 50%; }
.is-avoiding .pixel-auth-cat__eyes b,
.is-avoiding .pixel-auth-eyes i::after { transform: translate(-5px, 1px) !important; }
.is-avoiding .pixel-auth-cat__eyes b { transform: translate(-7px, 1px) !important; }

@keyframes pixel-auth-float { 0%, 100% { translate: 0 0; } 35% { translate: 0 -4px; } 72% { translate: 0 2px; } }
@keyframes pixel-auth-token-bob { 0%, 100% { translate: -50% 0; } 50% { translate: -50% -5px; } }

@media (max-width: 980px) {
  .pixel-auth-scene { min-height: 260px; }
  .pixel-auth-scene__copy { top: 30px; left: 26px; }
  .pixel-auth-scene__copy strong { font-size: 28px; }
  .pixel-auth-cat { bottom: -62px; left: 67%; transform: translateX(-50%) scale(.68); transform-origin: bottom center; }
  .pixel-auth-crew { right: 10px; left: 20%; bottom: 18px; transform: scale(.68); transform-origin: bottom right; }
}
</style>
