<template>
  <main class="min-h-screen bg-slate-950 text-white flex items-center justify-center px-6">
    <section class="w-full max-w-md rounded-2xl border border-white/10 bg-white/10 p-6 shadow-2xl">
      <p class="text-sm text-white/60">Beacon Toolkit</p>
      <h1 class="mt-2 text-2xl font-semibold">正在返回应用</h1>
      <p class="mt-3 text-sm leading-6 text-white/70">
        如果没有自动打开应用，请使用下方按钮继续。
      </p>
      <a
        class="mt-6 inline-flex w-full items-center justify-center rounded-xl bg-blue-500 px-4 py-3 font-medium text-white hover:bg-blue-400"
        :href="nativeUrl"
      >
        打开 Beacon Toolkit
      </a>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'

const nativeUrl = computed(() => {
  const params = new URLSearchParams(window.location.search)
  const target = new URL('com.beacontoolkit://oauth/callback')
  for (const [key, value] of params.entries()) {
    target.searchParams.append(key, value)
  }
  return target.toString()
})

onMounted(() => {
  window.location.href = nativeUrl.value
})
</script>
