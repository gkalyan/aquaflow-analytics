<template>
  <div class="layout-wrapper layout-static">
    <Toast />
    
    <!-- Sidebar -->
    <div 
      class="layout-sidebar"
      :class="{ 'layout-sidebar-mobile-active': sidebarVisible }"
    >
      <div class="layout-sidebar-header">
        <div class="flex align-items-center px-4 py-3">
          <i class="pi pi-database text-primary text-2xl mr-3"></i>
          <div>
            <div class="text-900 font-semibold text-lg">AquaFlow</div>
            <div class="text-500 text-sm">Water District Operations</div>
          </div>
        </div>
      </div>
      
      <div class="layout-menu-container">
        <PanelMenu :model="menuItems" class="w-full border-noround" />
      </div>
    </div>

    <!-- Main Content Area -->
    <div class="layout-main-container">
      <!-- Top Bar -->
      <div class="layout-topbar">
        <div class="layout-topbar-start">
          <Button 
            icon="pi pi-bars" 
            class="p-button-text p-button-plain p-button-rounded" 
            @click="toggleSidebar"
          />
          <span class="layout-topbar-separator"></span>
          <div class="layout-topbar-item-text hidden lg:block">Dashboard</div>
        </div>

        <div class="layout-topbar-end">
          <div class="layout-topbar-item">
            <Avatar 
              :label="authStore.user?.name?.charAt(0) || 'A'"
              class="mr-2"
              shape="circle"
              style="background-color: var(--primary-color); color: white;"
            />
            <div class="hidden lg:block">
              <div class="text-900 font-medium">{{ authStore.user?.name || 'Administrator' }}</div>
              <div class="text-500 text-sm">{{ authStore.user?.role || 'Admin' }}</div>
            </div>
            <Button 
              icon="pi pi-angle-down" 
              class="p-button-text p-button-plain p-button-rounded ml-2"
              @click="toggleUserMenu"
            />
          </div>
        </div>
      </div>

      <!-- Main Content -->
      <div class="layout-main">
        <div class="card">
          <!-- Welcome Section -->
          <div class="text-center mb-6">
            <h2 class="text-3xl font-bold text-900 mb-3">
              Ask about your water system
            </h2>
            <p class="text-lg text-600 line-height-3 m-0">
              Get operational answers in seconds, not minutes
            </p>
          </div>

          <!-- Core Query Interface -->
          <div class="max-w-4xl mx-auto">
            <div class="grid">
              <div class="col-12">
                <div class="p-inputgroup mb-4">
                  <InputText 
                    v-model="query"
                    @keyup.enter="submitQuery"
                    placeholder="Ask about your water system... (e.g., 'morning check', 'main canal flow status')"
                    class="w-full text-lg"
                    style="padding: 1.25rem;"
                  />
                  <Button 
                    icon="pi pi-search" 
                    @click="submitQuery"
                    :loading="loading"
                    :disabled="!query.trim()"
                    size="large"
                    style="padding: 1.25rem 2rem;"
                  />
                </div>
              </div>
              
              <!-- Quick Action Buttons -->
              <div class="col-12">
                <div class="flex flex-wrap gap-2 justify-content-center mb-4">
                  <Button 
                    label="Morning Check" 
                    outlined 
                    @click="quickQuery('morning check')"
                    icon="pi pi-sun"
                  />
                  <Button 
                    label="System Status" 
                    outlined 
                    @click="quickQuery('system status')"
                    icon="pi pi-cog"
                  />
                  <Button 
                    label="Canal Flow Status" 
                    outlined 
                    @click="quickQuery('main canal flow status')"
                    icon="pi pi-chart-line"
                  />
                  <Button 
                    label="Reservoir Levels" 
                    outlined 
                    @click="quickQuery('reservoir levels')"
                    icon="pi pi-database"
                  />
                </div>
              </div>
            </div>

            <!-- Response Area -->
            <div v-if="loading" class="text-center py-8">
              <ProgressSpinner style="width: 60px; height: 60px" strokeWidth="4" />
              <p class="mt-4 text-600 text-lg">Processing your question...</p>
            </div>
            
            <div v-else-if="response" class="mt-6">
              <Message severity="info" :closable="false" class="text-left">
                <div class="flex align-items-start">
                  <i class="pi pi-check-circle text-xl mr-3 mt-1"></i>
                  <div class="flex-1">
                    <h4 class="m-0 mb-3 text-900">Response</h4>
                    <p class="m-0 line-height-3 text-700">{{ response }}</p>
                    <div v-if="responseTime" class="text-500 text-sm mt-3">
                      <i class="pi pi-clock mr-1"></i>
                      Response time: {{ responseTime }}ms
                    </div>
                  </div>
                </div>
              </Message>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- User Menu Overlay -->
    <OverlayPanel ref="userMenu" class="w-12rem">
      <div class="flex flex-column">
        <Button label="Profile" icon="pi pi-user" class="p-button-text text-left" />
        <Button label="Settings" icon="pi pi-cog" class="p-button-text text-left" />
        <Divider />
        <Button label="Logout" icon="pi pi-sign-out" class="p-button-text text-left" @click="handleLogout" />
      </div>
    </OverlayPanel>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '../stores/auth'
import chatApi from '../services/chatApi'
import { v4 as uuidv4 } from 'uuid'

// PrimeVue Components
import Toast from 'primevue/toast'
import Button from 'primevue/button'
import PanelMenu from 'primevue/panelmenu'
import Avatar from 'primevue/avatar'
import InputText from 'primevue/inputtext'
import ProgressSpinner from 'primevue/progressspinner'
import Message from 'primevue/message'
import OverlayPanel from 'primevue/overlaypanel'
import Divider from 'primevue/divider'

