import { ApiError } from '@/core/http'

const MESSAGE_ZH: Record<string, string> = {
  'file is required': '请选择要上传的文件',
  'file is empty': '文件为空，无法上传',
  'filename must include an allowed extension': '文件名需包含受支持的扩展名',
  'file type is not allowed': '不支持的文件类型',
  'file exceeds size limit': '文件超过大小限制（文档与压缩包上限不同，请压缩后重试）',
  'file content does not match extension': '文件内容与扩展名不一致，请检查文件是否损坏或被篡改',
  'object storage is not configured': '资料存储未配置，请联系管理员',
  'failed to read upload': '读取上传内容失败，请重试',
  'permission denied': '无权执行此操作',
  'access denied': '无权访问该资料',
  'resource not found': '资料不存在或未向学生发布',
  'not signed in or session expired': '登录已过期，请重新登录',
  'internal server error': '服务暂时不可用，请稍后重试',
  'network error: is the backend running?': '网络异常，请检查连接或确认后端已启动',
}

function matchKnownMessage(raw: string): string | undefined {
  const key = raw.trim().toLowerCase()
  if (MESSAGE_ZH[key]) return MESSAGE_ZH[key]
  if (key.includes('size limit')) return MESSAGE_ZH['file exceeds size limit']
  if (key.includes('not allowed') && key.includes('extension')) {
    return MESSAGE_ZH['file type is not allowed']
  }
  if (key.includes('chapterkey not found') || key.includes('sectionkey not found')) {
    return '所选章节或小节不存在，请刷新课程结构后重试'
  }
  return undefined
}

/** 将课程资料相关 API 错误转为用户可读中文。 */
export function mapCourseResourceError(error: unknown, fallback = '操作失败'): string {
  if (error instanceof ApiError) {
    if (error.code === 0) {
      return MESSAGE_ZH['network error: is the backend running?'] ?? fallback
    }
    if (error.code === 401) {
      return MESSAGE_ZH['not signed in or session expired'] ?? fallback
    }
    if (error.code === 403) {
      return matchKnownMessage(error.message) ?? '无权访问该资料'
    }
    if (error.code === 404) {
      return MESSAGE_ZH['resource not found'] ?? fallback
    }
    if (error.code === 500) {
      return MESSAGE_ZH['internal server error'] ?? fallback
    }
    return matchKnownMessage(error.message) ?? error.message ?? fallback
  }
  if (error instanceof Error && error.message) {
    return matchKnownMessage(error.message) ?? error.message
  }
  return fallback
}
