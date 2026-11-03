import type { MessageTree } from '@/core/i18n/types'

export const zhAccount: MessageTree = {
  account: {
    title: '我的账号',
    roleHint: '{{role}}。角色与用户名不可修改。',
    profileSaved: '资料已保存。',
    saveProfileFailed: '无法保存资料。',
    username: '用户名',
    nickname: '昵称',
    nicknameTooLong: '昵称过长。',
    email: '邮箱',
    emailInvalid: '请输入有效邮箱。',
    emailTooLong: '邮箱过长。',
    major: '专业',
    saveProfile: '保存资料',
    passwordSection: '密码',
    passwordHint: '修改后需要重新登录。',
    currentPassword: '当前密码',
    currentPasswordRequired: '请输入当前密码。',
    newPassword: '新密码',
    newPasswordRequired: '请输入新密码。',
    passwordLength: '请使用 6 至 72 个字符。',
    changePassword: '修改密码',
    changePasswordFailed: '无法修改密码。',
  },
}
