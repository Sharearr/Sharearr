<script setup lang="ts">
import { computed } from 'vue'
import { useTheme } from 'vuetify'

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
      <v-app-bar-title class="text-xl font-bold tracking-wide font-heading">Sharearr</v-app-bar-title>
    </template>

    <div class="w-80 ml-4">
      <v-text-field
        type="search"
        placeholder="Search..."
        density="compact"
        variant="solo-filled"
        flat
        hide-details
        clearable
        rounded="lg"
        prepend-inner-icon="$search"
      />
    </div>

    <v-spacer />

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
          <v-btn v-bind="props" icon="$donate" variant="text" size="small" aria-label="Donate" />
        </template>
      </v-tooltip>

      <v-btn icon="$user" variant="text" size="small" aria-label="User profile" class="mr-2" />
    </template>

  </v-app-bar>
</template>

<style scoped>
:deep(.v-selection-control i.v-icon) {
  color: rgb(var(--v-theme-icon-sun));
}
:deep(.v-selection-control.v-selection-control--dirty i.v-icon) {
  color: rgb(var(--v-theme-icon-moon));
}
</style>
