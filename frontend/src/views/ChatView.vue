<template>
  <div class="chat-view">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">
          <i class="pi pi-comments"></i>
          AquaFlow Assistant
        </h1>
        <p class="page-description">
          Ask questions about your water systems using natural language
        </p>
        
        <!-- Connection Status -->
        <div class="connection-status">
          <span :class="['status-dot', connectionStatusClass]"></span>
          <span class="status-text">{{ connectionStatusText }}</span>
          <button 
            v-if="!isConnected" 
            @click="reconnect"
            class="reconnect-btn"
          >
            <i class="pi pi-refresh"></i>
            Reconnect
          </button>
        </div>
      </div>

      <!-- Session Management -->
      <div class="session-controls">
        <div class="current-session-info">
          <span class="session-label">Session:</span>
          <span class="session-id">{{ formatSessionId(currentSessionId) }}</span>
        </div>
        
        <div class="session-actions">
          <button @click="showSessionHistory = true" class="session-btn">
            <i class="pi pi-history"></i>
            History
          </button>
          <button @click="createNewSession" class="session-btn">
            <i class="pi pi-plus"></i>
            New Chat
          </button>
          <button @click="showSettings = true" class="session-btn">
            <i class="pi pi-cog"></i>
            Settings
          </button>
        </div>
      </div>
    </div>

    <!-- Main Chat Area -->
    <div class="chat-container">
      <ChatInterface ref="chatInterface" />
    </div>

    <!-- Session History Dialog -->
    <Dialog
      v-model:visible="showSessionHistory"
      header="Conversation History"
      :modal="true"
      :closable="true"
      style="width: 600px"
    >
      <div class="session-history">
        <div v-if="conversationHistory.length === 0" class="no-history">
          <i class="pi pi-info-circle"></i>
          <p>No previous conversations found.</p>
        </div>
        
        <div
          v-for="conversation in conversationHistory"
          :key="conversation.session_id"
          class="history-item"
          @click="switchToSession(conversation.session_id)"
        >
          <div class="history-content">
            <div class="history-query">{{ conversation.last_query || 'New conversation' }}</div>
            <div class="history-meta">
              <span class="history-time">{{ formatHistoryTime(conversation.last_activity) }}</span>
              <span class="history-messages">{{ conversation.message_count }} messages</span>
              <span class="history-duration">{{ conversation.duration_minutes }}m</span>
            </div>
          </div>
          <i class="pi pi-chevron-right"></i>
        </div>
      </div>
    </Dialog>

    <!-- Settings Dialog -->
    <Dialog
      v-model:visible="showSettings"
      header="Chat Settings"
      :modal="true"
      :closable="true"
      style="width: 500px"
    >
      <div class="settings-content">
        <!-- Entity Shortcuts -->
        <div class="settings-section">
          <h3>Entity Shortcuts</h3>
          <p class="section-description">
            Customize abbreviations for your water systems
          </p>
          
          <div class="entity-shortcuts">
            <div
              v-for="(fullForm, shortcut) in entityShortcuts"
              :key="shortcut"
              class="shortcut-item"
            >
              <span class="shortcut">{{ shortcut }}</span>
              <span class="arrow">→</span>
              <span class="full-form">{{ fullForm }}</span>
              <button
                @click="removeShortcut(shortcut)"
                class="remove-btn"
                title="Remove shortcut"
              >
                <i class="pi pi-times"></i>
              </button>
            </div>
          </div>

          <div class="add-shortcut">
            <input
              v-model="newShortcut"
              placeholder="Abbreviation (e.g., MC)"
              class="shortcut-input"
            />
            <input
              v-model="newFullForm"
              placeholder="Full form (e.g., Main Canal)"
              class="shortcut-input"
            />
            <button
              @click="addShortcut"
              :disabled="!newShortcut || !newFullForm"
              class="add-btn"
            >
              <i class="pi pi-plus"></i>
              Add
            </button>
          </div>
        </div>

        <!-- Preferences -->
        <div class="settings-section">
          <h3>Preferences</h3>
          
          <div class="preference-item">
            <label for="detail-level">Response Detail Level:</label>
            <select 
              id="detail-level"
              v-model="userPreferences.detail_level"
              @change="savePreferences"
            >
              <option value="brief">Brief</option>
              <option value="normal">Normal</option>
              <option value="detailed">Detailed</option>
            </select>
          </div>

          <div class="preference-item">
            <label for="preferred-units">Preferred Units:</label>
            <select
              id="preferred-units"
              v-model="userPreferences.preferred_units"
              @change="savePreferences"
            >
              <option value="imperial">Imperial (CFS, PSI, feet)</option>
              <option value="metric">Metric (m³/s, kPa, meters)</option>
            </select>
          </div>
        </div>

        <!-- Learning Analytics -->
        <div class="settings-section">
          <h3>Learning Analytics</h3>
          <div class="analytics-grid">
            <div class="analytic-item">
              <div class="analytic-value">{{ learningAnalytics.total_queries }}</div>
              <div class="analytic-label">Total Queries</div>
            </div>
            <div class="analytic-item">
              <div class="analytic-value">{{ successRate }}%</div>
              <div class="analytic-label">Success Rate</div>
            </div>
            <div class="analytic-item">
              <div class="analytic-value">{{ learningAnalytics.feedback_submissions }}</div>
              <div class="analytic-label">Feedback Given</div>
            </div>
            <div class="analytic-item">
              <div class="analytic-value">{{ Object.keys(learningAnalytics.learned_entities).length }}</div>
              <div class="analytic-label">Learned Terms</div>
            </div>
          </div>
        </div>
      </div>
    </Dialog>

    <!-- Error Banner -->
    <div v-if="lastError" class="error-banner">
      <i class="pi pi-exclamation-triangle"></i>
      <span>{{ lastError }}</span>
      <button @click="clearError" class="close-error">
        <i class="pi pi-times"></i>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import Dialog from 'primevue/dialog'
