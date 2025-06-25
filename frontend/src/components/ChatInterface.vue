<template>
  <div class="chat-interface">
    <!-- Chat Header -->
    <div class="chat-header">
      <h2 class="chat-title">
        <i class="pi pi-comments"></i>
        AquaFlow Assistant
      </h2>
      <div class="chat-status">
        <span :class="['status-indicator', connectionStatus]"></span>
        {{ connectionStatus === 'connected' ? 'Online' : 'Connecting...' }}
      </div>
    </div>

    <!-- Chat Messages -->
    <div class="chat-messages" ref="messagesContainer">
      <!-- Welcome Message -->
      <div v-if="messages.length === 0" class="welcome-message">
        <div class="welcome-content">
          <i class="pi pi-info-circle"></i>
          <h3>Welcome to AquaFlow Assistant</h3>
          <p>Ask me about your water systems using natural language:</p>
          <ul>
            <li>"What's the flow rate at Main Canal?"</li>
            <li>"Show me pump station 1 status"</li>
            <li>"Why is pressure low at PS1?"</li>
          </ul>
          <p class="tip">💡 You can use abbreviations like MC (Main Canal), PS1 (Pump Station 1)</p>
        </div>
      </div>

      <!-- Message List -->
      <div
        v-for="message in messages"
        :key="message.id"
        :class="['message', message.type]"
      >
        <div class="message-content">
          <!-- User Message -->
          <div v-if="message.type === 'user'" class="user-message">
            <div class="message-text">{{ message.content }}</div>
            <div class="message-time">{{ formatTime(message.timestamp) }}</div>
          </div>

          <!-- Assistant Message -->
          <div v-else-if="message.type === 'assistant'" class="assistant-message">
            <div class="message-text" v-html="formatAssistantMessage(message.content)"></div>
            
            <!-- Data Visualization -->
            <div v-if="message.data" class="message-data">
              <DataCard :data="message.data" />
            </div>

            <!-- Message Actions -->
            <div class="message-actions">
              <button
                @click="copyToClipboard(message.content)"
                class="action-btn"
                title="Copy response"
              >
                <i class="pi pi-copy"></i>
              </button>
              <button
                @click="provideFeedback(message, true)"
                class="action-btn helpful"
                title="This was helpful"
              >
                <i class="pi pi-thumbs-up"></i>
              </button>
              <button
                @click="provideFeedback(message, false)"
                class="action-btn not-helpful"
                title="This wasn't helpful"
              >
                <i class="pi pi-thumbs-down"></i>
              </button>
            </div>
            <div class="message-time">{{ formatTime(message.timestamp) }}</div>
          </div>

          <!-- Clarification Request -->
          <div v-else-if="message.type === 'clarification'" class="clarification-message">
            <div class="message-text">{{ message.content }}</div>
            <div v-if="message.clarification_questions" class="clarification-options">
              <button
                v-for="(question, index) in message.clarification_questions"
                :key="index"
                @click="selectClarification(message, question)"
                class="clarification-btn"
              >
                {{ question }}
              </button>
            </div>
          </div>

          <!-- System Message -->
          <div v-else-if="message.type === 'system'" class="system-message">
            <i class="pi pi-info-circle"></i>
            <span>{{ message.content }}</span>
          </div>
        </div>
      </div>

      <!-- Typing Indicator -->
      <div v-if="isTyping" class="typing-indicator">
        <div class="typing-dots">
          <span></span>
          <span></span>
          <span></span>
        </div>
        <span class="typing-text">AquaFlow is thinking...</span>
      </div>
    </div>

    <!-- Chat Input -->
    <div class="chat-input">
      <div class="input-container">
        <textarea
          v-model="currentMessage"
          @keydown="handleKeyDown"
          @input="adjustTextareaHeight"
          ref="messageInput"
          placeholder="Ask about your water systems... (e.g., 'MC flow rate tell me')"
          rows="1"
          :disabled="isTyping"
        ></textarea>
        <button
          @click="sendMessage"
          :disabled="!currentMessage.trim() || isTyping"
          class="send-button"
        >
          <i v-if="!isTyping" class="pi pi-send"></i>
          <i v-else class="pi pi-spinner pi-spin"></i>
        </button>
      </div>
      
      <!-- Quick Actions -->
      <div class="quick-actions">
        <button
          v-for="action in quickActions"
          :key="action.id"
          @click="useQuickAction(action)"
          class="quick-action-btn"
        >
          {{ action.label }}
        </button>
      </div>
    </div>

    <!-- Feedback Dialog -->
    <Dialog
      v-model:visible="showFeedbackDialog"
      header="Provide Feedback"
      :modal="true"
      :closable="true"
      style="width: 400px"
    >
      <div class="feedback-form">
        <p>Help us improve by providing feedback on this response:</p>
        
        <div class="field">
          <label for="feedback-comments">Comments (optional):</label>
          <textarea
            id="feedback-comments"
            v-model="feedbackComments"
            rows="3"
            placeholder="What could be improved?"
          ></textarea>
        </div>

        <div v-if="!feedbackHelpful" class="field">
          <label for="correct-answer">What should the correct answer be?</label>
          <textarea
            id="correct-answer"
            v-model="correctAnswer"
            rows="3"
            placeholder="Provide the correct information..."
          ></textarea>
        </div>
      </div>

      <template #footer>
        <div class="feedback-actions">
          <Button
            label="Cancel"
            severity="secondary"
            @click="closeFeedbackDialog"
          />
          <Button
            label="Submit Feedback"
            @click="submitFeedback"
            :loading="submittingFeedback"
          />
        </div>
      </template>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick, computed } from 'vue'
