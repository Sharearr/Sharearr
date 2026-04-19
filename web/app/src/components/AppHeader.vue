<script setup lang="ts">
import { computed } from 'vue'
import { useTheme } from 'vuetify'
import {
  MagnifyingGlassIcon,
  HeartIcon,
  UserCircleIcon,
} from '@heroicons/vue/24/solid'

const theme = useTheme()
const dark = computed({
  get: () => theme.global.name.value === 'dark',
  set: (val) => { theme.change(val ? 'dark' : 'light') },
})
</script>

<template>
  <v-app-bar flat density="compact">

    <template #prepend>
      <div class="w-8 h-8 rounded bg-surface-variant ml-2 mr-3" aria-label="Logo placeholder" />
      <span class="text-xl font-bold tracking-wide font-display mr-4">Sharearr</span>
      <v-text-field
        type="search"
        placeholder="Search..."
        density="compact"
        variant="solo-filled"
        flat
        hide-details
        rounded="lg"
        class="w-80"
      >
        <template #prepend-inner>
          <MagnifyingGlassIcon class="w-4 h-4 mr-1" />
        </template>
      </v-text-field>
    </template>

    <template #append>
      <v-switch
        v-model="dark"
        hide-details
        inset
        density="compact"
        true-icon="$moon"
        false-icon="$sun"
        class="mx-2"
      />

      <v-tooltip text="Support the project" location="bottom">
        <template #activator="{ props }">
          <v-btn v-bind="props" icon variant="text" size="small" aria-label="Donate">
            <HeartIcon class="w-5 h-5" />
          </v-btn>
        </template>
      </v-tooltip>

      <v-btn icon variant="text" size="small" aria-label="User profile" class="mr-2">
        <UserCircleIcon class="w-6 h-6" />
      </v-btn>
    </template>

  </v-app-bar>
</template>

<style scoped></style>
