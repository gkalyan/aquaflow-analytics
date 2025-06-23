<template>
  <div class="layout-container">
    <div class="layout-grid">
      <div class="layout-card">
        <div class="flex justify-content-between align-items-center">
          <h2 class="text-900 font-bold text-2xl m-0">ETL Jobs Monitor</h2>
          <div class="flex align-items-center gap-2">
            <i class="pi pi-clock text-500"></i>
            <span class="text-500">Next refresh in {{ countdown }}s</span>
          </div>
        </div>
      </div>
    
    <!-- Job Status Summary - Using Working Stats Structure -->
    <div class="stats">
      <div class="layout-card">
        <div class="stats-header">
          <span class="stats-title">Running</span>
          <span class="stats-icon-box">
            <i class="pi pi-spin pi-cog"></i>
          </span>
        </div>
        <div class="stats-content">
          <div class="stats-value">{{ runningCount }}</div>
          <div class="stats-subtitle">ETL Jobs</div>
        </div>
      </div>
      
      <div class="layout-card">
        <div class="stats-header">
          <span class="stats-title">Completed</span>
          <span class="stats-icon-box">
            <i class="pi pi-check-circle"></i>
          </span>
        </div>
        <div class="stats-content">
          <div class="stats-value">{{ completedCount }}</div>
          <div class="stats-subtitle">Success</div>
        </div>
      </div>
      
      <div class="layout-card">
        <div class="stats-header">
          <span class="stats-title">Failed</span>
          <span class="stats-icon-box">
            <i class="pi pi-times-circle"></i>
          </span>
        </div>
        <div class="stats-content">
          <div class="stats-value">{{ failedCount }}</div>
          <div class="stats-subtitle">Errors</div>
        </div>
      </div>
      
      <div class="layout-card">
        <div class="stats-header">
          <span class="stats-title">Pending</span>
          <span class="stats-icon-box">
            <i class="pi pi-clock"></i>
          </span>
        </div>
        <div class="stats-content">
          <div class="stats-value">{{ pendingCount }}</div>
          <div class="stats-subtitle">Queued</div>
        </div>
      </div>
    </div>

      <!-- Job List -->
      <div class="layout-card">
        <ETLJobList 
          :jobs="jobs" 
          @select-job="selectedJobId = $event"
        />
      </div>

      <!-- Log Viewer -->
      <div v-if="selectedJobId" class="layout-card">
        <ETLLogViewer 
          :job-id="selectedJobId"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { etlApi } from '@/services/etlApi'
import ETLJobList from './ETLJobList.vue'
import ETLLogViewer from './ETLLogViewer.vue'

const jobs = ref([])
const selectedJobId = ref(null)
const refreshInterval = ref(null)
const countdownInterval = ref(null)
const countdown = ref(5)

const runningCount = computed(() => 
  jobs.value.filter(j => j.status === 'running').length
)
const completedCount = computed(() => 
  jobs.value.filter(j => j.status === 'completed').length
)
const failedCount = computed(() => 
  jobs.value.filter(j => j.status === 'failed' || j.status === 'completed_with_errors').length
)
const pendingCount = computed(() => 
  jobs.value.filter(j => j.status === 'pending').length
)

const fetchJobs = async () => {
  try {
    const response = await etlApi.getJobs()
    jobs.value = response.data.jobs || []
  } catch (error) {
    console.error('Failed to fetch ETL jobs:', error)
  }
}

const startPolling = () => {
  // Reset countdown
  countdown.value = 5
  
  // Set up countdown timer (every 1 second)
  countdownInterval.value = setInterval(() => {
    countdown.value--
    if (countdown.value <= 0) {
      countdown.value = 5
    }
  }, 1000)
  
  // Set up data refresh timer (every 5 seconds)
  refreshInterval.value = setInterval(() => {
    fetchJobs()
  }, 5000)
}

const stopPolling = () => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
    refreshInterval.value = null
  }
  if (countdownInterval.value) {
    clearInterval(countdownInterval.value) 
    countdownInterval.value = null
  }
}

onMounted(() => {
  fetchJobs()
  startPolling()
})

onUnmounted(() => {
  stopPolling()
})
</script>

