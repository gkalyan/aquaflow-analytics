import apiClient from './apiClient'

export const etlApi = {
  // Get all ETL jobs
  getJobs(params = {}) {
    return apiClient.get('/etl/jobs', { params })
  },

  // Get job details
  getJobDetails(jobId) {
    return apiClient.get(`/etl/jobs/${jobId}`)
  },

  // Get job logs
  getJobLogs(jobId, params = {}) {
    return apiClient.get(`/etl/jobs/${jobId}/logs`, { params })
  },

  // Restart a failed job
  restartJob(jobId) {
    return apiClient.post(`/etl/jobs/${jobId}/restart`)
  }
}