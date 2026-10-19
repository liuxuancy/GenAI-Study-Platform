// 权限码匹配，规则和后端 access.Match 完全一致，两边共用 docs/permission-cases.json 里的用例。
// 前端判断只用来决定显示哪些菜单和按钮，真正的拦截在后端。
import type { CurrentUser } from '@/core/api/account'

const codeSeparator = ':'
const wildcard = '*'

// matchPermission 判断授予的权限码 granted 是否满足要求的权限码 required。
export function matchPermission(granted: string, required: string): boolean {
  const grantedSegments = parseCode(granted)
  const requiredSegments = parseCode(required)
  if (!grantedSegments || !requiredSegments) {
    return false
  }

  for (const [index, segment] of grantedSegments.entries()) {
    if (segment === wildcard) {
      return requiredSegments.length > index // '*' 至少要匹配一段
    }
    if (index >= requiredSegments.length || requiredSegments[index] !== segment) {
      return false
    }
  }
  return grantedSegments.length === requiredSegments.length
}

// isValidPermissionCode 判断权限码格式是否合法。
export function isValidPermissionCode(code: string): boolean {
  return parseCode(code) !== null
}

// hasPermission 判断当前用户是否拥有某个权限码；管理员全部放行。
export function hasPermission(user: CurrentUser | null, required: string): boolean {
  if (!user) {
    return false
  }
  if (user.isAdmin) {
    return true
  }
  return user.permissions.some((granted) => matchPermission(granted, required))
}

// parseCode 按 ':' 切分；任一段为空，或 '*' 不是完整的最后一段，都视为非法，返回 null。
function parseCode(code: string): string[] | null {
  if (code === '') {
    return null
  }
  const segments = code.split(codeSeparator)
  const lastIndex = segments.length - 1
  for (const [index, segment] of segments.entries()) {
    if (segment === '') {
      return null
    }
    if (segment.includes(wildcard) && (segment !== wildcard || index !== lastIndex)) {
      return null
    }
  }
  return segments
}
