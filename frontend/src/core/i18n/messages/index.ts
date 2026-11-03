import { mergeMessages } from '@/core/i18n/messages/merge'
import { enAccount } from '@/core/i18n/messages/parts/en/account'
import { enAdmin } from '@/core/i18n/messages/parts/en/admin'
import { enAi } from '@/core/i18n/messages/parts/en/ai'
import { enCommon } from '@/core/i18n/messages/parts/en/common'
import { enLogin } from '@/core/i18n/messages/parts/en/login'
import { enNav } from '@/core/i18n/messages/parts/en/nav'
import { enResources } from '@/core/i18n/messages/parts/en/resources'
import { enStudent } from '@/core/i18n/messages/parts/en/student'
import { enTeacher } from '@/core/i18n/messages/parts/en/teacher'
import { enTeacherWorkspace } from '@/core/i18n/messages/parts/en/teacherWorkspace'
import { enWorkspace } from '@/core/i18n/messages/parts/en/workspace'
import { zhAccount } from '@/core/i18n/messages/parts/zh/account'
import { zhAdmin } from '@/core/i18n/messages/parts/zh/admin'
import { zhAi } from '@/core/i18n/messages/parts/zh/ai'
import { zhCommon } from '@/core/i18n/messages/parts/zh/common'
import { zhLogin } from '@/core/i18n/messages/parts/zh/login'
import { zhNav } from '@/core/i18n/messages/parts/zh/nav'
import { zhResources } from '@/core/i18n/messages/parts/zh/resources'
import { zhStudent } from '@/core/i18n/messages/parts/zh/student'
import { zhTeacher } from '@/core/i18n/messages/parts/zh/teacher'
import { zhTeacherWorkspace } from '@/core/i18n/messages/parts/zh/teacherWorkspace'
import { zhWorkspace } from '@/core/i18n/messages/parts/zh/workspace'
import type { MessageTree } from '@/core/i18n/types'

export const enMessages: MessageTree = mergeMessages(
  enCommon,
  enNav,
  enLogin,
  enAccount,
  enAdmin,
  enAi,
  enResources,
  enStudent,
  enTeacher,
  enTeacherWorkspace,
  enWorkspace,
)

export const zhMessages: MessageTree = mergeMessages(
  zhCommon,
  zhNav,
  zhLogin,
  zhAccount,
  zhAdmin,
  zhAi,
  zhResources,
  zhStudent,
  zhTeacher,
  zhTeacherWorkspace,
  zhWorkspace,
)
