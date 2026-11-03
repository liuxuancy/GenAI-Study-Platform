import { ApiError } from '@/core/http'
import { getAppLocale } from '@/core/i18n/store'
import { translateApp } from '@/core/i18n/translateApp'

const MESSAGE_KEYS: Record<string, string> = {
  'file is required': 'courseResources.fileRequired',
  'file is empty': 'courseResources.fileEmpty',
  'filename must include an allowed extension': 'courseResources.extensionRequired',
  'file type is not allowed': 'courseResources.typeNotAllowed',
  'file exceeds size limit': 'courseResources.sizeLimit',
  'file content does not match extension': 'courseResources.contentMismatch',
  'object storage is not configured': 'courseResources.storageNotConfigured',
  'failed to read upload': 'courseResources.readUploadFailed',
  'permission denied': 'courseResources.permissionDenied',
  'access denied': 'courseResources.accessDenied',
  'resource not found': 'courseResources.notFound',
  'not signed in or session expired': 'courseResources.sessionExpired',
  'internal server error': 'courseResources.serverError',
  'network error: is the backend running?': 'http.networkError',
}

function t(key: string) {
  return translateApp(getAppLocale(), key)
}

function matchKnownMessage(raw: string): string | undefined {
  const key = raw.trim().toLowerCase()
  const messageKey = MESSAGE_KEYS[key]
  if (messageKey) {
    return t(messageKey)
  }
  if (key.includes('size limit')) {
    return t('courseResources.sizeLimit')
  }
  if (key.includes('not allowed') && key.includes('extension')) {
    return t('courseResources.typeNotAllowed')
  }
  if (key.includes('chapterkey not found') || key.includes('sectionkey not found')) {
    return t('courseResources.chapterNotFound')
  }
  return undefined
}

/** 将课程资料相关 API 错误转为用户可读文案。 */
export function mapCourseResourceError(error: unknown, fallback?: string): string {
  const fallbackMessage = fallback ?? t('courseResources.fallback')
  if (error instanceof ApiError) {
    if (error.code === 0) {
      return t('http.networkError')
    }
    if (error.code === 401) {
      return t('courseResources.sessionExpired')
    }
    if (error.code === 403) {
      return matchKnownMessage(error.message) ?? t('courseResources.accessDenied')
    }
    if (error.code === 404) {
      return t('courseResources.notFound')
    }
    if (error.code === 500) {
      return t('courseResources.serverError')
    }
    return matchKnownMessage(error.message) ?? error.message ?? fallbackMessage
  }
  if (error instanceof Error && error.message) {
    return matchKnownMessage(error.message) ?? error.message
  }
  return fallbackMessage
}
