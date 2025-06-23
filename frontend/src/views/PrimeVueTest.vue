<template>
  <div class="layout-container">
    <div class="layout-grid">
      <!-- Stats Widget Test -->
      <div class="stats">
        <div v-for="(stat, index) in stats" :key="index" class="layout-card">
          <div class="stats-header">
            <span class="stats-title">{{ stat.title }}</span>
            <span class="stats-icon-box">
              <i :class="['pi', stat.icon]"></i>
            </span>
          </div>
          <div class="stats-content">
            <div class="stats-value">{{ stat.value }}</div>
            <div class="stats-subtitle">{{ stat.subtitle }}</div>
          </div>
        </div>
      </div>

      <!-- Button Test -->
      <div class="layout-card">
        <h3>Button Test</h3>
        <div style="display: flex; gap: 1rem; flex-wrap: wrap;">
          <Button label="Primary" />
          <Button label="Secondary" outlined />
          <Button label="Success" severity="success" />
          <Button label="Warning" severity="warn" />
          <Button label="Danger" severity="danger" />
          <Button icon="pi pi-check" label="With Icon" />
        </div>
      </div>

      <!-- Card Test -->
      <div class="layout-grid-row">
        <Card class="flex-1">
          <template #title>Simple Card</template>
          <template #content>
            <p>This is a simple card to test PrimeVue styling.</p>
            <Button label="Action" class="mt-2" />
          </template>
        </Card>
        
        <Card class="flex-1">
          <template #title>Another Card</template>
          <template #content>
            <p>Testing multiple cards side by side.</p>
            <InputText placeholder="Test input" class="w-full" />
          </template>
        </Card>
      </div>

      <!-- Products Table Test -->
      <div class="layout-card">
        <div class="products-header">
          <span class="products-title">Products Overview Test</span>
          <IconField class="search-field">
            <InputIcon class="pi pi-search" />
            <InputText
              v-model="searchQuery"
              placeholder="Search products..."
              class="products-search"
            />
          </IconField>
        </div>
        <div class="products-table-container">
          <DataTable
            :value="filteredProducts"
            v-model:selection="selectedProduct"
            selectionMode="single"
            :loading="loading"
            :rows="5"
            class="products-table"
          >
            <Column field="name" header="Name" sortable></Column>
            <Column field="category" header="Category" sortable></Column>
            <Column field="price" header="Price" sortable>
              <template #body="{ data }"> ${{ data.price }} </template>
            </Column>
            <Column field="status" header="Status">
              <template #body="{ data }">
                <Tag
                  :severity="
                    data.status === 'In Stock' ? 'success' : data.status === 'Low Stock' ? 'warn' : 'danger'
                  "
                >
                  {{ data.status }}
                </Tag>
              </template>
            </Column>
          </DataTable>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from "vue";

// Stats data
const stats = [
  {
    title: "Running Jobs",
    icon: "pi-spin pi-cog",
    value: "3",
    subtitle: "ETL Processes",
  },
  {
    title: "Completed",
    icon: "pi-check-circle", 
    value: "15",
    subtitle: "Today",
  },
  {
    title: "Data Points",
    icon: "pi-chart-line",
    value: "70,297",
    subtitle: "Total Records",
  },
  {
    title: "Success Rate",
    icon: "pi-verified",
    value: "98.5%",
    subtitle: "Last 24h",
  },
];

// Products data  
const products = ref([
  {
    name: "ETL Job 1",
    category: "Historical Load",
    price: 1234,
    status: "In Stock",
  },
  {
    name: "ETL Job 2", 
    category: "Real-time Sync",
    price: 567,
    status: "Low Stock",
  },
  {
    name: "ETL Job 3",
    category: "Data Validation", 
    price: 890,
    status: "Out of Stock",
  },
  { 
    name: "ETL Job 4", 
    category: "Cleanup", 
    price: 234, 
    status: "In Stock" 
  },
]);

const selectedProduct = ref(null);
const searchQuery = ref("");
const loading = ref(false);
const filteredProducts = ref([]);

const searchProducts = () => {
  loading.value = true;
  filteredProducts.value = products.value.filter(
    (product) =>
      product.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      product.category.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
      product.status.toLowerCase().includes(searchQuery.value.toLowerCase())
  );
  setTimeout(() => {
    loading.value = false;
  }, 300);
};

watch(searchQuery, () => {
  searchProducts();
});

onMounted(() => {
  filteredProducts.value = [...products.value];
});
</script>

<style scoped>
.flex-1 {
  flex: 1;
}

.mt-2 {
  margin-top: 0.5rem;
}

.w-full {
  width: 100%;
}
</style>