import { v4 as uuidv4 } from 'uuid'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import DataCard from './DataCard.vue'
import { useChatStore } from '@/stores/chat'
import chatApi from '@/services/chatApi'

// Reactive state
const messages = ref([])
const currentMessage = ref('')
const isTyping = ref(false)
const connectionStatus = ref('connected')
const messagesContainer = ref(null)
const messageInput = ref(null)

// Feedback state
const showFeedbackDialog = ref(false)
const feedbackMessage = ref(null)
const feedbackHelpful = ref(true)
const feedbackComments = ref('')
const correctAnswer = ref('')
const submittingFeedback = ref(false)

// Chat session
const sessionId = ref(uuidv4())
const userId = ref('user-' + Date.now()) // In real app, get from auth

// Quick actions
const quickActions = ref([
  { id: 1, label: 'Morning Check', query: 'Show me morning system status' },
  { id: 2, label: 'MC Flow Rate', query: 'What is Main Canal flow rate?' },
  { id: 3, label: 'PS1 Status', query: 'Show pump station 1 status' },
  { id: 4, label: 'Pressure Check', query: 'Check all pressure levels' },
])

// Store
const chatStore = useChatStore()

// Computed
const formattedConnectionStatus = computed(() => {
  return connectionStatus.value.charAt(0).toUpperCase() + connectionStatus.value.slice(1)
})

// Methods
const sendMessage = async () => {
  if (!currentMessage.value.trim() || isTyping.value) return

  const userMessage = {
    id: uuidv4(),
    type: 'user',
    content: currentMessage.value.trim(),
    timestamp: new Date(),
  }

  messages.value.push(userMessage)
  const queryText = currentMessage.value.trim()
  currentMessage.value = ''

  // Reset textarea height
  adjustTextareaHeight()

  // Scroll to bottom
  await nextTick()
  scrollToBottom()

  // Set typing indicator
  isTyping.value = true

  try {
    // Send query to backend
    const response = await chatApi.sendMessage({
      query: queryText,
      session_id: sessionId.value,
      user_id: userId.value,
    })

    // Handle clarification request
    if (response.needs_clarification) {
      const clarificationMessage = {
        id: uuidv4(),
        type: 'clarification',
        content: response.response,
        clarification_questions: response.clarification_question ? [response.clarification_question] : [],
        timestamp: new Date(),
      }
      messages.value.push(clarificationMessage)
    } else {
      // Regular response
      const assistantMessage = {
        id: uuidv4(),
        type: 'assistant',
        content: response.response,
        data: response.data,
        metadata: {
          entity_mappings: response.entity_mappings,
          confidence: response.confidence,
        },
        timestamp: new Date(),
      }
      messages.value.push(assistantMessage)
    }

  } catch (error) {
    console.error('Error sending message:', error)
    
    const errorMessage = {
      id: uuidv4(),
      type: 'system',
      content: 'Sorry, I encountered an error processing your request. Please try again.',
      timestamp: new Date(),
    }
    messages.value.push(errorMessage)
  } finally {
    isTyping.value = false
    await nextTick()
    scrollToBottom()
  }
}

