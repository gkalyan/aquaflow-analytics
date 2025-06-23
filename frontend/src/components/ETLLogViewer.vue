<template>
  <div class="etl-log-viewer bg-white rounded-lg shadow">
    <div class="px-4 py-3 border-b border-gray-200 flex justify-between items-center">
      <h3 class="text-lg font-semibold">Job Logs</h3>
      <div class="flex items-center gap-2">
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
    
    <div 
      ref="logContainer"
      class="h-96 overflow-y-auto bg-gray-900 text-gray-100 p-4 font-mono text-sm"
    >
      <div v-if="loading" class="text-center py-4">
        <span class="text-gray-400">Loading logs...</span>
      </div>
      
      <div v-else-if="logs.length === 0" class="text-center py-4">
        <span class="text-gray-400">No logs available</span>
      </div>
      
      <div v-else>
        <div 
          v-for="log in logs" 
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
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
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