<template>
  <div class="etl-logs-view">
    <ETLAllLogsViewer />
  </div>
</template>

<script setup>
import ETLAllLogsViewer from '../components/ETLAllLogsViewer.vue'
</script>

<style scoped>
.etl-logs-view {
  min-height: 100vh;
  background-color: var(--surface-50);
}
</style>