import ChatInterface from '@/components/ChatInterface.vue'
import { useChatStore } from '@/stores/chat'

// Store
const chatStore = useChatStore()

// Reactive state
const showSessionHistory = ref(false)
const showSettings = ref(false)
const conversationHistory = ref([])
const newShortcut = ref('')
const newFullForm = ref('')

// Store getters
const currentSessionId = computed(() => chatStore.currentSessionId)
const isConnected = computed(() => chatStore.isConnected)
const lastError = computed(() => chatStore.lastError)
const entityShortcuts = computed(() => chatStore.entityShortcuts)
const userPreferences = computed(() => chatStore.userPreferences)
const learningAnalytics = computed(() => chatStore.learningAnalytics)
const successRate = computed(() => chatStore.successRate)

// Connection status
const connectionStatusClass = computed(() => {
  return isConnected.value ? 'connected' : 'disconnected'
})

const connectionStatusText = computed(() => {
  return isConnected.value ? 'Connected' : 'Disconnected'
})

// Methods
const formatSessionId = (sessionId) => {
  if (!sessionId) return 'No session'
  return sessionId.substring(0, 8) + '...'
}

const formatHistoryTime = (timestamp) => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now - date
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffDays = Math.floor(diffHours / 24)
  
  if (diffDays > 0) {
    return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`
  } else if (diffHours > 0) {
    return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`
  } else {
    return 'Recently'
  }
}

const reconnect = async () => {
  try {
    await chatStore.checkHealth()
    if (!isConnected.value) {
      // Try to reinitialize the session
      const userId = 'user-' + Date.now() // In real app, get from auth
      await chatStore.initializeSession(currentSessionId.value, userId)
    }
  } catch (error) {
    console.error('Reconnection failed:', error)
  }
}

const createNewSession = async () => {
  try {
    const userId = 'user-' + Date.now() // In real app, get from auth
    await chatStore.createNewSession(userId)
  } catch (error) {
    console.error('Failed to create new session:', error)
  }
}

const loadConversationHistory = async () => {
  try {
    const userId = 'user-' + Date.now() // In real app, get from auth
    const history = await chatStore.loadConversationHistory(userId, 20)
    conversationHistory.value = history
  } catch (error) {
    console.error('Failed to load conversation history:', error)
  }
}

const switchToSession = async (sessionId) => {
  try {
    const userId = 'user-' + Date.now() // In real app, get from auth
    await chatStore.switchSession(sessionId, userId)
    showSessionHistory.value = false
  } catch (error) {
    console.error('Failed to switch session:', error)
  }
}

const addShortcut = async () => {
  if (!newShortcut.value || !newFullForm.value) return
  
  try {
    const userId = 'user-' + Date.now() // In real app, get from auth
    await chatStore.addEntityShortcut(userId, newShortcut.value.toLowerCase(), newFullForm.value)
    newShortcut.value = ''
    newFullForm.value = ''
  } catch (error) {
    console.error('Failed to add shortcut:', error)
  }
}

const removeShortcut = async (shortcut) => {
  try {
    const userId = 'user-' + Date.now() // In real app, get from auth
    await chatStore.removeEntityShortcut(userId, shortcut)
  } catch (error) {
    console.error('Failed to remove shortcut:', error)
  }
}

const savePreferences = async () => {
  try {
    const userId = 'user-' + Date.now() // In real app, get from auth
    await chatStore.updateUserPreferences(userId, userPreferences.value)
  } catch (error) {
    console.error('Failed to save preferences:', error)
  }
}

