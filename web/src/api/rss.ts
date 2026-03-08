import request from '@/utils/request'
import type { ApiResponse } from '@/model/response'

import type { RssFeed, RssPost, PaginatedResponse } from '@/model/rss'

/**
 * Fetches the list of all RSS feeds.
 * Corresponds to GET /api/action/rss
 */
export const getRssFeeds = (
  page = 1,
  pageSize = 20
): Promise<ApiResponse<PaginatedResponse<RssFeed>>> => {
  return request({
    url: '/action/rss',
    method: 'get',
    params: {
      page,
      page_size: pageSize
    }
  })
}

/**
 * Fetches posts for a specific RSS feed.
 * Corresponds to GET /api/rss
 * @param rssId - The ID of the RSS feed.
 * @param page - The page number for pagination.
 * @param pageSize - The number of items per page.
 */
export const getPostsByFeed = (
  rssId: number,
  page = 1,
  pageSize = 20
): Promise<ApiResponse<PaginatedResponse<RssPost>>> => {
  return request({
    url: '/public/rss',
    method: 'get',
    params: {
      rss_id: rssId,
      page,
      page_size: pageSize
    }
  })
}

/**
 * Fetches posts from all RSS feeds.
 * Corresponds to GET /api/public/rss
 * @param page - The page number for pagination.
 * @param pageSize - The number of items per page.
 */
export const getAllPosts = (
  page = 1,
  pageSize = 20
): Promise<ApiResponse<PaginatedResponse<RssPost>>> => {
  return request({
    url: '/public/rss',
    method: 'get',
    params: {
      page,
      page_size: pageSize
    }
  })
}

/**
 * Deletes one or more RSS feeds by their IDs.
 * Corresponds to DELETE /api/action/rss
 * @param ids - An array of RSS feed IDs to delete.
 */
export const deleteRssFeed = (ids: number[]): Promise<ApiResponse> => {
  return request({
    url: `/action/rss/${ids[0]}`,
    method: 'delete'
  })
}

/**
 * Updates an existing RSS feed.
 * Corresponds to PUT /api/action/rss
 * @param id - The ID of the RSS feed to update.
 * @param data - The data to update.
 */
export const updateRssFeed = (
  id: number,
  data: { name?: string; rss_url?: string; status?: string; is_died?: boolean }
): Promise<ApiResponse> => {
  return request({
    url: `/action/rss/${id}`,
    method: 'put',
    data: {
      data
    }
  })
}

/**
 * Creates a new RSS feed.
 * Corresponds to POST /api/action/rss
 * @param name - The name of the RSS feed.
 * @param rss_url - The URL of the RSS feed to create.
 */
export const createRssFeed = (
  name: string,
  rss_url: string
): Promise<ApiResponse> => {
  return request({
    url: '/action/rss',
    method: 'post',
    data: {
      name,
      rss_url
    }
  })
}
