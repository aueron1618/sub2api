/**
 * User Quota API
 * API for regular users to view their own quotas and progress
 */

import { apiClient } from './client'
import type { UserSubscription, SubscriptionProgress } from '@/types'

/**
 * Quota summary for user dashboard
 */
export interface QuotaSummary {
  active_count: number
  total_used_usd: number
  quotas: Array<{
    id: number
    group_name: string
    status: string
    daily_used_usd: number
    daily_limit_usd: number
    weekly_used_usd: number
    weekly_limit_usd: number
    monthly_used_usd: number
    monthly_limit_usd: number
    expires_at: string | null
  }>
}

/**
 * Get list of current user's quotas
 */
export async function getMyQuotas(): Promise<UserSubscription[]> {
  const response = await apiClient.get<UserSubscription[]>('/quotas')
  return response.data
}

/**
 * Get current user's active quotas
 */
export async function getActiveQuotas(): Promise<UserSubscription[]> {
  const response = await apiClient.get<UserSubscription[]>('/quotas/active')
  return response.data
}

/**
 * Get progress for all user's active quotas
 */
export async function getQuotasProgress(): Promise<SubscriptionProgress[]> {
  const response = await apiClient.get<SubscriptionProgress[]>('/quotas/progress')
  return response.data
}

/**
 * Get quota summary for dashboard display
 */
export async function getQuotaSummary(): Promise<QuotaSummary> {
  const response = await apiClient.get<QuotaSummary>('/quotas/summary')
  return response.data
}

/**
 * Get progress for a specific quota
 */
export async function getQuotaProgress(
  quotaId: number
): Promise<SubscriptionProgress> {
  const response = await apiClient.get<SubscriptionProgress>(
    `/quotas/${quotaId}/progress`
  )
  return response.data
}

const quotasAPI = {
  getMyQuotas,
  getActiveQuotas,
  getQuotasProgress,
  getQuotaSummary,
  getQuotaProgress
}

export default quotasAPI
