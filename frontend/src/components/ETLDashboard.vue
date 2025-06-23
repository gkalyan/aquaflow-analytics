<template>
  <div class="etl-dashboard">
    <h2 class="text-2xl font-bold mb-4">ETL Jobs Monitor</h2>
    
    <!-- Job Status Summary -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
      <div class="bg-white rounded-lg shadow p-4">
        <div class="text-sm text-gray-500">Running</div>
        <div class="text-2xl font-bold text-blue-600">{{ runningCount }}</div>
      </div>
      <div class="bg-white rounded-lg shadow p-4">
        <div class="text-sm text-gray-500">Completed</div>
        <div class="text-2xl font-bold text-green-600">{{ completedCount }}</div>
      </div>
      <div class="bg-white rounded-lg shadow p-4">
        <div class="text-sm text-gray-500">Failed</div>
        <div class="text-2xl font-bold text-red-600">{{ failedCount }}</div>
      </div>
      <div class="bg-white rounded-lg shadow p-4">
        <div class="text-sm text-gray-500">Pending</div>
        <div class="text-2xl font-bold text-yellow-600">{{ pendingCount }}</div>
      </div>
    </div>

    <!-- Job List -->
    <ETLJobList 
      :jobs="jobs" 
      @select-job="selectedJobId = $event"
      @refresh="fetchJobs"
    />

    <!-- Log Viewer -->
    <ETLLogViewer 
      v-if="selectedJobId"
      :job-id="selectedJobId"
      class="mt-6"
    />
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

onMounted(() => {
  fetchJobs()
  // Refresh every 5 seconds
  refreshInterval.value = setInterval(fetchJobs, 5000)
})

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
})
</script>

<style scoped>
.etl-dashboard {
  padding: 1rem;
}
</style>