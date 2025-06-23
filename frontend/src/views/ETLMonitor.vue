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
          <div class="layout-topbar-item-text hidden lg:block">ETL Monitor</div>
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
        <ETLDashboard />
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
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '../stores/auth'

// Components
import Toast from 'primevue/toast'
import Button from 'primevue/button'
import PanelMenu from 'primevue/panelmenu'
import Avatar from 'primevue/avatar'
import OverlayPanel from 'primevue/overlaypanel'
import Divider from 'primevue/divider'
import ETLDashboard from '../components/ETLDashboard.vue'

const router = useRouter()
const toast = useToast()
const authStore = useAuthStore()

// Refs
const sidebarVisible = ref(false)
const userMenu = ref()

// Menu items (same as Dashboard)
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
        label: 'Natural Language',
        icon: 'pi pi-fw pi-comment'
      },
      {
        key: '1_1',
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