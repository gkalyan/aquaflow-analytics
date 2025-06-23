import api from './api'

export const etlApi = {
  // Get all ETL jobs
  getJobs(params = {}) {
    return api.get('/etl/jobs', { params })
  },

  // Get job details
  getJobDetails(jobId) {
    return api.get(`/etl/jobs/${jobId}`)
  },

  // Get job logs
  getJobLogs(jobId, params = {}) {
    return api.get(`/etl/jobs/${jobId}/logs`, { params })
  },

  // Restart a failed job
  restartJob(jobId) {
    return api.post(`/etl/jobs/${jobId}/restart`)
  }
}