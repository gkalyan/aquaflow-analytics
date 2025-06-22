import { defineStore } from 'pinia'
import api from '../services/api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null,
    token: localStorage.getItem('auth_token') || null,
    isAuthenticated: false,
    loading: false,
    error: null
  }),

  getters: {
    currentUser: (state) => state.user,
    isLoggedIn: (state) => state.isAuthenticated && state.token !== null,
    authError: (state) => state.error
  },

  actions: {
    async login(username, password) {
      this.loading = true
      this.error = null
      
      try {
        const response = await api.post('/auth/login', {
          username,
          password
        })
        
        const { token, user } = response.data
        
        this.token = token
        this.user = user
        this.isAuthenticated = true
        
        // Store token in localStorage
        localStorage.setItem('auth_token', token)
        
        // Token is automatically set by api interceptor
        
        return { success: true }
      } catch (error) {
        this.error = error.response?.data?.error || 'Login failed'
        return { success: false, error: this.error }
      } finally {
        this.loading = false
      }
    },

    async logout() {
      try {
        await api.post('/auth/logout')
      } catch (error) {
        console.error('Logout error:', error)
      } finally {
        this.user = null
        this.token = null
        this.isAuthenticated = false
        localStorage.removeItem('auth_token')
        // Token removal handled by clearing localStorage
      }
    },

    async fetchCurrentUser() {
      if (!this.token) return
      
      try {
        const response = await api.get('/me')
        this.user = response.data
        this.isAuthenticated = true
      } catch (error) {
        console.error('Failed to fetch user:', error)
        this.logout()
      }
    },

    async initializeAuth() {
      if (this.token) {
        await this.fetchCurrentUser()
      }
    }
  }
})