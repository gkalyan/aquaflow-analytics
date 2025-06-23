<template>
  <div class="layout-container">
    <div class="layout-grid">
      <!-- Header with filters -->
      <div class="layout-card">
        <div class="flex justify-content-between align-items-center mb-4">
          <h2 class="text-900 font-bold text-2xl m-0">ETL Job Logs</h2>
          <div class="flex align-items-center gap-2">
            <i class="pi pi-clock text-500"></i>
            <span class="text-500">Next refresh in {{ countdown }}s</span>
          </div>
        </div>

        <!-- Filters -->
        <div class="grid">
          <div class="col-12 md:col-3">
            <label class="block text-900 font-medium mb-2">Job Name</label>
            <InputText 
              v-model="filters.jobName" 
              placeholder="Filter by job name..." 
              class="w-full"
              @input="applyFilters"
            />
          </div>
          <div class="col-12 md:col-2">
            <label class="block text-900 font-medium mb-2">Log Level</label>
            <Dropdown 
              v-model="filters.logLevel" 
              :options="logLevels"
              placeholder="All Levels"
              class="w-full"
              @change="applyFilters"
            />
          </div>
          <div class="col-12 md:col-2">
            <label class="block text-900 font-medium mb-2">Series ID</label>
            <InputText 
              v-model="filters.seriesId" 
              placeholder="Series ID..." 
              class="w-full"
              @input="applyFilters"
            />
          </div>
          <div class="col-12 md:col-2">
            <label class="block text-900 font-medium mb-2">Limit</label>
            <Dropdown 
              v-model="filters.limit" 
              :options="limitOptions"
              class="w-full"
              @change="applyFilters"
            />
          </div>
          <div class="col-12 md:col-3">
            <label class="block text-900 font-medium mb-2">Actions</label>
            <div class="flex gap-2">
              <Button @click="clearFilters" label="Clear" severity="secondary" size="small" />
              <Button @click="refreshLogs" label="Refresh" severity="primary" size="small" />
              <Button 
                @click="toggleAutoScroll" 
                :label="autoScroll ? 'Stop Auto-scroll' : 'Auto-scroll'"
                :severity="autoScroll ? 'warn' : 'success'"
                size="small"
              />
            </div>
          </div>
        </div>
      </div>

      <!-- Logs table -->
      <div class="layout-card">
        <DataTable 
          :value="logs" 
          :loading="loading"
          scrollable
          scrollHeight="600px"
          class="p-datatable-sm"
          :rowClass="getRowClass"
        >
          <template #empty>
            <div class="text-center py-4">
              <i class="pi pi-info-circle text-500 text-3xl mb-3"></i>
              <p class="text-500 m-0">No logs found</p>
            </div>
          </template>

          <Column field="timestamp" header="Timestamp" style="min-width: 200px">
            <template #body="{ data }">
              <span class="text-sm">{{ formatTimestamp(data.timestamp) }}</span>
            </template>
          </Column>

          <Column field="log_level" header="Level" style="width: 80px">
            <template #body="{ data }">
              <Tag 
                :value="data.log_level" 
                :severity="getLogLevelSeverity(data.log_level)"
                class="text-xs"
              />
            </template>
          </Column>

          <Column field="job_name" header="Job" style="min-width: 200px">
            <template #body="{ data }">
              <span class="text-sm font-medium">{{ data.job_name }}</span>
              <br>
              <span class="text-xs text-500">{{ data.job_type }}</span>
            </template>
          </Column>

          <Column field="message" header="Message" style="min-width: 300px">
            <template #body="{ data }">
              <span class="text-sm">{{ data.message }}</span>
            </template>
          </Column>

          <Column field="context" header="Context" style="min-width: 200px">
            <template #body="{ data }">
              <div v-if="data.context" class="text-xs">
                <div v-if="data.context.series_id" class="mb-1">
                  <strong>Series:</strong> {{ data.context.series_id }}
                </div>
                <div v-if="data.context.page" class="mb-1">
                  <strong>Page:</strong> {{ data.context.page }}
                </div>
                <div v-if="data.context.batch_size" class="mb-1">
                  <strong>Batch:</strong> {{ data.context.batch_size }}
                </div>
                <div v-if="data.context.records_inserted" class="mb-1">
                  <strong>Inserted:</strong> {{ data.context.records_inserted }}
                </div>
                <div v-if="data.context.error" class="text-red-500">
                  <strong>Error:</strong> {{ data.context.error }}
                </div>
              </div>
              <span v-else class="text-500">-</span>
            </template>
          </Column>

          <Column field="batch_id" header="Batch ID" style="width: 120px">
            <template #body="{ data }">
              <span class="text-xs font-mono">{{ data.batch_id.substring(0, 8) }}...</span>
            </template>
          </Column>
        </DataTable>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { etlApi } from '../services/etlApi'

