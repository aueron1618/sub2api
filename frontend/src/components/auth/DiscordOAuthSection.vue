<template>
  <div>
    <button type="button" :disabled="disabled" class="btn btn-secondary w-full" @click="startLogin">
      <DiscordIcon class="mr-2" style="width: 20px; height: 20px" />
      {{ t('auth.discord.signIn') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import DiscordIcon from '@/components/icons/DiscordIcon.vue'

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
