import request from '@/utils/request';
import type { SystemConfig } from '@/model/config';
import type { ApiResponse } from '@/model/response';

const CONFIG_FILE_PATH = '/config/system_config.json';

/**
 * 获取系统配置
 * 注意：资源 API 直接返回文件内容，不包装在 ApiResponse 中
 * @returns SystemConfig
 */
export const getSystemConfig = () => {
  return request.get<any, SystemConfig>(`/action/resource/${CONFIG_FILE_PATH}`, {
    params: {
      // 添加时间戳以防止缓存
      _t: new Date().getTime(),
    },
  });
};

/**
 * 批量更新系统配置项
 * @param updates 配置更新数组
 * @returns
 */
export const updateSystemConfig = (updates: { key: string; value: any }[]) => {
  return request.put<any, ApiResponse<{ message: string }>>('/action/config', updates);
};

/**
 * 请求重启后端进程。
 * 实际拉起由外部守护进程负责。
 */
export const restartSystem = () => {
  return request.post<any, ApiResponse<{ detail: string }>>('/action/system/restart');
};
