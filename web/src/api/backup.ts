import request from '@/utils/request'
import type { ApiResponse } from '@/model/response'
import type { BackupModule, ImportResult, ImportStrategy, RestoreResult } from '@/model/backup'

export const exportFullBackup = (): Promise<Blob> => {
  return request({
    url: '/action/backup/export',
    method: 'post',
    responseType: 'blob'
  })
}

export const importFullBackup = (file: File): Promise<ApiResponse<RestoreResult>> => {
  const form = new FormData()
  form.append('file', file)
  return request({
    url: '/action/backup/import',
    method: 'post',
    data: form
  })
}

export const exportModule = (module: BackupModule): Promise<Blob> => {
  return request({
    url: `/action/export/${encodeURIComponent(module)}`,
    method: 'get',
    responseType: 'blob'
  })
}

export const importModule = (
  module: BackupModule,
  file: File,
  strategy: ImportStrategy
): Promise<ApiResponse<ImportResult>> => {
  const form = new FormData()
  form.append('file', file)
  return request({
    url: `/action/import/${encodeURIComponent(module)}`,
    method: 'post',
    data: form,
    params: { strategy }
  })
}
