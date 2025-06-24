<template>
  <div class="refresh-interval-selector">
    <div class="flex align-items-center gap-3">
      <div class="flex align-items-center gap-2">
        <label class="text-900 font-medium text-sm whitespace-nowrap">Refresh:</label>
        <Dropdown 
          v-model="selectedInterval" 
          :options="intervalOptions"
          optionLabel="label"
          optionValue="value"
          @change="onIntervalChange"
          class="refresh-dropdown"
          :class="{ 'p-dropdown-sm': size === 'small' }"
        />
      </div>
      
      <div class="flex align-items-center gap-2">
        <i class="pi pi-clock text-500"></i>
        <span class="text-500 text-sm">{{ countdownText }}</span>
      </div>
      
      <Button 
        v-if="showManualRefresh"
        @click="onManualRefresh"
        icon="pi pi-refresh"
        :class="{ 'p-button-sm': size === 'small' }"
        class="p-button-text p-button-plain"
        :loading="refreshing"
        v-tooltip.top="'Refresh Now'"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import Dropdown from 'primevue/dropdown'
import Button from 'primevue/button'

const props = defineProps({
  defaultInterval: {
    type: Number,
    default: 30000 // 30 seconds
  },
  size: {
    type: String,
    default: 'normal', // 'small' or 'normal'
    validator: (value) => ['small', 'normal'].includes(value)
  },
  showManualRefresh: {
    type: Boolean,
    default: true
  },
  storageKey: {
    type: String,
    default: 'etl-refresh-interval'
  }
})

const emit = defineEmits(['interval-changed', 'manual-refresh'])

// Interval options
const intervalOptions = [
  { label: '5 seconds', value: 5000 },
  { label: '15 seconds', value: 15000 }, 
  { label: '30 seconds', value: 30000 },
  { label: '1 minute', value: 60000 },
  { label: '5 minutes', value: 300000 },
  { label: '15 minutes', value: 900000 }
]

// State
const selectedInterval = ref(props.defaultInterval)
const countdown = ref(0)
const refreshing = ref(false)
let countdownTimer = null

// Computed
const countdownText = computed(() => {
  if (countdown.value <= 0) return 'Refreshing...'
  
  const seconds = Math.ceil(countdown.value / 1000)
  if (seconds < 60) {
    return `Next refresh in ${seconds}s`
  }
  
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  if (remainingSeconds === 0) {
    return `Next refresh in ${minutes}m`
  }
  return `Next refresh in ${minutes}m ${remainingSeconds}s`
})

// Methods
const startCountdown = () => {
  countdown.value = selectedInterval.value
  
  if (countdownTimer) {
    clearInterval(countdownTimer)
  }
  
  countdownTimer = setInterval(() => {
    countdown.value -= 1000
    
    if (countdown.value <= 0) {
      // Reset countdown for next cycle
      countdown.value = selectedInterval.value
      // Don't emit here - let the parent handle the actual refresh timing
    }
  }, 1000)
}

const stopCountdown = () => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

const resetCountdown = () => {
  countdown.value = selectedInterval.value
}

const onIntervalChange = () => {
  // Save to localStorage
  localStorage.setItem(props.storageKey, selectedInterval.value.toString())
  
  // Emit the change
  emit('interval-changed', selectedInterval.value)
  
  // Restart countdown with new interval
  startCountdown()
}

const onManualRefresh = () => {
  refreshing.value = true
  emit('manual-refresh')
  
  // Reset countdown
  resetCountdown()
  
  // Stop refreshing indicator after a short delay
  setTimeout(() => {
    refreshing.value = false
  }, 500)
}

// Load saved interval from localStorage
const loadSavedInterval = () => {
  const saved = localStorage.getItem(props.storageKey)
  if (saved) {
    const savedValue = parseInt(saved)
    if (intervalOptions.some(opt => opt.value === savedValue)) {
      selectedInterval.value = savedValue
    }
  }
}

// Expose methods for parent components
defineExpose({
  resetCountdown,
  startCountdown,
  stopCountdown
})

// Lifecycle
onMounted(() => {
  loadSavedInterval()
  startCountdown()
  
  // Emit initial interval
  emit('interval-changed', selectedInterval.value)
})

onUnmounted(() => {
  stopCountdown()
})

// Watch for external countdown resets
watch(() => props.defaultInterval, (newVal) => {
  if (!localStorage.getItem(props.storageKey)) {
    selectedInterval.value = newVal
    startCountdown()
  }
})
</script>

<style scoped>
.refresh-interval-selector {
  display: flex;
  align-items: center;
}

.refresh-dropdown {
  min-width: 120px;
}

.refresh-dropdown.p-dropdown-sm {
  min-width: 100px;
}

.whitespace-nowrap {
  white-space: nowrap;
}
</style>