const clearError = () => {
  chatStore.clearError()
}

// Lifecycle
onMounted(async () => {
  // Initialize chat session
  const userId = 'user-' + Date.now() // In real app, get from auth
  await chatStore.initializeSession('', userId)
  
  // Load conversation history
  await loadConversationHistory()
})

// Watch for session history dialog opening
const handleSessionHistoryOpen = () => {
  if (showSessionHistory.value) {
    loadConversationHistory()
  }
}
</script>

<style scoped>
.chat-view {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: #f8fafc;
}

.page-header {
  background: white;
  border-bottom: 1px solid #e2e8f0;
  padding: 1.5rem;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.page-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  font-size: 1.75rem;
  color: #1f2937;
}

.page-description {
  color: #6b7280;
  margin: 0.5rem 0 0 0;
}

.connection-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.status-dot.connected {
  background: #10b981;
}

.status-dot.disconnected {
  background: #ef4444;
  animation: pulse 2s infinite;
}

.status-text {
  font-size: 0.875rem;
  color: #6b7280;
}

.reconnect-btn {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  background: #ef4444;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.75rem;
}

.session-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.current-session-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: #6b7280;
}

.session-id {
  font-family: monospace;
  background: #f3f4f6;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.session-actions {
  display: flex;
  gap: 0.5rem;
}

.session-btn {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.5rem 0.75rem;
  background: white;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.875rem;
  color: #374151;
  transition: all 0.2s;
}

.session-btn:hover {
  background: #f9fafb;
  border-color: #2563eb;
  color: #2563eb;
}

.chat-container {
  flex: 1;
  padding: 1rem;
  overflow: hidden;
}

.session-history {
  max-height: 400px;
  overflow-y: auto;
}

.no-history {
  text-align: center;
  padding: 2rem;
  color: #6b7280;
}

.history-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid #f1f5f9;
  cursor: pointer;
  transition: background-color 0.2s;
}

.history-item:hover {
  background: #f8fafc;
}

.history-content {
  flex: 1;
}

.history-query {
  font-weight: 500;
  color: #1f2937;
  margin-bottom: 0.25rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.history-meta {
  display: flex;
  gap: 1rem;
  font-size: 0.75rem;
  color: #6b7280;
}

.settings-content {
  padding: 1rem 0;
}

.settings-section {
  margin-bottom: 2rem;
  padding-bottom: 2rem;
  border-bottom: 1px solid #f1f5f9;
}

.settings-section:last-child {
  border-bottom: none;
}

.settings-section h3 {
  margin: 0 0 0.5rem 0;
  color: #1f2937;
}

.section-description {
  color: #6b7280;
  font-size: 0.875rem;
  margin-bottom: 1rem;
}

.entity-shortcuts {
  margin-bottom: 1rem;
}

.shortcut-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem;
  background: #f8fafc;
  border-radius: 6px;
  margin-bottom: 0.5rem;
}

.shortcut {
  font-family: monospace;
  font-weight: 600;
  color: #2563eb;
}

.arrow {
  color: #6b7280;
}

.full-form {
  flex: 1;
  color: #374151;
}

.remove-btn {
  padding: 0.25rem;
  background: none;
  border: none;
  color: #ef4444;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.remove-btn:hover {
  background: #fee2e2;
}

.add-shortcut {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.shortcut-input {
  padding: 0.5rem;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 0.875rem;
}

.add-btn {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.5rem 0.75rem;
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.875rem;
}

.add-btn:disabled {
  background: #9ca3af;
  cursor: not-allowed;
}

.preference-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.preference-item label {
  color: #374151;
  font-weight: 500;
}

.preference-item select {
  padding: 0.5rem;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 0.875rem;
}

.analytics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 1rem;
}

.analytic-item {
  text-align: center;
  padding: 1rem;
  background: #f8fafc;
  border-radius: 8px;
}

.analytic-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: #2563eb;
  margin-bottom: 0.25rem;
}

.analytic-label {
  font-size: 0.75rem;
  color: #6b7280;
  text-transform: uppercase;
  font-weight: 500;
}

.error-banner {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  background: #fee2e2;
  border-bottom: 1px solid #fecaca;
  color: #991b1b;
  padding: 0.75rem 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  z-index: 1000;
}

.close-error {
  margin-left: auto;
  background: none;
  border: none;
  color: #991b1b;
  cursor: pointer;
  padding: 0.25rem;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* Responsive Design */
@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }

  .session-controls {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .session-actions {
    flex-wrap: wrap;
  }

  .add-shortcut {
    flex-direction: column;
    align-items: stretch;
  }

  .preference-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }

  .analytics-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>