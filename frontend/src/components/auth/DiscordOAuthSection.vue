<template>
  <div>
    <button type="button" :disabled="disabled" class="btn btn-secondary w-full" @click="startLogin">
      <svg class="mr-2 h-5 w-5" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
        <path
          d="M20.317 4.A19.791 19.791 0 0015.885 3c-.191.343-.403.804-.552 1.173a18.27 18.27 0 00-5.001 0A12.64 12.64 0 009.78 3a19.736 19.736 0 00-4.434 1.371C2.57 8.548 1.817 12.61 2.195 16.615a19.9 19.9 0 005.44 2.753c.44-.602.833-1.24 1.173-1.91a12.9 12.9 0 01-1.84-.885c.154-.115.304-.234.448-.356 3.546 1.621 7.397 1.621 10.901 0 .146.122.296.241.448.356a12.8 12.8 0 01-1.843.887c.34.668.732 1.306 1.173 1.908a19.86 19.86 0 005.442-2.753c.443-4.641-.756-8.666-3.217-12.246zM9.49 14.154c-1.08 0-1.966-.982-1.966-2.19 0-1.209.866-2.19 1.966-2.19 1.108 0 1.984.991 1.966 2.19 0 1.208-.866 2.19-1.966 2.19zm5.02 0c-1.08 0-1.966-.982-1.966-2.19 0-1.209.866-2.19 1.966-2.19 1.108 0 1.984.991 1.966 2.19 0 1.208-.858 2.19-1.966 2.19z"
        />
      </svg>
      {{ t('auth.discord.signIn') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

defineProps<{
  disabled?: boolean
}>()

const route = useRoute()
const { t } = useI18n()

function startLogin(): void {
  const redirectTo = (route.query.redirect as string) || '/dashboard'
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api/v1'
  const normalized = apiBase.replace(/\/$/, '')
  const startURL = `${normalized}/auth/oauth/discord/start?redirect=${encodeURIComponent(redirectTo)}`
  window.location.href = startURL
}
</script>
