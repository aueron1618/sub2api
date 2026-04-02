/**
 * Admin Quota API endpoints
 * Handles user quota management for administrators
 */

import { apiClient } from '../client'
import type {
  UserSubscription,
  SubscriptionProgress,
  AssignSubscriptionRequest,
  BulkAssignSubscriptionRequest,
  ExtendSubscriptionRequest,
  PaginatedResponse
} from '@/types'

/**
 * List all quotas with pagination
 * @param page - Page number (default: 1)
 * @param pageSize - Items per page (default: 20)
 * @param filters - Optional filters (status, user_id, group_id, sort_by, sort_order)
 * @returns Paginated list of quotas
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: 'active' | 'expired' | 'revoked'
    user_id?: number
    group_id?: number
    platform?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<UserSubscription>> {
  const { data } = await apiClient.get<PaginatedResponse<UserSubscription>>(
    '/admin/quotas',
    {
      params: {
        page,
        page_size: pageSize,
        ...filters
      },
      signal: options?.signal
    }
  )
  return data
}

/**
 * Get quota by ID
 * @param id - Quota ID
 * @returns Quota details
 */
export async function getById(id: number): Promise<UserSubscription> {
  const { data } = await apiClient.get<UserSubscription>(`/admin/quotas/${id}`)
  return data
}

/**
 * Get quota progress
 * @param id - Quota ID
 * @returns Quota progress with usage stats
 */
export async function getProgress(id: number): Promise<SubscriptionProgress> {
  const { data } = await apiClient.get<SubscriptionProgress>(`/admin/quotas/${id}/progress`)
  return data
}

/**
 * Assign quota to user
 * @param request - Assignment request
 * @returns Created quota
 */
export async function assign(request: AssignSubscriptionRequest): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>('/admin/quotas/assign', request)
  return data
}

/**
 * Bulk assign quotas to multiple users
 * @param request - Bulk assignment request
 * @returns Created quotas
 */
export async function bulkAssign(
  request: BulkAssignSubscriptionRequest
): Promise<UserSubscription[]> {
  const { data } = await apiClient.post<UserSubscription[]>(
    '/admin/quotas/bulk-assign',
    request
  )
  return data
}

/**
 * Extend quota validity
 * @param id - Quota ID
 * @param request - Extension request with days
 * @returns Updated quota
 */
export async function extend(
  id: number,
  request: ExtendSubscriptionRequest
): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>(
    `/admin/quotas/${id}/extend`,
    request
  )
  return data
}

/**
 * Revoke quota
 * @param id - Quota ID
 * @returns Success confirmation
 */
export async function revoke(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/quotas/${id}`)
  return data
}

/**
 * Reset daily, weekly, and/or monthly usage quota for a quota record
 * @param id - Quota ID
 * @param options - Which windows reset
 * @returns Updated quota
 */
export async function resetQuota(
  id: number,
  options: { daily: boolean; weekly: boolean; monthly: boolean }
): Promise<UserSubscription> {
  const { data } = await apiClient.post<UserSubscription>(
    `/admin/quotas/${id}/reset-quota`,
    options
  )
  return data
}

/**
 * List quotas by group
 * @param groupId - Group ID
 * @param page - Page number
 * @param pageSize - Items per page
 * @returns Paginated list of quotas in the group
 */
export async function listByGroup(
  groupId: number,
  page: number = 1,
  pageSize: number = 20
): Promise<PaginatedResponse<UserSubscription>> {
  const { data } = await apiClient.get<PaginatedResponse<UserSubscription>>(
    `/admin/groups/${groupId}/quotas`,
    {
      params: { page, page_size: pageSize }
    }
  )
  return data
}

/**
 * List quotas by user
 * @param userId - User ID
 * @param page - Page number
 * @param pageSize - Items per page
 * @returns Paginated list of user's quotas
 */
export async function listByUser(
  userId: number,
  page: number = 1,
  pageSize: number = 20
): Promise<PaginatedResponse<UserSubscription>> {
  const { data } = await apiClient.get<PaginatedResponse<UserSubscription>>(
    `/admin/users/${userId}/quotas`,
    {
      params: { page, page_size: pageSize }
    }
  )
  return data
}

export const quotasAPI = {
  list,
  getById,
  getProgress,
  assign,
  bulkAssign,
  extend,
  revoke,
  resetQuota,
  listByGroup,
  listByUser
}

export default quotasAPI