// Data
const logs = ref([])
const loading = ref(false)
const countdown = ref(5)
const autoScroll = ref(true)

let intervalId = null

// Filters
const filters = reactive({
  jobName: '',
  logLevel: null,
  seriesId: '',
  limit: 200
})

const logLevels = [
  { label: 'All Levels', value: null },
  { label: 'DEBUG', value: 'DEBUG' },
  { label: 'INFO', value: 'INFO' },
  { label: 'WARN', value: 'WARN' },
  { label: 'ERROR', value: 'ERROR' }
]

const limitOptions = [
  { label: '50', value: 50 },
  { label: '100', value: 100 },
  { label: '200', value: 200 },
  { label: '500', value: 500 }
]

// Methods
const fetchLogs = async () => {
  try {
    loading.value = true
    
    const params = {
      limit: filters.limit
    }
    
    if (filters.jobName.trim()) {
      params.job_name = filters.jobName.trim()
    }
    
    if (filters.logLevel) {
      params.level = filters.logLevel
    }
    
    if (filters.seriesId.trim()) {
      params.series_id = filters.seriesId.trim()
    }

    const response = await etlApi.getAllLogs(params)
    logs.value = response.data.logs || []
    
    // Auto-scroll to bottom if enabled
    if (autoScroll.value) {
      await nextTick()
      scrollToBottom()
    }
  } catch (error) {
    console.error('Failed to fetch logs:', error)
  } finally {
    loading.value = false
  }
}

const refreshLogs = () => {
  fetchLogs()
  countdown.value = 5
}

const applyFilters = () => {
  fetchLogs()
}

const clearFilters = () => {
  filters.jobName = ''
  filters.logLevel = null
  filters.seriesId = ''
  filters.limit = 200
  fetchLogs()
}

const toggleAutoScroll = () => {
  autoScroll.value = !autoScroll.value
}

const scrollToBottom = () => {
  const scrollContainer = document.querySelector('.p-datatable-scrollable-body')
  if (scrollContainer) {
    scrollContainer.scrollTop = scrollContainer.scrollHeight
  }
}

const formatTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString()
}

const getLogLevelSeverity = (level) => {
  switch (level) {
    case 'ERROR': return 'danger'
    case 'WARN': return 'warn'
    case 'INFO': return 'info'
    case 'DEBUG': return 'secondary'
    default: return 'secondary'
  }
}

const getRowClass = (data) => {
  switch (data.log_level) {
    case 'ERROR': return 'bg-red-50'
    case 'WARN': return 'bg-yellow-50'
    default: return ''
  }
}

const startPolling = () => {
  intervalId = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      fetchLogs()
      countdown.value = 5
    }
  }, 1000)
}

const stopPolling = () => {
  if (intervalId) {
    clearInterval(intervalId)
    intervalId = null
  }
}

// Lifecycle
onMounted(() => {
  fetchLogs()
  startPolling()
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.layout-container {
  padding: 1.5rem;
  max-width: 100%;
}

.layout-grid {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.layout-card {
  background: var(--surface-0);
  border: 1px solid var(--surface-border);
  border-radius: 8px;
  box-shadow: 0 2px 8px 0 rgba(0, 0, 0, 0.08);
  padding: 1.5rem;
}

.grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  gap: 1rem;
}

.col-12 { grid-column: span 12; }
.col-3 { grid-column: span 3; }
.col-2 { grid-column: span 2; }

@media (max-width: 768px) {
  .col-3, .col-2 { grid-column: span 12; }
}

.bg-red-50 {
  background-color: #fef2f2;
}

.bg-yellow-50 {
  background-color: #fefce8;
}
</style>