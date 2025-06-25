<template>
  <div class="data-card">
    <!-- Simple Value Display -->
    <div v-if="isSimpleValue" class="simple-value">
      <div class="value-container">
        <span class="value">{{ data.value }}</span>
        <span v-if="data.unit" class="unit">{{ data.unit }}</span>
      </div>
      <div class="value-meta">
        <span v-if="data.location" class="location">{{ data.location }}</span>
        <span v-if="data.parameter" class="parameter">{{ data.parameter }}</span>
        <span v-if="data.status" :class="['status', data.status]">{{ getStatusLabel(data.status) }}</span>
      </div>
      <div v-if="data.timestamp" class="timestamp">
        Last updated: {{ formatTimestamp(data.timestamp) }}
      </div>
    </div>

    <!-- Table Display -->
    <div v-else-if="isTableData" class="table-data">
      <div class="table-header">
        <h4>{{ getTableTitle() }}</h4>
      </div>
      <table class="data-table">
        <thead>
          <tr>
            <th v-for="column in tableColumns" :key="column">
              {{ formatColumnName(column) }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, index) in tableRows" :key="index">
            <td v-for="column in tableColumns" :key="column">
              <span v-if="column === 'status'" :class="['status-badge', row[column]]">
                {{ getStatusLabel(row[column]) }}
              </span>
              <span v-else-if="isNumericValue(row[column])">
                {{ formatNumber(row[column]) }}
              </span>
              <span v-else>{{ row[column] || '-' }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Chart Display -->
    <div v-else-if="isChartData" class="chart-data">
      <div class="chart-header">
        <h4>{{ getChartTitle() }}</h4>
      </div>
      <div class="chart-container">
        <!-- Simple line chart representation -->
        <div class="mini-chart">
          <div
            v-for="(point, index) in chartPoints"
            :key="index"
            class="chart-point"
            :style="{ height: point.height + '%', left: point.x + '%' }"
            :title="`${point.value} at ${point.time}`"
          ></div>
        </div>
        <div class="chart-summary">
          <div class="summary-item">
            <label>Current:</label>
            <span>{{ getCurrentValue() }}</span>
          </div>
          <div class="summary-item">
            <label>Average:</label>
            <span>{{ getAverageValue() }}</span>
          </div>
          <div class="summary-item">
            <label>Range:</label>
            <span>{{ getValueRange() }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Error Display -->
    <div v-else-if="isErrorData" class="error-data">
      <i class="pi pi-exclamation-triangle"></i>
      <span>{{ data.error || 'Error loading data' }}</span>
    </div>

    <!-- Generic Object Display -->
    <div v-else class="generic-data">
      <pre>{{ JSON.stringify(data, null, 2) }}</pre>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  data: {
    type: [Object, Array, String, Number],
    required: true
  }
})

// Computed properties to determine data type
const isSimpleValue = computed(() => {
  return props.data && 
         typeof props.data === 'object' && 
         'value' in props.data &&
         !Array.isArray(props.data)
})

const isTableData = computed(() => {
  return Array.isArray(props.data) && 
         props.data.length > 0 && 
         typeof props.data[0] === 'object'
})

const isChartData = computed(() => {
  return props.data && 
         typeof props.data === 'object' && 
         ('series' in props.data || 'timeseries' in props.data)
})

const isErrorData = computed(() => {
  return props.data && 
         typeof props.data === 'object' && 
         'error' in props.data
})

// Table data processing
const tableColumns = computed(() => {
  if (!isTableData.value) return []
  return Object.keys(props.data[0])
})

const tableRows = computed(() => {
  if (!isTableData.value) return []
  return props.data
})

// Chart data processing
const chartPoints = computed(() => {
  if (!isChartData.value) return []
  
  const series = props.data.series || props.data.timeseries || []
  if (!series.length) return []
  
  const values = series.map(point => point.value || point.y || 0)
  const maxValue = Math.max(...values)
  const minValue = Math.min(...values)
  const range = maxValue - minValue || 1
  
  return series.map((point, index) => ({
    value: point.value || point.y || 0,
    time: point.timestamp || point.time || point.x,
    height: ((point.value || point.y || 0) - minValue) / range * 100,
    x: (index / (series.length - 1)) * 100
  }))
})

// Methods
const getStatusLabel = (status) => {
  const statusLabels = {
    'normal': 'Normal',
    'warning': 'Warning', 
    'alert': 'Alert',
    'error': 'Error',
    'good': 'Good',
    'bad': 'Bad',
    'ok': 'OK'
  }
  return statusLabels[status] || status
}

const formatTimestamp = (timestamp) => {
  if (!timestamp) return ''
  
  try {
    const date = new Date(timestamp)
    return date.toLocaleString()
  } catch (error) {
    return timestamp
  }
}

const getTableTitle = () => {
  if (tableRows.value.length === 1) {
    return 'System Status'
  }
  return `Data Results (${tableRows.value.length} items)`
}

