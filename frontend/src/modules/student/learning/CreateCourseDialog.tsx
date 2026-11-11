import { useEffect, useRef } from 'react'

import { useAuthStore } from '@/core/auth/store'
import type { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import { useI18n } from '@/core/i18n/useI18n'
import homeStyles from '@/modules/student/StudentHomePage.module.css'

const structuredCourseType = 'Structured'

interface Props {
  open: boolean
  onClose: () => void
  title: string
  onTitleChange: (value: string) => void
  description: string
  onDescriptionChange: (value: string) => void
  nameError: string
  createError: string
  creating: boolean
  selection: ReturnType<typeof useAiProviderSelect>
  onSubmit: () => void
}

export function CreateCourseDialog({ open, onClose, title, onTitleChange, description, onDescriptionChange, nameError, createError, creating, selection, onSubmit }: Props) {
  const { t } = useI18n()
  const major = useAuthStore((state) => state.currentUser?.major ?? '')
  const dialogRef = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) {
      return
    }
    if (open && !dialog.open) {
      dialog.showModal()
    }
    if (!open && dialog.open) {
      dialog.close()
    }
  }, [open])

  return (
    <dialog ref={dialogRef} className={homeStyles.dialog} onCancel={(event) => {
      event.preventDefault()
      onClose()
    }}>
      <form
        className={homeStyles.dialogBody}
        onSubmit={(event) => {
          event.preventDefault()
          onSubmit()
        }}
      >
        <h2 className={homeStyles.dialogTitle}>{t('student.createCourse.title')}</h2>
        <label className={homeStyles.field}>
          {t('student.createCourse.nameLabel')}
          <input value={title} disabled={creating} onChange={(event) => onTitleChange(event.target.value)} />
        </label>
        {nameError ? <p className={homeStyles.error}>{nameError}</p> : null}
        <label className={homeStyles.field}>
          {t('student.createCourse.descLabel')}
          <input value={description} disabled={creating} onChange={(event) => onDescriptionChange(event.target.value)} />
        </label>
        {createError ? <p className={homeStyles.error}>{createError}</p> : null}
        <label className={homeStyles.field}>
          {t('student.createCourse.majorLabel')}
          <input value={major} disabled />
        </label>
        <label className={homeStyles.field}>
          {t('student.createCourse.typeLabel')}
          <select defaultValue={structuredCourseType}>
            <option value={structuredCourseType}>{t('student.createCourse.typeStructured')}</option>
          </select>
        </label>
        <label className={homeStyles.field}>
          {t('student.createCourse.avatarProviderLabel')}
          <select
            value={selection.provider}
            disabled={creating || selection.loading || !selection.optionsAvailable}
            onChange={(event) => {
              const option = selection.options.find((item) => item.value === event.target.value && !item.disabled)
              if (option) selection.setProvider(option.value)
            }}
          >
            {selection.options.map((option) => (
              <option key={option.value} value={option.value} disabled={option.disabled}>{option.label}</option>
            ))}
          </select>
        </label>
        {selection.loading ? <p role="status">{t('common.loading')}</p> : null}
        {selection.error ? <p className={homeStyles.error} role="alert">{selection.error}</p> : null}
        {!selection.loading && !selection.error && !selection.optionsAvailable
          ? <p className={homeStyles.error} role="alert">{t('ai.providerNotConfigured')}</p> : null}
        <div className={homeStyles.dialogActions}>
          <button className={homeStyles.cancel} type="button" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button className={homeStyles.create} type="submit" disabled={creating || selection.loading || !selection.available[selection.provider]}>
            {creating ? t('student.createCourse.creating') : t('student.createCourse.create')}
          </button>
        </div>
      </form>
    </dialog>
  )
}
