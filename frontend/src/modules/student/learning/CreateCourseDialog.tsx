import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { useI18n } from '@/core/i18n/useI18n'
import { createStudentCourse } from '@/modules/student/api'
import homeStyles from '@/modules/student/StudentHomePage.module.css'

const structuredCourseType = 'Structured'

export function CreateCourseDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const major = useAuthStore((state) => state.currentUser?.major ?? '')
  const dialogRef = useRef<HTMLDialogElement>(null)
  const [courseName, setCourseName] = useState('')
  const [description, setDescription] = useState('')
  const [nameError, setNameError] = useState('')
  const [createError, setCreateError] = useState('')
  const [creating, setCreating] = useState(false)

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

  async function createCourse() {
    const title = courseName.trim()
    if (!title) {
      setNameError(t('student.createCourse.nameRequired'))
      return
    }
    setNameError('')
    setCreating(true)
    setCreateError('')
    try {
      const created = await createStudentCourse({ title, description: description.trim() })
      onClose()
      navigate(`/student/courses/${created.id}`)
    } catch (error) {
      setCreateError(error instanceof Error ? error.message : t('student.createCourse.failed'))
    } finally {
      setCreating(false)
    }
  }

  return (
    <dialog ref={dialogRef} className={homeStyles.dialog} onClose={onClose}>
      <form
        className={homeStyles.dialogBody}
        onSubmit={(event) => {
          event.preventDefault()
          void createCourse()
        }}
      >
        <h2 className={homeStyles.dialogTitle}>{t('student.createCourse.title')}</h2>
        <label className={homeStyles.field}>
          {t('student.createCourse.nameLabel')}
          <input value={courseName} onChange={(event) => setCourseName(event.target.value)} />
        </label>
        {nameError ? <p className={homeStyles.error}>{nameError}</p> : null}
        <label className={homeStyles.field}>
          {t('student.createCourse.descLabel')}
          <input value={description} onChange={(event) => setDescription(event.target.value)} />
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
        <div className={homeStyles.dialogActions}>
          <button className={homeStyles.cancel} type="button" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button className={homeStyles.create} type="submit" disabled={creating}>
            {creating ? t('student.createCourse.creating') : t('student.createCourse.create')}
          </button>
        </div>
      </form>
    </dialog>
  )
}
