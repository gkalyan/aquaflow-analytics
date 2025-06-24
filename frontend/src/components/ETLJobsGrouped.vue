<template>
  <div>
    <div class="flex justify-content-between align-items-center mb-4">
      <h3 class="text-900 font-bold text-xl m-0">ETL Jobs</h3>
      <div class="flex align-items-center gap-2">
        <Button 
          label="Show All Runs" 
          icon="pi pi-list" 
          size="small"
          :outlined="!showAllRuns"
          @click="showAllRuns = !showAllRuns"
        />
      </div>
    </div>
    
    <!-- Grouped View -->
    <div v-if="!showAllRuns" class="etl-jobs-grouped">
      <DataTable 
        :value="groupedJobs" 
        :expandedRows="expandedRows"
        @rowExpand="onRowExpand"
        @rowCollapse="onRowCollapse"
        dataKey="job_id"
        class="etl-jobs-table"
      >
        <Column :expander="true" headerStyle="width: 3rem" />
        <Column field="job_name" header="Job Name" sortable>
          <template #body="{ data }">
            <div>
              <div class="font-medium">{{ data.job_name }}</div>
              <div class="text-500 text-sm">{{ data.job_type }}</div>
            </div>
          </template>
        </Column>
        <Column field="latest_status" header="Latest Status">
          <template #body="{ data }">
            <Tag :severity="getStatusSeverity(data.latest_status)">
              {{ data.latest_status }}
            </Tag>
          </template>
        </Column>
        <Column field="run_count" header="Total Runs">
          <template #body="{ data }">
            <span class="font-medium">{{ data.run_count }}</span>
          </template>
        </Column>
        <Column field="schedule" header="Schedule">
          <template #body="{ data }">
            <div v-if="data.schedule" class="text-sm">
              <div class="font-medium">{{ formatSchedule(data.schedule) }}</div>
            </div>
            <span v-else class="text-500 text-sm">Manual</span>
          </template>
        </Column>
        <Column field="next_run" header="Next Run">
          <template #body="{ data }">
            <div v-if="data.next_run" class="text-sm">
              <div class="font-medium">{{ formatNextRun(data.next_run) }}</div>
              <div class="text-500 text-xs">{{ formatTimeUntilNext(data.next_run) }}</div>
            </div>
            <span v-else class="text-500 text-sm">-</span>
          </template>
        </Column>
        <Column header="Actions">
          <template #body="{ data }">
            <Button
              v-if="canRestart(data.latest_run)"
              @click="restartJob(data.latest_run.batch_id)"
              label="Restart Latest"
              size="small"
              outlined
            />
          </template>
        </Column>
        <template #expansion="slotProps">
          <div class="p-3">
            <h5 class="mb-3">Run History</h5>
            <DataTable :value="slotProps.data.runs" class="run-history-table">
              <Column field="run_number" header="Run #" style="width: 80px">
                <template #body="{ data }">
                  <span class="font-medium">#{{ data.run_number }}</span>
                </template>
              </Column>
              <Column field="run_name" header="Run Name">
                <template #body="{ data }">
                  {{ getRunTimestamp(data.job_name) || 'Run ' + data.run_number }}
                </template>
              </Column>
              <Column field="status" header="Status">
                <template #body="{ data }">
                  <Tag :severity="getStatusSeverity(data.status)" size="small">
                    {{ data.status }}
                  </Tag>
                </template>
              </Column>
              <Column field="started_at" header="Started">
                <template #body="{ data }">
                  {{ formatTime(data.started_at) }}
                </template>
              </Column>
              <Column field="records_processed" header="Records">
                <template #body="{ data }">
                  {{ data.records_processed }} / {{ data.records_failed }} failed
                </template>
              </Column>
              <Column header="Actions">
                <template #body="{ data }">
                  <div class="flex gap-2">
                    <Button
                      @click="selectJob(data.batch_id)"
                      icon="pi pi-eye"
                      size="small"
                      outlined
                      v-tooltip="'View Logs'"
                    />
                    <Button
                      v-if="canRestart(data)"
                      @click="restartJob(data.batch_id)"
                      icon="pi pi-refresh"
                      size="small"
                      outlined
                      v-tooltip="'Restart'"
                    />
                  </div>
                </template>
              </Column>
            </DataTable>
          </div>
        </template>
      </DataTable>
    </div>

    <!-- All Runs View (existing table) -->
    <DataTable 
      v-else
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
          <span class="text-sm font-medium">#{{ data.run_number }}</span>
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
import { ref, computed } from 'vue'
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

const showAllRuns = ref(false)
const expandedRows = ref([])

// Group jobs by job_id
const groupedJobs = computed(() => {
  const groups = {}
  
  props.jobs.forEach(job => {
    if (!groups[job.job_id]) {
      groups[job.job_id] = {
        job_id: job.job_id,
        job_name: getBaseJobName(job.job_name),
        job_type: job.job_type,
        schedule: job.schedule,
        next_run: job.next_run,
        runs: [],
        run_count: 0,
        latest_status: null,
        latest_run: null
      }
    }
    
    groups[job.job_id].runs.push(job)
    groups[job.job_id].run_count++
    
    // Track latest run
    if (!groups[job.job_id].latest_run || job.run_number === 1) {
      groups[job.job_id].latest_status = job.status
      groups[job.job_id].latest_run = job
    }
  })
  
  // Sort runs within each group
  Object.values(groups).forEach(group => {
    group.runs.sort((a, b) => b.run_number - a.run_number)
  })
  
  return Object.values(groups)
})

const onRowExpand = (event) => {
  // Optionally load detailed run history
}

const onRowCollapse = (event) => {
  // Handle collapse
}

const onRowSelect = (event) => {
  emit('select-job', event.data.batch_id)
}

const selectJob = (batchId) => {
  emit('select-job', batchId)
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
  if (job.job_type === 'historical_load' && job.parameters?.series_ids) {
    const totalSeries = job.parameters.series_ids.length
    const processedSeries = Math.floor(job.records_processed / 1000)
    return Math.min(100, (processedSeries / totalSeries) * 100)
  }
  return 50
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
  const match = jobName.match(/^(.+?)\\s*[-\\(]/)
  return match ? match[1].trim() : jobName
}

const getRunTimestamp = (jobName) => {
  const timestampMatch = jobName.match(/- (\\d{4}-\\d{2}-\\d{2} \\d{2}:\\d{2})/)
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
</script>

<style scoped>
.run-history-table {
  font-size: 0.875rem;
}

.etl-jobs-grouped :deep(.p-datatable-row-expansion) {
  background-color: var(--surface-50);
}
</style>