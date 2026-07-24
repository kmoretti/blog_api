import type { ApiResponse } from '@/model/response'

// Corresponds to model.StatusData in the backend
export interface FriendLinkStatusCount {
  status: string
  count: number
}

export interface RssPostCountMonthly {
  month: string
  count: number
}

export interface StatusData {
  friend_link_count: number
  rss_count: number
  rss_post_count: number
  friend_link_status_pie: FriendLinkStatusCount[]
  rss_post_count_monthly: RssPostCountMonthly[]
}

// Corresponds to model.SystemStatus in the backend
export interface SystemStatus {
  uptime: string
  status_data: StatusData
  database_size_bytes: number
  data_folder_size_bytes: number
  time: number
}

// The actual data structure within the main ApiResponse
export type SystemStatusResponse = ApiResponse<SystemStatus>
