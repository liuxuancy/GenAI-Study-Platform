import type { MessageTree } from '@/core/i18n/types'

export const enAccount: MessageTree = {
  account: {
    title: 'Account',
    roleHint: '{{role}}. Role and username cannot be changed.',
    profileSaved: 'Profile saved.',
    saveProfileFailed: 'Could not save your profile.',
    username: 'Username',
    nickname: 'Nickname',
    nicknameTooLong: 'Nickname is too long.',
    email: 'Email',
    emailInvalid: 'Enter a valid email.',
    emailTooLong: 'Email is too long.',
    major: 'Major',
    saveProfile: 'Save profile',
    passwordSection: 'Password',
    passwordHint: 'After this change you need to sign in again.',
    currentPassword: 'Current password',
    currentPasswordRequired: 'Enter your current password.',
    newPassword: 'New password',
    newPasswordRequired: 'Enter a new password.',
    passwordLength: 'Use 6 to 72 characters.',
    changePassword: 'Change password',
    changePasswordFailed: 'Could not change your password.',
  },
}
