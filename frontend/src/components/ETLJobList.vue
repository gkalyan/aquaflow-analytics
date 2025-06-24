<template>
  <div>
    <div class="flex justify-content-between align-items-center mb-4">
      <h3 class="text-900 font-bold text-xl m-0">ETL Jobs</h3>
    </div>
    
    <DataTable 
      :value="jobs" 
      selectionMode="single"
      @rowSelect="onRowSelect"
      class="etl-jobs-table"
    >
      <Column field="job_name" header="Job Name" sortable>
        <template #body="{ data }">
          <div>
            <div class="font-medium">{{ getBaseJobName(data.job_name) }}</div>
            <div class="text-500 text-sm">{{ getRunTimestamp(data.job_name) }}</div>
          </div>
        </template>
      </Column>
      <Column field="run_number" header="Run #" style="width: 80px">
        <template #body="{ data }">
          <span class="text-sm font-medium">#{{ getRunNumber(data) }}</span>
        </template>
      </Column>
      <Column field="job_type" header="Type" sortable></Column>
      <Column field="status" header="Status">
        <template #body="{ data }">
          <Tag :severity="getStatusSeverity(data.status)">
            {{ data.status }}
          </Tag>
        </template>
      </Column>
      <Column field="progress" header="Progress">
        <template #body="{ data }">
          <div v-if="data.status === 'running'" class="flex align-items-center gap-2">
            <ProgressBar :value="getProgress(data)" style="width: 6rem; height: 0.5rem" />
            <span class="text-sm">{{ data.records_processed }}</span>
          </div>
          <span v-else class="text-sm">
            {{ data.records_processed }} / {{ data.records_failed || 0 }} failed
          </span>
        </template>
      </Column>
      <Column field="started_at" header="Started">
        <template #body="{ data }">
          {{ formatTime(data.started_at) }}
        </template>
      </Column>
      <Column field="schedule" header="Schedule" style="min-width: 120px">
        <template #body="{ data }">
          <div v-if="data.schedule" class="text-sm">
            <div class="font-medium">{{ formatSchedule(data.schedule) }}</div>
          </div>
          <span v-else class="text-500 text-sm">Manual</span>
        </template>
      </Column>
      <Column field="next_run" header="Next Load" style="min-width: 130px">
        <template #body="{ data }">
          <div v-if="data.next_run && data.status !== 'running'" class="text-sm">
            <div class="font-medium">{{ formatNextRun(data.next_run) }}</div>
            <div class="text-500 text-xs">{{ formatTimeUntilNext(data.next_run) }}</div>
          </div>
          <span v-else-if="data.status === 'running'" class="text-500 text-sm">Running now</span>
          <span v-else class="text-500 text-sm">-</span>
        </template>
      </Column>
      <Column header="Actions">
        <template #body="{ data }">
          <Button
            v-if="canRestart(data)"
            @click="restartJob(data.batch_id)"
            label="Restart"
            size="small"
            outlined
          />
        </template>
      </Column>
    </DataTable>
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

const emit = defineEmits(['select-job'])
const toast = useToast()

const onRowSelect = (event) => {
  emit('select-job', event.data.batch_id)
}

const getStatusSeverity = (status) => {
  const severities = {
    'pending': 'warn',
    'running': 'info', 
    'completed': 'success',
    'failed': 'danger',
    'completed_with_errors': 'warn'
  }
  return severities[status] || 'secondary'
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
    // Automatic refresh will pick up the change in 5 seconds
  } catch (error) {
    toast.add({ 
      severity: 'error', 
      summary: 'Restart Failed', 
      detail: error.message,
      life: 3000 
    })
  }
}

const formatSchedule = (schedule) => {
  if (!schedule) return 'Manual'
  
  // Handle common cron patterns and custom formats
  const scheduleMap = {
    '*/15 * * * *': 'Every 15m',
    '0 * * * *': 'Hourly',
    '0 */2 * * *': 'Every 2h',
    '0 */6 * * *': 'Every 6h',
    '0 2 * * *': 'Daily at 2AM',
    '0 0 * * 0': 'Weekly',
    '0 0 1 * *': 'Monthly'
  }
  
  return scheduleMap[schedule] || schedule
}

const formatNextRun = (nextRun) => {
  if (!nextRun) return '-'
  const date = new Date(nextRun)
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatTimeUntilNext = (nextRun) => {
  if (!nextRun) return ''
  
  const now = new Date()
  const next = new Date(nextRun)
  const diff = next - now
  
  if (diff <= 0) return 'Overdue'
  
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  if (days > 0) return `in ${days}d ${hours % 24}h`
  if (hours > 0) return `in ${hours}h ${minutes % 60}m`
  if (minutes > 0) return `in ${minutes}m`
  
  return 'in <1m'
}

const getBaseJobName = (jobName) => {
  // Extract base job name without timestamp
  // Examples: "Hourly Flow Sync - 2025-06-24 07:00" -> "Hourly Flow Sync"
  //          "Real-time Data Sync (Manual)" -> "Real-time Data Sync"
  const match = jobName.match(/^(.+?)\s*[-\(]/)
  return match ? match[1].trim() : jobName
}

const getRunTimestamp = (jobName) => {
  // Extract timestamp or manual indicator
  const timestampMatch = jobName.match(/- (\d{4}-\d{2}-\d{2} \d{2}:\d{2})/)
  if (timestampMatch) {
    const date = new Date(timestampMatch[1])
    return date.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }
  
  if (jobName.includes('(Manual)')) {
    return 'Manual Run'
  }
  
  return ''
}

const getRunNumber = (job) => {
  // Use the run_number from the backend if available
  return job.run_number || 1
}
</script>