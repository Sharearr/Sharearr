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
  <header class="flex items-center gap-6 px-6 py-3 dark:bg-gray-900 light:bg-gray-100 dark:text-white light:text-gray-900">
    <div class="flex-shrink-0 w-8 h-8 rounded dark:bg-gray-700 light:bg-gray-300" aria-label="Logo placeholder" />

    <span class="flex-shrink-0 text-xl font-bold tracking-wide font-display">
      Sharearr
    </span>

    <div class="flex items-center gap-2 dark:bg-gray-800 light:bg-white rounded-lg px-3 py-1.5 w-80">
      <MagnifyingGlassIcon class="w-4 h-4 text-gray-400 flex-shrink-0" />
      <input
        type="search"
        placeholder="Search..."
        class="bg-transparent text-sm dark:text-white light:text-gray-900 dark:placeholder-gray-500 light:placeholder-gray-400 outline-none w-full"
      />
    </div>

    <div class="flex-1" />

    <v-switch
      v-model="dark"
      hide-details
      inset
      density="compact"
      color="primary"
      true-icon="$moon"
      false-icon="$sun"
      class="flex-shrink-0"
    />

    <div class="relative group flex-shrink-0">
      <button
        class="flex items-center text-gray-400 hover:text-pink-400 transition-colors"
        aria-label="Donate"
      >
        <HeartIcon class="w-5 h-5" />
      </button>
      <span class="absolute right-0 top-full mt-1.5 px-2 py-1 text-xs rounded dark:bg-gray-800 light:bg-gray-200 dark:text-gray-200 light:text-gray-700 whitespace-nowrap opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
        Support the project
      </span>
    </div>

    <div class="flex-shrink-0">
      <button
        class="flex items-center text-gray-400 dark:hover:text-white light:hover:text-gray-900 transition-colors"
        aria-label="User profile"
      >
        <UserCircleIcon class="w-6 h-6" />
      </button>
    </div>

  </header>
</template>