const selectClarification = async (clarificationMessage, selectedOption) => {
  try {
    const response = await chatApi.handleClarification({
      session_id: sessionId.value,
      message_id: clarificationMessage.id,
      clarification: selectedOption,
      user_choice: selectedOption,
    })

    // Add user's clarification selection as a message
    const userClarification = {
      id: uuidv4(),
      type: 'user',
      content: selectedOption,
      timestamp: new Date(),
    }
    messages.value.push(userClarification)

    // Add assistant's response
    const assistantMessage = {
      id: uuidv4(),
      type: 'assistant',
      content: response.response,
      data: response.data,
      metadata: {
        entity_mappings: response.entity_mappings,
        confidence: response.confidence,
      },
      timestamp: new Date(),
    }
    messages.value.push(assistantMessage)

    await nextTick()
    scrollToBottom()

  } catch (error) {
    console.error('Error handling clarification:', error)
  }
}

const useQuickAction = (action) => {
  currentMessage.value = action.query
  sendMessage()
}

const handleKeyDown = (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}

const adjustTextareaHeight = () => {
  const textarea = messageInput.value
  if (textarea) {
    textarea.style.height = 'auto'
    textarea.style.height = Math.min(textarea.scrollHeight, 120) + 'px'
  }
}

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const formatTime = (timestamp) => {
  return new Date(timestamp).toLocaleTimeString([], { 
    hour: '2-digit', 
    minute: '2-digit' 
  })
}

const formatAssistantMessage = (content) => {
  // Simple formatting for now - could be enhanced with markdown support
  return content.replace(/\n/g, '<br>')
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    // Could show a toast notification here
  } catch (error) {
    console.error('Failed to copy to clipboard:', error)
  }
}

const provideFeedback = (message, helpful) => {
  feedbackMessage.value = message
  feedbackHelpful.value = helpful
  feedbackComments.value = ''
  correctAnswer.value = ''
  showFeedbackDialog.value = true
}

const submitFeedback = async () => {
  if (!feedbackMessage.value) return

  submittingFeedback.value = true

  try {
    await chatApi.submitFeedback({
      session_id: sessionId.value,
      message_id: feedbackMessage.value.id,
      original_query: '', // Would need to track the original query
      helpful: feedbackHelpful.value,
      comments: feedbackComments.value,
      correct_sql: correctAnswer.value,
    })

    closeFeedbackDialog()
    
    // Add system message acknowledging feedback
    const systemMessage = {
      id: uuidv4(),
      type: 'system',
      content: 'Thank you for your feedback! This helps me learn and improve.',
      timestamp: new Date(),
    }
    messages.value.push(systemMessage)

  } catch (error) {
    console.error('Error submitting feedback:', error)
  } finally {
    submittingFeedback.value = false
  }
}

const closeFeedbackDialog = () => {
  showFeedbackDialog.value = false
  feedbackMessage.value = null
  feedbackComments.value = ''
  correctAnswer.value = ''
}

// Lifecycle
onMounted(() => {
  // Focus on input
  if (messageInput.value) {
    messageInput.value.focus()
  }

  // Load conversation history if needed
  chatStore.initializeSession(sessionId.value, userId.value)
})
</script>

<style scoped>
.chat-interface {
  display: flex;
  flex-direction: column;
  height: 100vh;
  max-height: 800px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  color: white;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.chat-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.chat-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  opacity: 0.9;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #10b981;
}

.status-indicator.connecting {
  background: #f59e0b;
  animation: pulse 2s infinite;
}

