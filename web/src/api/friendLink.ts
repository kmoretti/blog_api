import request from '@/utils/request'
import type { ApiResponse } from '@/model/response'
import type {
  FriendLinkListParams,
  PaginatedFriendLinks,
  CreateFriendLinkPayload,
  UpdateFriendLinkPayload
} from '@/model/friendLink'

/**
 * 获取友链列表
 */
export const getFriendLinks = (params: FriendLinkListParams): Promise<ApiResponse<PaginatedFriendLinks>> => {
  return request({
    url: '/action/friend',
    method: 'get',
    params
  })
}


/**
 * 创建友链
 */
export const createFriendLink = (data: CreateFriendLinkPayload): Promise<ApiResponse> => {
  return request({
    url: '/action/friend',
    method: 'post',
    data
  })
}


/**
 * 更新友链
 */
export const updateFriendLink = (id: number, payload: UpdateFriendLinkPayload): Promise<ApiResponse> => {
  const fieldMap: Record<string, string> = {
    name: 'website_name',
    link: 'website_url',
    avatar: 'website_icon_url',
    friend_link_page: 'friend_link_page',
    feed: 'feed'
  }
  const mappedData: Record<string, unknown> = {}
  Object.entries(payload.data).forEach(([key, value]) => {
    const targetKey = fieldMap[key] ?? key
    mappedData[targetKey] = value
  })
  return request({
    url: `/action/friend/${id}`,
    method: 'put',
    data: {
      ...payload,
      data: mappedData
    }
  })
}


/**
 * 删除友链
 */
export const deleteFriendLink = (id: number): Promise<ApiResponse> => {
  return request({
    url: `/action/friend/${id}`,
    method: 'delete'
  })
}
