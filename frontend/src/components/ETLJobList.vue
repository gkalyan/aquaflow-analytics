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
      <Column field="job_name" header="Job Name" sortable></Column>
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
</script>