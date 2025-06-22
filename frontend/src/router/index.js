import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    name: 'Dashboard',
    component: () => import('../views/Dashboard.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  
  console.log('Router guard:', {
    to: to.path,
    from: from.path,
    isLoggedIn: authStore.isLoggedIn,
    token: !!authStore.token,
    requiresAuth: to.meta.requiresAuth
  })
  
  // Initialize auth if token exists
  if (authStore.token && !authStore.isAuthenticated) {
    await authStore.initializeAuth()
  }
  
  if (to.meta.requiresAuth && !authStore.isLoggedIn) {
    console.log('Redirecting to login - not authenticated')
    next('/login')
  } else if (to.path === '/login' && authStore.isLoggedIn) {
    console.log('Redirecting to dashboard - already authenticated')
    next('/')
  } else {
    console.log('Allowing navigation to:', to.path)
    next()
  }
})

export default router