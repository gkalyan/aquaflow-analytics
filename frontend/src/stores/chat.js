import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import chatApi from '@/services/chatApi'

export const useChatStore = defineStore('chat', () => {
  // State
  const currentSession = ref(null)
  const sessions = ref(new Map())
  const userPreferences = ref({
    entity_shortcuts: {},
    detail_level: 'normal',
    preferred_units: 'imperial'
  })
  const isConnected = ref(false)
  const lastError = ref(null)
  const learningAnalytics = ref({
    total_queries: 0,
    successful_queries: 0,
    clarification_requests: 0,
    feedback_submissions: 0,
    learned_entities: {}
  })

  // Getters
  const currentSessionId = computed(() => currentSession.value?.session_id)
  const currentMessages = computed(() => currentSession.value?.messages || [])
  const entityShortcuts = computed(() => userPreferences.value.entity_shortcuts || {})
  const successRate = computed(() => {
    const total = learningAnalytics.value.total_queries
    const successful = learningAnalytics.value.successful_queries
    return total > 0 ? (successful / total * 100).toFixed(1) : 0
  })

  // Actions
  const initializeSession = async (sessionId, userId) => {
    try {
      let session
      
      if (sessionId) {
        // Try to load existing session
        try {
          session = await chatApi.getSession(sessionId, userId)
        } catch (error) {
          console.warn('Session not found, creating new one')
          session = await chatApi.createSession(userId)
        }
      } else {
        // Create new session
        session = await chatApi.createSession(userId)
      }

      currentSession.value = session
      sessions.value.set(session.session_id, session)
      
      // Load user preferences
      await loadUserPreferences(userId)
      
      // Load learning analytics
      await loadLearningAnalytics(userId)
      
      isConnected.value = true
      lastError.value = null
      
      return session
    } catch (error) {
      console.error('Error initializing session:', error)
      lastError.value = error.message
      isConnected.value = false
      throw error
    }
  }

  const sendMessage = async (query, userId) => {
    if (!currentSession.value) {
      throw new Error('No active session')
    }

    try {
      const response = await chatApi.sendMessage({
        query,
        session_id: currentSession.value.session_id,
        user_id: userId
      })

      // Update session with new messages
      updateSessionMessages(response)
      
      // Update analytics
      learningAnalytics.value.total_queries++
      if (response.confidence > 0.7) {
        learningAnalytics.value.successful_queries++
      }
      if (response.needs_clarification) {
        learningAnalytics.value.clarification_requests++
      }

      return response
    } catch (error) {
      console.error('Error sending message:', error)
      lastError.value = error.message
      throw error
    }
  }

  const handleClarification = async (clarificationRequest) => {
    if (!currentSession.value) {
      throw new Error('No active session')
    }

    try {
      const response = await chatApi.handleClarification(clarificationRequest)
      updateSessionMessages(response)
      return response
    } catch (error) {
      console.error('Error handling clarification:', error)
      lastError.value = error.message
      throw error
    }
  }

  const submitFeedback = async (feedback) => {
    try {
      await chatApi.submitFeedback(feedback)
      learningAnalytics.value.feedback_submissions++
      
      // Update learned entities if provided
      if (feedback.entity_mappings) {
        Object.assign(learningAnalytics.value.learned_entities, feedback.entity_mappings)
      }
    } catch (error) {
      console.error('Error submitting feedback:', error)
      lastError.value = error.message
      throw error
    }
  }

  const loadConversationHistory = async (userId, limit = 10) => {
    try {
      const history = await chatApi.getConversationHistory(userId, limit)
      return history.conversations || []
    } catch (error) {
      console.error('Error loading conversation history:', error)
      lastError.value = error.message
      return []
    }
  }

  const loadUserPreferences = async (userId) => {
    try {
      const preferences = await chatApi.getUserPreferences(userId)
      userPreferences.value = {
        ...userPreferences.value,
        ...preferences
      }
    } catch (error) {
      console.warn('Could not load user preferences:', error)
      // Use defaults if preferences can't be loaded
    }
  }

  const updateUserPreferences = async (userId, newPreferences) => {
    try {
      const updated = await chatApi.updateUserPreferences(userId, newPreferences)
      userPreferences.value = {
        ...userPreferences.value,
        ...updated
      }
      return updated
    } catch (error) {
      console.error('Error updating user preferences:', error)
      lastError.value = error.message
      throw error
    }
  }

  const loadLearningAnalytics = async (userId) => {
    try {
      const analytics = await chatApi.getLearningAnalytics(userId)
      learningAnalytics.value = {
        ...learningAnalytics.value,
        ...analytics
      }
    } catch (error) {
      console.warn('Could not load learning analytics:', error)
      // Use defaults if analytics can't be loaded
    }
  }

  const addEntityShortcut = async (userId, shortcut, fullForm) => {
    const newShortcuts = {
      ...userPreferences.value.entity_shortcuts,
      [shortcut]: fullForm
    }
    
    await updateUserPreferences(userId, {
      entity_shortcuts: newShortcuts
    })
    
    // Also update learning analytics
    learningAnalytics.value.learned_entities[shortcut] = fullForm
  }

  const removeEntityShortcut = async (userId, shortcut) => {
    const newShortcuts = { ...userPreferences.value.entity_shortcuts }
    delete newShortcuts[shortcut]
    
    await updateUserPreferences(userId, {
      entity_shortcuts: newShortcuts
    })
  }

  const switchSession = async (sessionId, userId) => {
    if (sessions.value.has(sessionId)) {
      currentSession.value = sessions.value.get(sessionId)
    } else {
      await initializeSession(sessionId, userId)
    }
  }

  const createNewSession = async (userId) => {
    const session = await chatApi.createSession(userId)
    currentSession.value = session
    sessions.value.set(session.session_id, session)
    return session
  }

  const checkHealth = async () => {
    try {
      await chatApi.healthCheck()
      isConnected.value = true
      return true
    } catch (error) {
      isConnected.value = false
      lastError.value = error.message
      return false
    }
  }

  const getAvailableCommands = async () => {
    try {
      return await chatApi.getAvailableCommands()
    } catch (error) {
      console.error('Error getting available commands:', error)
      return { commands: [], tips: [] }
    }
  }

  const clearError = () => {
    lastError.value = null
  }

  const clearSession = () => {
    currentSession.value = null
  }

  const clearAllSessions = () => {
    sessions.value.clear()
    currentSession.value = null
  }

  // Helper functions
  const updateSessionMessages = (response) => {
    if (!currentSession.value) return

    // Update the current session with new message data
    // This would typically involve adding the assistant's response
    // to the session's message history
    
    currentSession.value.last_activity = new Date()
    
    // Update entity mappings if provided
    if (response.entities) {
      currentSession.value.entity_mappings = {
        ...currentSession.value.entity_mappings,
        ...response.entities
      }
    }
  }

  const getSessionSummary = (sessionId) => {
    const session = sessions.value.get(sessionId)
    if (!session) return null

    return {
      session_id: sessionId,
      message_count: session.messages?.length || 0,
      last_activity: session.last_activity,
      last_query: session.messages?.filter(m => m.type === 'user').slice(-1)[0]?.content || '',
      entity_count: Object.keys(session.entity_mappings || {}).length
    }
  }

  const getRecentSessions = (limit = 5) => {
    return Array.from(sessions.value.values())
      .sort((a, b) => new Date(b.last_activity) - new Date(a.last_activity))
      .slice(0, limit)
      .map(session => getSessionSummary(session.session_id))
  }

  return {
    // State
    currentSession,
    sessions,
    userPreferences,
    isConnected,
    lastError,
    learningAnalytics,

    // Getters
    currentSessionId,
    currentMessages,
    entityShortcuts,
    successRate,

    // Actions
    initializeSession,
    sendMessage,
    handleClarification,
    submitFeedback,
    loadConversationHistory,
    loadUserPreferences,
    updateUserPreferences,
    loadLearningAnalytics,
    addEntityShortcut,
    removeEntityShortcut,
    switchSession,
    createNewSession,
    checkHealth,
    getAvailableCommands,
    clearError,
    clearSession,
    clearAllSessions,
    getSessionSummary,
    getRecentSessions
  }
})