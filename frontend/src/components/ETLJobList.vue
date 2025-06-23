<template>
  <div class="etl-job-list bg-white rounded-lg shadow">
    <div class="px-4 py-3 border-b border-gray-200 flex justify-between items-center">
      <h3 class="text-lg font-semibold">ETL Jobs</h3>
      <button 
        @click="$emit('refresh')"
        class="text-sm px-3 py-1 bg-blue-500 text-white rounded hover:bg-blue-600"
      >
        Refresh
      </button>
    </div>
    
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Job Name
            </th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Type
            </th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Progress
            </th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Started
            </th>
            <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr 
            v-for="job in jobs" 
            :key="job.batch_id"
            @click="$emit('select-job', job.batch_id)"
            class="hover:bg-gray-50 cursor-pointer"
          >
            <td class="px-4 py-3 whitespace-nowrap text-sm font-medium text-gray-900">
              {{ job.job_name }}
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
              {{ job.job_type }}
            </td>
            <td class="px-4 py-3 whitespace-nowrap">
              <span 
                :class="getStatusClass(job.status)"
                class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full"
              >
                {{ job.status }}
              </span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
              <div v-if="job.status === 'running'" class="flex items-center">
                <div class="w-24 bg-gray-200 rounded-full h-2 mr-2">
                  <div 
                    class="bg-blue-600 h-2 rounded-full"
                    :style="`width: ${getProgress(job)}%`"
                  ></div>
                </div>
                <span class="text-xs">{{ job.records_processed }}</span>
              </div>
              <span v-else>
                {{ job.records_processed }} / {{ job.records_failed || 0 }} failed
              </span>
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
              {{ formatTime(job.started_at) }}
            </td>
            <td class="px-4 py-3 whitespace-nowrap text-sm font-medium">
              <button
                v-if="canRestart(job)"
                @click.stop="restartJob(job.batch_id)"
                class="text-blue-600 hover:text-blue-900"
              >
                Restart
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { etlApi } from '@/services/etlApi'
import { useToast } from 'primevue/usetoast'

const props = defineProps({
  jobs: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['select-job', 'refresh'])
const toast = useToast()

const getStatusClass = (status) => {
  const classes = {
    'pending': 'bg-yellow-100 text-yellow-800',
    'running': 'bg-blue-100 text-blue-800',
    'completed': 'bg-green-100 text-green-800',
    'failed': 'bg-red-100 text-red-800',
    'completed_with_errors': 'bg-orange-100 text-orange-800'
  }
  return classes[status] || 'bg-gray-100 text-gray-800'
}

const getProgress = (job) => {
  // Estimate progress based on time or records
  if (job.job_type === 'historical_load' && job.parameters?.series_ids) {
    const totalSeries = job.parameters.series_ids.length
    const processedSeries = Math.floor(job.records_processed / 1000) // Rough estimate
    return Math.min(100, (processedSeries / totalSeries) * 100)
  }
  return 50 // Default for running jobs
}

const formatTime = (timestamp) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now - date
  
  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  
  return date.toLocaleString()
}

const canRestart = (job) => {
  return job.status === 'failed' || job.status === 'completed_with_errors'
}

const restartJob = async (batchId) => {
  try {
    await etlApi.restartJob(batchId)
    toast.add({ 
      severity: 'success', 
      summary: 'Job Restarted', 
      detail: 'ETL job has been queued for restart',
      life: 3000 
    })
    emit('refresh')
  } catch (error) {
    toast.add({ 
      severity: 'error', 
      summary: 'Restart Failed', 
      detail: error.message,
      life: 3000 
    })
  }
}
</script>