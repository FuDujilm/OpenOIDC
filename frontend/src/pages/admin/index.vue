<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'
import { Users, Monitor, Activity, Loader2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t, tm, locale } = useI18n()

interface AuditEvent {
  id: string
  action: string
  user_id: string | null
  user_uid?: number | string
  user_email?: string
  user_display_name?: string
  resource_type?: string
  details?: Record<string, any>
  details_text?: string
  created_at: string
}

interface Stats {
  total_users: number
  total_clients: number
  total_sessions: number
  recent_events: AuditEvent[]
}

const stats = ref<Stats | null>(null)
const loading = ref(false)
const error = ref('')

async function fetchStats() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<Stats>('/admin/stats')
    stats.value = res.data ?? null
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function formatTime(iso: string) {
  const displayLocale = String(locale.value).startsWith('en') ? 'en-US' : 'zh-CN'
  return new Date(iso).toLocaleString(displayLocale, {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
  })
}

function recordLabel(path: string, key: string | undefined | null, fallback = '-') {
  if (!key) return fallback
  const record = tm(path)
  if (record && typeof record === 'object') {
    const value = (record as Record<string, unknown>)[key]
    if (typeof value === 'string') return value
  }
  return key
}

function actionLabel(action: string): string {
  return recordLabel('adminAudit.actions', action, action)
}

function formatDetailKey(key: string): string {
  return recordLabel('adminAudit.detailKeys', key, key)
}

function formatDetailValue(value: any): string {
  if (value === null || value === undefined || value === '') return '-'
  const raw = String(value)
  return recordLabel('adminAudit.detailValues', raw, raw)
}

function formatDetails(event: AuditEvent): string {
  const details = event.details ?? {}
  const keys = Object.keys(details)
  if (keys.length === 0) return event.details_text || '-'
  return keys.map(key => `${formatDetailKey(key)}=${formatDetailValue(details[key])}`).join(', ')
}

function actorDisplay(event: AuditEvent): string {
  if (event.user_email) return event.user_email
  return '-'
}

function uidDisplay(event: AuditEvent): string {
  if (event.user_uid !== undefined && event.user_uid !== null) return String(event.user_uid)
  if (event.user_id) return event.user_id
  return '-'
}

const cards = [
  { key: 'total_users', labelKey: 'adminOverview.totalUsers', icon: Users },
  { key: 'total_clients', labelKey: 'adminOverview.totalClients', icon: Monitor },
  { key: 'total_sessions', labelKey: 'adminOverview.activeSessions', icon: Activity },
] as const

onMounted(fetchStats)
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-lg font-semibold">{{ $t('adminOverview.title') }}</h2>
      <p class="text-sm text-muted-foreground mt-1">{{ $t('adminOverview.subtitle') }}</p>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-20 text-muted-foreground">
      <Loader2 class="w-5 h-5 animate-spin mr-2" /> {{ $t('loading') }}
    </div>

    <div v-else-if="error" class="text-center py-20 text-destructive">{{ error }}</div>

    <template v-else-if="stats">
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-5 mb-10">
        <div v-for="card in cards" :key="card.key" class="relative border border-border rounded-xl p-6 bg-background">
          <component :is="card.icon" class="absolute top-5 right-5 w-5 h-5 text-muted-foreground/50" />
          <div class="text-3xl font-bold tracking-tight">{{ (stats as any)[card.key]?.toLocaleString() ?? '—' }}</div>
          <div class="text-sm text-muted-foreground mt-1">{{ t(card.labelKey) }}</div>
        </div>
      </div>

      <div>
        <h2 class="text-lg font-semibold mb-4">{{ t('adminOverview.recentActivity') }}</h2>

        <div v-if="!stats.recent_events || stats.recent_events.length === 0" class="border border-dashed border-border rounded-xl py-8 text-center text-muted-foreground text-sm">
          {{ t('adminOverview.noActivity') }}
        </div>

        <div v-else>
          <div class="hidden md:block border border-border rounded-xl overflow-hidden">
            <table class="w-full text-sm table-fixed">
              <thead>
                <tr class="border-b border-border bg-muted/40">
                  <th class="text-left px-4 py-3 font-medium text-muted-foreground w-32">{{ t('adminAudit.time') }}</th>
                  <th class="text-left px-4 py-3 font-medium text-muted-foreground w-56">{{ t('adminAudit.actor') }}</th>
                  <th class="text-left px-4 py-3 font-medium text-muted-foreground w-32">{{ t('adminUsers.uid') }}</th>
                  <th class="text-left px-4 py-3 font-medium text-muted-foreground w-40">{{ t('adminAudit.action') }}</th>
                  <th class="text-left px-4 py-3 font-medium text-muted-foreground">{{ t('adminAudit.details') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="event in stats.recent_events" :key="event.id" class="border-b border-border last:border-b-0 hover:bg-muted/30 transition-colors">
                  <td class="px-4 py-3 text-muted-foreground whitespace-nowrap text-xs">{{ formatTime(event.created_at) }}</td>
                  <td class="px-4 py-3 text-xs min-w-0">
                    <div class="truncate" :title="actorDisplay(event)">{{ actorDisplay(event) }}</div>
                  </td>
                  <td class="px-4 py-3 text-muted-foreground text-xs font-mono truncate" :title="uidDisplay(event)">{{ uidDisplay(event) }}</td>
                  <td class="px-4 py-3">
                    <span class="inline-block px-2 py-0.5 rounded-md bg-muted text-xs font-medium max-w-full truncate">{{ actionLabel(event.action) }}</span>
                  </td>
                  <td class="px-4 py-3 text-muted-foreground text-xs truncate" :title="formatDetails(event)">{{ formatDetails(event) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="md:hidden space-y-3">
            <div v-for="event in stats.recent_events" :key="event.id" class="border border-border rounded-xl p-4 bg-background">
              <div class="flex items-start justify-between gap-3">
                <span class="px-2 py-0.5 rounded-md bg-muted text-xs font-medium break-words min-w-0">{{ actionLabel(event.action) }}</span>
                <span class="text-xs text-muted-foreground whitespace-nowrap shrink-0">{{ formatTime(event.created_at) }}</span>
              </div>
              <div class="mt-3 text-xs text-muted-foreground break-words">
                <span class="font-medium text-foreground">{{ t('adminAudit.actor') }}：</span>{{ actorDisplay(event) }}
              </div>
              <div class="mt-2 text-xs text-muted-foreground break-words">
                <span class="font-medium text-foreground">{{ t('adminUsers.uid') }}：</span>{{ uidDisplay(event) }}
              </div>
              <div class="mt-2 text-xs text-muted-foreground break-words">
                <span class="font-medium text-foreground">{{ t('adminAudit.details') }}：</span>{{ formatDetails(event) }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
