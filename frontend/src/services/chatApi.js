import api from './api'

const chatApi = {
  /**
   * Send a message to the chat system
   * @param {Object} message - The message object
   * @param {string} message.query - The user's query
   * @param {string} message.session_id - The session ID
   * @param {string} message.user_id - The user ID
   * @param {Object} [message.context] - Additional context
   * @returns {Promise<Object>} The response from the server
   */
  async sendMessage(message) {
    try {
      // Transform message format to match backend expectations
      const payload = {
        session_id: message.session_id,
        message: message.query || message.message,
        user_id: message.user_id
      }
      const response = await api.post('/chat', payload)
      return response.data
    } catch (error) {
      console.error('Error sending message:', error)
      throw error
    }
  },

  /**
   * Handle clarification response
   * @param {Object} clarification - The clarification object
   * @param {string} clarification.session_id - The session ID
   * @param {string} clarification.message_id - The original message ID
   * @param {string} clarification.clarification - The clarification text
   * @param {string} clarification.user_choice - The user's choice
   * @returns {Promise<Object>} The response from the server
   */
  async handleClarification(clarification) {
    try {
      // For now, send as regular message since backend doesn't have separate clarification endpoint
      const payload = {
        session_id: clarification.session_id,
        message: clarification.user_choice || clarification.clarification,
        user_id: clarification.user_id
      }
      const response = await api.post('/chat', payload)
      return response.data
    } catch (error) {
      console.error('Error handling clarification:', error)
      throw error
    }
  },

  /**
   * Submit feedback for a chat response
   * @param {Object} feedback - The feedback object
   * @param {string} feedback.session_id - The session ID
   * @param {string} feedback.message_id - The message ID
   * @param {string} feedback.original_query - The original query
   * @param {boolean} feedback.helpful - Whether the response was helpful
   * @param {string} [feedback.comments] - Additional comments
   * @param {string} [feedback.correct_sql] - The correct SQL if applicable
   * @param {Object} [feedback.entity_mappings] - Correct entity mappings
   * @returns {Promise<Object>} The response from the server
   */
  async submitFeedback(feedback) {
    try {
      const response = await api.post('/chat/feedback', feedback)
      return response.data
    } catch (error) {
      console.error('Error submitting feedback:', error)
      throw error
    }
  },

  /**
   * Get conversation history for a user
   * @param {string} userId - The user ID
   * @param {number} [limit=10] - Maximum number of conversations to return
   * @returns {Promise<Object>} The conversation history
   */
  async getConversationHistory(userId, limit = 10) {
    try {
      const response = await api.get(`/chat/history/${userId}?limit=${limit}`)
      return response.data
    } catch (error) {
      console.error('Error getting conversation history:', error)
      throw error
    }
  },

  /**
   * Get a specific chat session
   * @param {string} sessionId - The session ID
   * @param {string} userId - The user ID
   * @returns {Promise<Object>} The session data
   */
  async getSession(sessionId, userId) {
    try {
      const response = await api.get(`/chat/sessions/${sessionId}`)
      return response.data
    } catch (error) {
      console.error('Error getting session:', error)
      throw error
    }
  },

  /**
   * Create a new chat session
   * @param {string} userId - The user ID
   * @returns {Promise<Object>} The new session data
   */
  async createSession(userId) {
    try {
      const response = await api.post('/chat/session', { user_id: userId })
      return response.data
    } catch (error) {
      console.error('Error creating session:', error)
      throw error
    }
  },

  /**
   * Check the health of the chat service
   * @returns {Promise<Object>} The health status
   */
  async healthCheck() {
    try {
      const response = await api.get('/chat/health')
      return response.data
    } catch (error) {
      console.error('Error checking chat health:', error)
      throw error
    }
  },

  /**
   * Get available chat commands and examples
   * @returns {Promise<Object>} The available commands
   */
  async getAvailableCommands() {
    try {
      const response = await api.get('/chat/commands')
      return response.data
    } catch (error) {
      console.error('Error getting available commands:', error)
      throw error
    }
  },

  /**
   * Get user preferences
   * @param {string} userId - The user ID
   * @returns {Promise<Object>} The user preferences
   */
  async getUserPreferences(userId) {
    try {
      const response = await api.get(`/chat/preferences/${userId}`)
      return response.data
    } catch (error) {
      console.error('Error getting user preferences:', error)
      throw error
    }
  },

  /**
   * Update user preferences
   * @param {string} userId - The user ID
   * @param {Object} preferences - The preferences to update
   * @returns {Promise<Object>} The updated preferences
   */
  async updateUserPreferences(userId, preferences) {
    try {
      const response = await api.put(`/chat/preferences/${userId}`, preferences)
      return response.data
    } catch (error) {
      console.error('Error updating user preferences:', error)
      throw error
    }
  },

  /**
   * Get learning analytics for a user
   * @param {string} userId - The user ID
   * @returns {Promise<Object>} Learning analytics data
   */
  async getLearningAnalytics(userId) {
    try {
      const response = await api.get(`/chat/analytics/${userId}`)
      return response.data
    } catch (error) {
      console.error('Error getting learning analytics:', error)
      throw error
    }
  }
}

export default chatApi