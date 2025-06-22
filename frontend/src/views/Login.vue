<template>
  <div class="surface-0 flex align-items-center justify-content-center min-h-screen min-w-screen overflow-hidden">
    <div class="grid justify-content-center p-2 lg:p-0" style="min-width:80%">
      <div class="col-12 xl:col-6" style="border-radius:56px; padding:0.3rem; background: linear-gradient(180deg, var(--primary-color) 10%, rgba(33, 150, 243, 0) 30%);">
        <div class="h-full w-full m-0 py-7 px-4" style="border-radius:53px; background: linear-gradient(180deg, var(--surface-50) 38.9%, var(--surface-0));">
          <div class="text-center mb-5">
            <div class="text-900 text-3xl font-medium mb-3">Welcome to AquaFlow</div>
            <span class="text-600 font-medium">Daily Operations Assistant for Water Districts</span>
          </div>

          <div class="w-full md:w-10 mx-auto">
            <label for="username1" class="block text-900 text-xl font-medium mb-2">Username</label>
            <InputText 
              id="username1" 
              v-model="username" 
              type="text" 
              placeholder="Username" 
              class="w-full md:w-30rem mb-5" 
              style="padding:1rem"
              :class="{ 'p-invalid': submitted && !username }"
            />

            <label for="password1" class="block text-900 font-medium text-xl mb-2">Password</label>
            <Password 
              id="password1" 
              v-model="password" 
              placeholder="Password" 
              :toggleMask="true" 
              class="w-full mb-3" 
              inputClass="w-full" 
              :inputStyle="{ padding: '1rem' }"
              :feedback="false"
              :class="{ 'p-invalid': submitted && !password }"
            />

            <div class="flex align-items-center justify-content-between mb-5 gap-5">
              <div class="flex align-items-center">
                <Checkbox v-model="checked" id="rememberme1" binary class="mr-2"></Checkbox>
                <label for="rememberme1">Remember me</label>
              </div>
              <a class="font-medium no-underline ml-2 text-right cursor-pointer" style="color: var(--primary-color)">Forgot password?</a>
            </div>

            <Button 
              label="Sign In" 
              class="w-full p-3 text-xl"
              :loading="loading"
              @click="handleLogin"
              :disabled="loading"
            ></Button>

            <!-- Demo Credentials Notice -->
            <div class="mt-5 p-4 border-round" style="background: var(--surface-100);">
              <div class="text-center">
                <i class="pi pi-info-circle text-blue-500 mr-2"></i>
                <span class="text-600 text-sm">Demo credentials: </span>
                <span class="text-900 font-bold">admin / admin987</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <div class="col-12 xl:col-6 flex align-items-center justify-content-center">
        <div class="text-center">
          <div class="mb-4">
            <i class="pi pi-database text-primary" style="font-size: 8rem; opacity: 0.2;"></i>
          </div>
          <h3 class="text-900 font-semibold mb-3">Secure Access</h3>
          <p class="text-600 line-height-3 m-0">
            Enterprise-grade water district management system with real-time monitoring and operational intelligence.
          </p>
        </div>
      </div>
    </div>

    <!-- Error Toast -->
    <Toast ref="toast" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '../stores/auth'

// PrimeVue Components
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import Password from 'primevue/password'
import Checkbox from 'primevue/checkbox'
import Toast from 'primevue/toast'

const router = useRouter()
const toast = useToast()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const checked = ref(false)
const loading = ref(false)
const submitted = ref(false)

const handleLogin = async () => {
  submitted.value = true
  
  if (!username.value || !password.value) {
    toast.add({ 
      severity: 'error', 
      summary: 'Error', 
      detail: 'Please fill in all fields', 
      life: 3000 
    })
    return
  }

  loading.value = true
  
  try {
    const result = await authStore.login(username.value, password.value)
    
    if (result.success) {
      console.log('Login successful, auth store state:', {
        isAuthenticated: authStore.isAuthenticated,
        isLoggedIn: authStore.isLoggedIn,
        token: !!authStore.token,
        user: authStore.user
      })
      
      toast.add({ 
        severity: 'success', 
        summary: 'Success', 
        detail: 'Login successful', 
        life: 2000 
      })
      
      // Force redirect with nextTick to ensure store state is updated
      await new Promise(resolve => setTimeout(resolve, 100))
      router.replace('/')
    } else {
      toast.add({ 
        severity: 'error', 
        summary: 'Error', 
        detail: result.error, 
        life: 3000 
      })
    }
  } catch (error) {
    toast.add({ 
      severity: 'error', 
      summary: 'Error', 
      detail: 'An unexpected error occurred', 
      life: 3000 
    })
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.pi-eye {
  transform: scale(1.6);
  margin-right: 1rem;
}

.pi-eye-slash {
  transform: scale(1.6);
  margin-right: 1rem;
}
</style>