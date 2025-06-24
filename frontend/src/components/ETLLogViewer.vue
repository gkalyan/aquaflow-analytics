<template>
  <div class="etl-log-viewer bg-white rounded-lg shadow">
    <div class="px-4 py-3 border-b border-gray-200">
      <div class="flex justify-between items-center mb-2">
        <h3 class="text-lg font-semibold">Job Logs</h3>
        <div class="flex items-center gap-3">
          <div class="flex items-center gap-2">
            <label class="text-sm font-medium">Time Window:</label>
            <select 
              v-model="timeWindow" 
              @change="onTimeWindowChange"
              class="text-sm border border-gray-300 rounded px-2 py-1"
            >
              <option value="30">Last 30s</option>
              <option value="60">Last 1m</option>
              <option value="300">Last 5m</option>
              <option value="900">Last 15m</option>
              <option value="all">All logs</option>
            </select>
          </div>
          <label class="flex items-center">
            <input 
              type="checkbox" 
              v-model="autoScroll"
              class="mr-2"
            >
            <span class="text-sm">Auto-scroll</span>
          </label>
          <button 
            @click="clearLogs"
            class="text-sm px-3 py-1 bg-gray-500 text-white rounded hover:bg-gray-600"
          >
            Clear
          </button>
        </div>
      </div>
      <div class="flex justify-between items-center text-sm text-gray-500">
        <span>{{ getTimeWindowDescription() }}</span>
        <span>{{ filteredLogs.length }} logs in window</span>
      </div>
    </div>
    
    <div 
      ref="logContainer"
      class="h-96 overflow-y-auto bg-gray-900 text-gray-100 p-4 font-mono text-sm"
    >
      <div v-if="loading" class="text-center py-4">
        <span class="text-gray-400">Loading logs...</span>
      </div>
      
      <div v-else-if="filteredLogs.length === 0" class="text-center py-4">
        <span class="text-gray-400">No logs available in time window</span>
      </div>
      
      <div v-else>
        <div 
          v-for="log in filteredLogs" 
          :key="log.log_id"
          class="mb-1 flex"
        >
          <span class="text-gray-500 mr-2">{{ formatTimestamp(log.timestamp) }}</span>
          <span :class="getLogLevelClass(log.log_level)" class="mr-2">[{{ log.log_level }}]</span>
          <span class="flex-1">{{ log.message }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { etlApi } from '@/services/etlApi'

const props = defineProps({
  jobId: {
    type: String,
    required: true
  }
})

const logs = ref([])
const loading = ref(false)
const autoScroll = ref(true)
const logContainer = ref(null)
const pollInterval = ref(null)
const lastLogTimestamp = ref(null)
const timeWindow = ref(300) // Default 5 minutes

// Computed property for filtered logs based on time window
const filteredLogs = computed(() => {
  if (timeWindow.value === 'all') {
    return logs.value
  }
  
  const cutoffTime = new Date(Date.now() - (timeWindow.value * 1000))
  return logs.value.filter(log => new Date(log.timestamp) >= cutoffTime)
})

const getLogLevelClass = (level) => {
  const classes = {
    'DEBUG': 'text-gray-400',
    'INFO': 'text-blue-400',
    'WARN': 'text-yellow-400',
    'ERROR': 'text-red-400'
  }
  return classes[level] || 'text-gray-400'
}

const formatTimestamp = (timestamp) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', { 
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const fetchLogs = async (since = null) => {
  try {
    loading.value = !logs.value.length
    const params = {}
    if (since) {
      params.since = since
    }
    
    const response = await etlApi.getJobLogs(props.jobId, params)
    const newLogs = response.data.logs || []
    
    if (newLogs.length > 0) {
      // Prepend new logs (they come in reverse order)
      logs.value = [...newLogs.reverse(), ...logs.value]
      lastLogTimestamp.value = newLogs[0].timestamp
      
      if (autoScroll.value) {
        await nextTick()
        scrollToBottom()
      }
    }
  } catch (error) {
    console.error('Failed to fetch logs:', error)
  } finally {
    loading.value = false
  }
}

const scrollToBottom = () => {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

const clearLogs = () => {
  logs.value = []
  lastLogTimestamp.value = null
}

const getTimeWindowDescription = () => {
  const descriptions = {
    '30': 'Showing logs from last 30 seconds',
    '60': 'Showing logs from last 1 minute', 
    '300': 'Showing logs from last 5 minutes',
    '900': 'Showing logs from last 15 minutes',
    'all': 'Showing all logs'
  }
  return descriptions[timeWindow.value] || 'Showing filtered logs'
}

const onTimeWindowChange = () => {
  // Save preference to localStorage
  localStorage.setItem('etl-log-viewer-time-window', timeWindow.value.toString())
  
  // Prune logs if window got smaller
  if (timeWindow.value !== 'all') {
    pruneLogs()
  }
}

const pruneLogs = () => {
  if (timeWindow.value === 'all') return
  
  const cutoffTime = new Date(Date.now() - (timeWindow.value * 1000))
  const oldLogCount = logs.value.length
  
  // Keep only logs within the time window, plus a small buffer for smooth transitions
  logs.value = logs.value.filter(log => new Date(log.timestamp) >= cutoffTime)
  
  // If we pruned logs, update the last timestamp
  if (logs.value.length < oldLogCount && logs.value.length > 0) {
    lastLogTimestamp.value = logs.value[0].timestamp
  }
}

const loadSavedPreferences = () => {
  const saved = localStorage.getItem('etl-log-viewer-time-window')
  if (saved && (saved === 'all' || !isNaN(Number(saved)))) {
    timeWindow.value = saved === 'all' ? 'all' : Number(saved)
  }
}

const startPolling = () => {
  // Initial fetch
  fetchLogs()
  
  // Poll every second for new logs
  pollInterval.value = setInterval(() => {
    if (lastLogTimestamp.value) {
      fetchLogs(lastLogTimestamp.value)
    } else {
      fetchLogs()
    }
    
    // Prune old logs periodically to prevent memory bloat
    if (timeWindow.value !== 'all') {
      pruneLogs()
    }
  }, 1000)
}

const stopPolling = () => {
  if (pollInterval.value) {
    clearInterval(pollInterval.value)
    pollInterval.value = null
  }
}

watch(() => props.jobId, (newJobId) => {
  if (newJobId) {
    clearLogs()
    stopPolling()
    startPolling()
  }
})

onMounted(() => {
  loadSavedPreferences()
  if (props.jobId) {
    startPolling()
  }
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.etl-log-viewer {
  font-family: 'Courier New', Courier, monospace;
}
</style>