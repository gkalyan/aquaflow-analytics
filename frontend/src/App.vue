<template>
  <div id="app" class="min-h-screen bg-gray-100">
    <!-- Header -->
    <header class="bg-blue-600 text-white shadow-sm">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex items-center justify-between h-16">
          <div class="flex items-center">
            <h1 class="text-xl font-semibold">AquaFlow Analytics</h1>
            <span class="ml-2 text-blue-200 text-sm">Daily Operations Assistant</span>
          </div>
          <div class="flex items-center space-x-4">
            <span class="text-sm">Welcome, Olivia</span>
            <div class="h-2 w-2 rounded-full bg-green-400" title="System Online"></div>
          </div>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
      <div class="px-4 py-6 sm:px-0">
        <div class="text-center mb-8">
          <h2 class="text-3xl font-bold text-gray-900 mb-4">
            Ask about your water system
          </h2>
          <p class="text-lg text-gray-600">
            Get operational answers in seconds, not minutes
          </p>
        </div>

        <!-- Query Interface -->
        <div class="max-w-2xl mx-auto">
          <div class="bg-white shadow rounded-lg p-6">
            <div class="mb-4">
              <input
                type="text"
                v-model="query"
                @keyup.enter="submitQuery"
                placeholder="Ask about your water system... (e.g., 'morning check', 'main canal flow status')"
                class="w-full px-4 py-3 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            
            <div class="flex justify-between items-center mb-4">
              <button
                @click="submitQuery"
                :disabled="!query.trim()"
                class="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700 disabled:bg-gray-300 disabled:cursor-not-allowed"
              >
                Ask Question
              </button>
              
              <div class="flex space-x-2">
                <button
                  @click="quickQuery('morning check')"
                  class="bg-gray-200 text-gray-700 px-3 py-1 rounded text-sm hover:bg-gray-300"
                >
                  Morning Check
                </button>
                <button
                  @click="quickQuery('system status')"
                  class="bg-gray-200 text-gray-700 px-3 py-1 rounded text-sm hover:bg-gray-300"
                >
                  System Status
                </button>
              </div>
            </div>

            <!-- Response Area -->
            <div v-if="response" class="border-t pt-4">
              <div class="bg-gray-50 rounded-md p-4">
                <h3 class="font-medium text-gray-900 mb-2">Response:</h3>
                <p class="text-gray-700">{{ response }}</p>
                <div class="mt-2 text-xs text-gray-500">
                  Response time: {{ responseTime }}ms
                </div>
              </div>
            </div>

            <!-- Loading State -->
            <div v-if="loading" class="border-t pt-4">
              <div class="flex items-center space-x-2 text-gray-600">
                <div class="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
                <span>Processing your question...</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Quick Stats -->
        <div class="mt-8 grid grid-cols-1 md:grid-cols-3 gap-6">
          <div class="bg-white shadow rounded-lg p-6">
            <h3 class="text-lg font-medium text-gray-900 mb-2">System Status</h3>
            <p class="text-sm text-gray-600">All systems operational</p>
            <div class="mt-2 flex items-center">
              <div class="h-3 w-3 rounded-full bg-green-500 mr-2"></div>
              <span class="text-sm text-green-600">Online</span>
            </div>
          </div>
          
          <div class="bg-white shadow rounded-lg p-6">
            <h3 class="text-lg font-medium text-gray-900 mb-2">Data Freshness</h3>
            <p class="text-sm text-gray-600">Last update: {{ lastUpdate }}</p>
            <div class="mt-2 flex items-center">
              <div class="h-3 w-3 rounded-full bg-blue-500 mr-2"></div>
              <span class="text-sm text-blue-600">Real-time</span>
            </div>
          </div>
          
          <div class="bg-white shadow rounded-lg p-6">
            <h3 class="text-lg font-medium text-gray-900 mb-2">Time Saved Today</h3>
            <p class="text-sm text-gray-600">{{ timeSaved }} minutes</p>
            <div class="mt-2 flex items-center">
              <div class="h-3 w-3 rounded-full bg-purple-500 mr-2"></div>
              <span class="text-sm text-purple-600">Efficiency</span>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

const query = ref('')
const response = ref('')
const loading = ref(false)
const responseTime = ref(0)
const lastUpdate = ref('')
const timeSaved = ref(0)

const submitQuery = async () => {
  if (!query.value.trim()) return
  
  loading.value = true
  response.value = ''
  
  const startTime = Date.now()
  
  try {
    // TODO: Replace with actual API call
    await new Promise(resolve => setTimeout(resolve, 500)) // Simulate API call
    
    response.value = `This is a demo response for: "${query.value}". The actual query processing system will be implemented to connect with your SCADA systems and provide real operational data.`
    responseTime.value = Date.now() - startTime
    timeSaved.value += Math.floor(Math.random() * 10) + 15 // Simulate time saved
    
  } catch (error) {
    response.value = 'Error processing your question. Please try again.'
  } finally {
    loading.value = false
  }
}

const quickQuery = (queryText) => {
  query.value = queryText
  submitQuery()
}

onMounted(() => {
  // Set current time as last update
  lastUpdate.value = new Date().toLocaleTimeString()
  
  // Update time every 30 seconds to simulate real-time data
  setInterval(() => {
    lastUpdate.value = new Date().toLocaleTimeString()
  }, 30000)
})</script>