.chat-messages {
  flex: 1;
  padding: 1rem;
  overflow-y: auto;
  scroll-behavior: smooth;
}

.welcome-message {
  text-align: center;
  padding: 2rem;
  color: #6b7280;
}

.welcome-content h3 {
  color: #1f2937;
  margin-bottom: 1rem;
}

.welcome-content ul {
  text-align: left;
  max-width: 400px;
  margin: 1rem auto;
}

.welcome-content .tip {
  background: #f3f4f6;
  padding: 0.75rem;
  border-radius: 8px;
  margin-top: 1rem;
  font-size: 0.875rem;
}

.message {
  margin-bottom: 1rem;
}

.message-content {
  max-width: 80%;
}

.user-message {
  margin-left: auto;
  background: #2563eb;
  color: white;
  padding: 0.75rem 1rem;
  border-radius: 18px 18px 4px 18px;
  text-align: right;
}

.assistant-message {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  padding: 1rem;
  border-radius: 18px 18px 18px 4px;
  position: relative;
}

.clarification-message {
  background: #fef3c7;
  border: 1px solid #f59e0b;
  padding: 1rem;
  border-radius: 8px;
}

.clarification-options {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 1rem;
}

.clarification-btn {
  padding: 0.5rem 1rem;
  background: white;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.clarification-btn:hover {
  background: #f3f4f6;
  border-color: #2563eb;
}

.system-message {
  background: #f0f9ff;
  border: 1px solid #0ea5e9;
  padding: 0.75rem;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #0369a1;
  font-size: 0.875rem;
}

.message-data {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #e2e8f0;
}

.message-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 1rem;
  padding-top: 0.5rem;
  border-top: 1px solid #e2e8f0;
}

.action-btn {
  padding: 0.25rem 0.5rem;
  background: none;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  cursor: pointer;
  color: #6b7280;
  transition: all 0.2s;
  font-size: 0.75rem;
}

.action-btn:hover {
  background: #f3f4f6;
}

.action-btn.helpful:hover {
  color: #10b981;
  border-color: #10b981;
}

.action-btn.not-helpful:hover {
  color: #ef4444;
  border-color: #ef4444;
}

.message-time {
  font-size: 0.75rem;
  color: #9ca3af;
  margin-top: 0.25rem;
}

.typing-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #6b7280;
  font-size: 0.875rem;
}

.typing-dots {
  display: flex;
  gap: 0.25rem;
}

.typing-dots span {
  width: 6px;
  height: 6px;
  background: #6b7280;
  border-radius: 50%;
  animation: typing 1.4s infinite;
}

.typing-dots span:nth-child(2) {
  animation-delay: 0.2s;
}

.typing-dots span:nth-child(3) {
  animation-delay: 0.4s;
}

.chat-input {
  padding: 1rem;
  border-top: 1px solid #e2e8f0;
  background: #f8fafc;
}

.input-container {
  display: flex;
  gap: 0.75rem;
  align-items: flex-end;
}

.input-container textarea {
  flex: 1;
  padding: 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  resize: none;
  font-family: inherit;
  font-size: 0.875rem;
  line-height: 1.4;
  transition: border-color 0.2s;
}

.input-container textarea:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
}

.send-button {
  padding: 0.75rem;
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.send-button:hover:not(:disabled) {
  background: #1d4ed8;
}

.send-button:disabled {
  background: #9ca3af;
  cursor: not-allowed;
}

.quick-actions {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  flex-wrap: wrap;
}

.quick-action-btn {
  padding: 0.5rem 0.75rem;
  background: white;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.75rem;
  color: #374151;
  transition: all 0.2s;
}

.quick-action-btn:hover {
  background: #f3f4f6;
  border-color: #2563eb;
  color: #2563eb;
}

.feedback-form {
  padding: 1rem 0;
}

.field {
  margin-bottom: 1rem;
}

.field label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #374151;
}

.field textarea {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-family: inherit;
  resize: vertical;
}

.feedback-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
}

@keyframes typing {
  0%, 60%, 100% {
    transform: translateY(0);
  }
  30% {
    transform: translateY(-10px);
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>