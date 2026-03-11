<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'

const lines = [
  '$ hamburger',
  'Hamburger - Next Generation Gateway for JJApps.',
  'Version: 0.1.0',
  'Usage:',
  '  hamburger [flags]',
  '  hamburger [command]',
  'Available Commands:',
  '  completion  Generate the autocompletion script for the specified shell',
  '  generate    generate config file',
  '  help        Help about any command',
  '  reload      reload service',
  '  run         run gateway',
  '  test        test config',
]

const typedLines = ref<string[]>([''])
const rowIndex = ref(0)
const charIndex = ref(0)
const isTyping = ref(true)
let timer: number | undefined

const renderTick = () => {
  if (!isTyping.value) {
    return
  }

  const targetLine = lines[rowIndex.value]
  if (!targetLine) {
    isTyping.value = false
    return
  }

  if (charIndex.value < targetLine.length) {
    charIndex.value += 1
    typedLines.value[rowIndex.value] = targetLine.slice(0, charIndex.value)
    return
  }

  if (rowIndex.value < lines.length - 1) {
    rowIndex.value += 1
    charIndex.value = 0
    typedLines.value.push('')
    return
  }

  isTyping.value = false
}

onMounted(() => {
  timer = window.setInterval(renderTick, 28)
})

onUnmounted(() => {
  if (timer !== undefined) {
    window.clearInterval(timer)
  }
})
</script>

<template>
  <div class="hero-shell">
    <div v-for="(line, index) in typedLines" :key="index">
      <span>{{ line }}</span>
      <span v-if="isTyping && index === rowIndex" class="cursor">▋</span>
      <span v-if="!isTyping && index === typedLines.length - 1" class="cursor done">▋</span>
    </div>
  </div>
</template>
