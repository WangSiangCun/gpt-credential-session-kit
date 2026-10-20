<template>
  <svg
    class="pixel-nav-icon"
    viewBox="0 0 16 16"
    fill="none"
    stroke="currentColor"
    stroke-width="1.5"
    stroke-linecap="square"
    stroke-linejoin="miter"
    shape-rendering="crispEdges"
    aria-hidden="true"
  >
    <path v-for="(path, index) in iconPaths" :key="index" :d="path" />
  </svg>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ path: string }>()

const iconPaths = computed(() => {
  const path = props.path.toLowerCase()
  if (/dashboard/.test(path)) return ['M2 2h5v5H2z', 'M9 2h5v5H9z', 'M2 9h5v5H2z', 'M9 9h5v5H9z']
  if (/\/admin\/ops/.test(path)) return ['M2 8h2l2-4 3 8 2-5h3', 'M2 14h12']
  if (/usage/.test(path)) return ['M2 14V9h3v5', 'M7 14V5h3v9', 'M12 14V2h2v12', 'M1 14h14']
  if (/monitor/.test(path)) return ['M2 12V8', 'M5 12V5', 'M8 12V2', 'M11 12V7', 'M14 12V4', 'M1 14h14']
  if (/keys/.test(path)) return ['M3 10a3 3 0 113-3 3 3 0 01-3 3z', 'M6 7h7v2h-2v2H9V9H6']
  if (/batch-image/.test(path)) return ['M2 3h9v9H2z', 'M5 6h1v1H5z', 'M3 10l3-3 4 4', 'M11 6h3v7H7']
  if (/redeem/.test(path)) return ['M2 5h12v6H2z', 'M5 5v6', 'M11 5v6', 'M7 8h2']
  if (/promo/.test(path)) return ['M2 3h12v10H2z', 'M5 3v10', 'M10 3v10', 'M7 6h2', 'M7 10h2']
  if (/activities|lottery/.test(path)) return ['M3 2v12', 'M4 3h8l-2 3 2 3H4z', 'M2 14h4']
  if (/users/.test(path)) return ['M5 7a3 3 0 110-6 3 3 0 010 6z', 'M1 14v-3l2-2h4l2 2v3z', 'M11 5h4', 'M13 3v4']
  if (/groups/.test(path)) return ['M3 3h3v3H3z', 'M10 3h3v3h-3z', 'M6 10h4v3H6z', 'M6 4h4', 'M8 4v6']
  if (/channels\/pricing/.test(path)) return ['M2 3h8l4 4-7 7-5-5z', 'M5 6h2v2H5z']
  if (/available-channels/.test(path)) return ['M2 3h4v4H2z', 'M10 3h4v4h-4z', 'M6 5h4', 'M6 11h4', 'M8 5v6', 'M6 10h4v4H6z']
  if (/channels/.test(path)) return ['M2 4h3v3H2z', 'M11 2h3v3h-3z', 'M11 10h3v3h-3z', 'M5 5h3v6', 'M8 4l3-1', 'M8 10l3 1']
  if (/accounts/.test(path)) return ['M2 2h12v12H2z', 'M4 5h8', 'M4 8h8', 'M4 11h3', 'M10 11h2']
  if (/subscription/.test(path)) return ['M3 2h10v12H3z', 'M5 1v3', 'M11 1v3', 'M3 5h10', 'M5 8h2', 'M9 8h2', 'M5 11h2']
  if (/purchase|payment\/?$/.test(path)) return ['M2 5h12v8H2z', 'M2 7h12', 'M5 10h4']
  if (/orders/.test(path)) return ['M3 2h10v12H3z', 'M5 5h6', 'M5 8h6', 'M5 11h3']
  if (/announcement/.test(path)) return ['M2 7h3l6-3v8L5 9H2z', 'M5 9v4h2', 'M12 6v4']
  if (/proxies|\/ip|ip-/.test(path)) return ['M8 1l5 3v4c0 3-2 5-5 7-3-2-5-4-5-7V4z', 'M5 8h6', 'M8 5v6']
  if (/prompt-audit/.test(path)) return ['M3 2h8l2 2v10H3z', 'M5 7h6', 'M5 10h4', 'M10 2v3h3']
  if (/risk-control|security-audit/.test(path)) return ['M8 1l5 3v4c0 3-2 5-5 7-3-2-5-4-5-7V4z', 'M6 8l2 2 3-4']
  if (/logs/.test(path)) return ['M3 2h7l3 3v9H3z', 'M10 2v3h3', 'M5 8h6', 'M5 11h6']
  if (/settings/.test(path)) return ['M6 2h4l1 2 2 1v4l-2 1-1 2H6L5 10 3 9V5l2-1z', 'M8 6a2 2 0 110 4 2 2 0 010-4z']
  if (/affiliate/.test(path)) return ['M3 5h4v3H3z', 'M9 8h4v3H9z', 'M7 6h2', 'M8 6v4', 'M5 12h6']
  if (/profile/.test(path)) return ['M8 7a3 3 0 110-6 3 3 0 010 6z', 'M3 14v-3l3-2h4l3 2v3z']
  return ['M3 3h10v10H3z', 'M5 5h2v2H5z', 'M9 5h2v2H9z', 'M7 9h2v2H7z']
})
</script>

<style scoped>
.pixel-nav-icon {
  display: block;
  width: 16px;
  height: 16px;
  flex: 0 0 16px;
  color: #9aa5b5;
  image-rendering: pixelated;
  transition: color .12s steps(2), transform .12s steps(2);
}
:global(.sidebar-link:hover) .pixel-nav-icon { color: #d9e4ef; transform: translateY(-1px); }
:global(.sidebar-link-active) .pixel-nav-icon { color: #6ec177; }
</style>
