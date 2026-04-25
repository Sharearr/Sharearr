<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function login() {
  console.log('login called')
  error.value = ''
  loading.value = true
  try {
    const res = await fetch('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value, password: password.value }),
    })
    if (res.status === 401) {
      error.value = 'Invalid username or password.'
      return
    }
    if (!res.ok) {
      error.value = 'An error occurred. Please try again.'
      return
    }
    const data = await res.json()
    auth.setAuth(data.user, data.expires_at)
  } catch {
    error.value = 'An error occurred. Please try again.'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <v-main class="bg-background">
    <v-container class="h-full flex items-center justify-center">
      <v-card width="360" rounded="lg" class="pa-6">
        <v-card-title class="font-heading text-2xl font-bold mb-2">Sign in</v-card-title>

        <v-card-text class="pa-0">
          <v-alert v-if="error" type="error" variant="tonal" class="mb-4" density="compact">
            {{ error }}
          </v-alert>

          <v-text-field
            v-model="username"
            label="Username"
            autocomplete="username"
            density="compact"
            variant="outlined"
            class="mb-3"
            hide-details="auto"
            @keyup.enter="login"
          />

          <v-text-field
            v-model="password"
            label="Password"
            type="password"
            autocomplete="current-password"
            density="compact"
            variant="outlined"
            hide-details="auto"
            @keyup.enter="login"
          />
        </v-card-text>

        <v-card-actions class="pa-0 mt-6">
          <v-btn
            block
            color="primary"
            variant="flat"
            rounded="lg"
            :loading="loading"
            @click="login"
          >
            Sign in
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-container>
  </v-main>
</template>