const getChartTitle = () => {
  return props.data.title || 'Time Series Data'
}

const formatColumnName = (column) => {
  return column
    .replace(/_/g, ' ')
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/\b\w/g, l => l.toUpperCase())
}

const isNumericValue = (value) => {
  return typeof value === 'number' || (typeof value === 'string' && !isNaN(value))
}

const formatNumber = (value) => {
  const num = Number(value)
  if (isNaN(num)) return value
  
  if (num >= 1000) {
    return num.toLocaleString()
  }
  
  return num % 1 === 0 ? num.toString() : num.toFixed(2)
}

const getCurrentValue = () => {
  if (!chartPoints.value.length) return '-'
  const latest = chartPoints.value[chartPoints.value.length - 1]
  return formatNumber(latest.value)
}

const getAverageValue = () => {
  if (!chartPoints.value.length) return '-'
  const sum = chartPoints.value.reduce((acc, point) => acc + point.value, 0)
  return formatNumber(sum / chartPoints.value.length)
}

const getValueRange = () => {
  if (!chartPoints.value.length) return '-'
  const values = chartPoints.value.map(p => p.value)
  const min = Math.min(...values)
  const max = Math.max(...values)
  return `${formatNumber(min)} - ${formatNumber(max)}`
}
</script>

<style scoped>
.data-card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 1rem;
  margin-top: 0.5rem;
}

/* Simple Value Display */
.simple-value {
  text-align: center;
}

.value-container {
  display: flex;
  align-items: baseline;
  justify-content: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.value {
  font-size: 2rem;
  font-weight: 700;
  color: #1f2937;
}

.unit {
  font-size: 1rem;
  color: #6b7280;
  font-weight: 500;
}

.value-meta {
  display: flex;
  justify-content: center;
  gap: 1rem;
  margin-bottom: 0.5rem;
  flex-wrap: wrap;
}

.location, .parameter {
  font-size: 0.875rem;
  color: #4b5563;
}

.status {
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
}

.status.normal, .status.good, .status.ok {
  background: #d1fae5;
  color: #065f46;
}

.status.warning {
  background: #fef3c7;
  color: #92400e;
}

.status.alert, .status.error, .status.bad {
  background: #fee2e2;
  color: #991b1b;
}

.timestamp {
  font-size: 0.75rem;
  color: #9ca3af;
}

/* Table Display */
.table-data {
  overflow-x: auto;
}

.table-header h4 {
  margin: 0 0 1rem 0;
  color: #1f2937;
  font-size: 1rem;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.data-table th {
  background: #f8fafc;
  padding: 0.5rem;
  text-align: left;
  border-bottom: 1px solid #e2e8f0;
  font-weight: 600;
  color: #374151;
}

.data-table td {
  padding: 0.5rem;
  border-bottom: 1px solid #f1f5f9;
}

.status-badge {
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

.status-badge.normal, .status-badge.good {
  background: #d1fae5;
  color: #065f46;
}

.status-badge.warning {
  background: #fef3c7;
  color: #92400e;
}

.status-badge.alert, .status-badge.error {
  background: #fee2e2;
  color: #991b1b;
}

/* Chart Display */
.chart-data {
  min-height: 200px;
}

.chart-header h4 {
  margin: 0 0 1rem 0;
  color: #1f2937;
  font-size: 1rem;
}

.chart-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.mini-chart {
  position: relative;
  height: 100px;
  background: #f8fafc;
  border-radius: 4px;
  overflow: hidden;
}

.chart-point {
  position: absolute;
  bottom: 0;
  width: 2px;
  background: #2563eb;
  transition: all 0.2s;
  cursor: pointer;
}

.chart-point:hover {
  background: #1d4ed8;
  width: 4px;
}

.chart-summary {
  display: flex;
  justify-content: space-around;
  background: #f8fafc;
  padding: 0.75rem;
  border-radius: 4px;
}

.summary-item {
  text-align: center;
}

.summary-item label {
  display: block;
  font-size: 0.75rem;
  color: #6b7280;
  margin-bottom: 0.25rem;
}

.summary-item span {
  font-weight: 600;
  color: #1f2937;
}

/* Error Display */
.error-data {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #dc2626;
  background: #fef2f2;
  padding: 1rem;
  border-radius: 4px;
  border: 1px solid #fecaca;
}

/* Generic Display */
.generic-data {
  background: #f8fafc;
  padding: 1rem;
  border-radius: 4px;
  overflow-x: auto;
}

.generic-data pre {
  margin: 0;
  font-size: 0.75rem;
  color: #374151;
}

/* Responsive Design */
@media (max-width: 640px) {
  .value-meta {
    flex-direction: column;
    gap: 0.25rem;
  }
  
  .chart-summary {
    flex-direction: column;
    gap: 0.5rem;
  }
  
  .data-table {
    font-size: 0.75rem;
  }
  
  .data-table th,
  .data-table td {
    padding: 0.375rem;
  }
}
</style>