const router = useRouter()
const toast = useToast()
const authStore = useAuthStore()

// Refs
const sidebarVisible = ref(false)
const userMenu = ref()
const query = ref('')
const response = ref('')
const loading = ref(false)
const responseTime = ref(0)

// Chat session management
const sessionId = ref(uuidv4())
const userId = ref(null)

// Simplified menu focused on core functionality
const menuItems = ref([
  {
    key: '0',
    label: 'Dashboard',
    icon: 'pi pi-fw pi-home',
    command: () => router.push('/')
  },
  {
    key: '1',
    label: 'Query System',
    icon: 'pi pi-fw pi-search',
    items: [
      {
        key: '1_0',
        label: 'Chat Assistant',
        icon: 'pi pi-fw pi-comments',
        command: () => router.push('/chat')
      },
      {
        key: '1_1',
        label: 'Natural Language',
        icon: 'pi pi-fw pi-comment'
      },
      {
        key: '1_2',
        label: 'Quick Templates',
        icon: 'pi pi-fw pi-list'
      }
    ]
  },
  {
    key: '2',
    label: 'ETL Monitor',
    icon: 'pi pi-fw pi-sync',
    command: () => router.push('/etl')
  }
])

// Methods
const toggleSidebar = () => {
  sidebarVisible.value = !sidebarVisible.value
}

const toggleUserMenu = (event) => {
  userMenu.value.toggle(event)
}

const submitQuery = async () => {
  if (!query.value.trim()) return
  
  loading.value = true
  response.value = ''
  
  const startTime = Date.now()
  
  try {
    // Call the chat API with Ollama integration
    const chatResponse = await chatApi.sendMessage({
      query: query.value.trim(),
      session_id: sessionId.value,
      user_id: userId.value || authStore.user?.id || 'default'
    })
    
    // Handle the response
    if (chatResponse.needs_clarification) {
      response.value = `${chatResponse.response}\n\nClarification: ${chatResponse.clarification_question || 'Please provide more details.'}`
    } else {
      response.value = chatResponse.response
    }
    
    responseTime.value = Date.now() - startTime
    
  } catch (error) {
    console.error('Query error:', error)
    
    // Check if it's a cancelled request
    if (error.code === 'ECONNABORTED' || error.message?.includes('timeout')) {
      toast.add({ 
        severity: 'warn', 
        summary: 'Request Timeout', 
        detail: 'The request took too long. Please try again.', 
        life: 5000 
      })
    } else if (error.response?.status === 401) {
      toast.add({ 
        severity: 'error', 
        summary: 'Authentication Error', 
        detail: 'Please login again', 
        life: 3000 
      })
      setTimeout(() => router.push('/login'), 1500)
    } else if (error.response?.status === 503) {
      toast.add({ 
        severity: 'warn', 
        summary: 'Service Unavailable', 
        detail: 'Chat service is starting up. Please try again in a moment.', 
        life: 5000 
      })
    } else {
      toast.add({ 
        severity: 'error', 
        summary: 'Error', 
        detail: error.response?.data?.error || error.message || 'Failed to process query', 
        life: 3000 
      })
    }
  } finally {
    loading.value = false
  }
}

const quickQuery = (queryText) => {
  query.value = queryText
  submitQuery()
}

const handleLogout = async () => {
  await authStore.logout()
  toast.add({ 
    severity: 'success', 
    summary: 'Success', 
    detail: 'Logged out successfully', 
    life: 3000 
  })
  setTimeout(() => {
    router.push('/login')
  }, 1000)
}

onMounted(() => {
  // Initialize user ID from auth store
  userId.value = authStore.user?.id || authStore.user?.email || 'default'
  
  // Generate new session ID for this dashboard session
  sessionId.value = uuidv4()
})
</script>

<style scoped>
.layout-wrapper {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.layout-sidebar {
  position: fixed;
  width: 300px;
  height: 100vh;
  z-index: 999;
  overflow-y: auto;
  background: var(--surface-overlay);
  border-right: 1px solid var(--surface-border);
  transition: transform 0.2s;
  transform: translateX(0);
}

.layout-sidebar-mobile-active {
  transform: translateX(0);
}

.layout-sidebar-header {
  border-bottom: 1px solid var(--surface-border);
}

.layout-main-container {
  margin-left: 300px;
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.layout-topbar {
  position: fixed;
  top: 0;
  left: 300px;
  right: 0;
  height: 5rem;
  z-index: 997;
  background: var(--surface-overlay);
  border-bottom: 1px solid var(--surface-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 2rem;
}

.layout-topbar-start {
  display: flex;
  align-items: center;
}

.layout-topbar-end {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.layout-topbar-item {
  display: flex;
  align-items: center;
}

.layout-topbar-item-text {
  font-weight: 600;
  color: var(--text-color);
}

.layout-topbar-separator {
  width: 1px;
  height: 2rem;
  background: var(--surface-border);
  margin: 0 1rem;
}

.layout-main {
  margin-top: 5rem;
  padding: 2rem;
  flex: 1;
}

@media (max-width: 991px) {
  .layout-sidebar {
    position: fixed;
    z-index: 1100;
    transform: translateX(-100%);
  }
  
  .layout-sidebar-mobile-active {
    transform: translateX(0);
  }
  
  .layout-main-container {
    margin-left: 0;
  }
  
  .layout-topbar {
    left: 0;
  }
}
</style>