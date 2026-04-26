<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import AppHeader from './components/AppHeader.vue'
import LoginView from './components/LoginView.vue'

const auth = useAuthStore()
</script>

<template>
  <v-app>
    <AppHeader />
    <LoginView v-if="!auth.isAuthenticated" />
    <v-main v-else>
      <!-- page content -->
    </v-main>
  </v-app>
</template>

<style scoped></style>
