import { message } from 'antd'
import { createContext, useCallback, useContext, useEffect, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { useAiProviderSelect } from '@/core/ai/useAiProviderSelect'
import { useI18n } from '@/core/i18n/useI18n'
import { createStudentCourse } from '@/modules/student/api'
import { CreateCourseDialog } from './CreateCourseDialog'

interface CourseCreationContextValue {
  openDialog: () => void
  revision: number
  pendingTitle: string | null
}

const CourseCreationContext = createContext<CourseCreationContextValue | null>(null)

export function useCourseCreation() {
  const context = useContext(CourseCreationContext)
  if (!context) throw new Error('CourseCreationProvider is missing')
  return context
}

export function CourseCreationProvider({ children }: { children: ReactNode }) {
  const userId = useAuthStore((state) => state.currentUser?.id)
  return <CourseCreationSession key={userId} userId={userId}>{children}</CourseCreationSession>
}

function CourseCreationSession({ children, userId }: { children: ReactNode; userId: number | undefined }) {
  const { t } = useI18n()
  const selection = useAiProviderSelect({ courseAvatar: true })
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const [open, setOpen] = useState(false)
  const openRef = useRef(false)
  const pathRef = useRef(pathname)
  const active = useRef(false)
  const inFlight = useRef(false)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [nameError, setNameError] = useState('')
  const [createError, setCreateError] = useState('')
  const [creating, setCreating] = useState(false)
  const [pendingTitle, setPendingTitle] = useState<string | null>(null)
  const [revision, setRevision] = useState(0)

  useEffect(() => {
    active.current = true
    return () => { active.current = false }
  }, [])

  useLayoutEffect(() => {
    if (pathRef.current !== pathname) {
      pathRef.current = pathname
      openRef.current = false
      setOpen(false)
    }
  }, [pathname])

  const openDialog = useCallback(() => {
    openRef.current = true
    setOpen(true)
  }, [])

  function closeDialog() {
    openRef.current = false
    setOpen(false)
  }

  async function submit() {
    if (inFlight.current || selection.loading || !selection.available[selection.provider]) return
    const trimmedTitle = title.trim()
    if (!trimmedTitle) {
      setNameError(t('student.createCourse.nameRequired'))
      return
    }
    inFlight.current = true
    setCreating(true)
    setPendingTitle(trimmedTitle)
    setNameError('')
    setCreateError('')
    try {
      const created = await createStudentCourse({ title: trimmedTitle, description: description.trim(), provider: selection.provider })
      if (!active.current || useAuthStore.getState().currentUser?.id !== userId) return
      setTitle('')
      setDescription('')
      setRevision((value) => value + 1)
      if (openRef.current) {
        closeDialog()
        navigate(`/student/courses/${created.id}`)
      }
    } catch (error) {
      if (!active.current || useAuthStore.getState().currentUser?.id !== userId) return
      const reason = error instanceof Error ? error.message : t('student.createCourse.failed')
      setCreateError(reason)
      setRevision((value) => value + 1)
      void message.error(`${t('student.createCourse.failed')}: ${reason}`)
    } finally {
      inFlight.current = false
      if (active.current && useAuthStore.getState().currentUser?.id === userId) {
        setPendingTitle(null)
        setCreating(false)
      }
    }
  }

  return (
    <CourseCreationContext.Provider value={{ openDialog, revision, pendingTitle }}>
      {children}
      <CreateCourseDialog
        open={open}
        onClose={closeDialog}
        title={title}
        onTitleChange={setTitle}
        description={description}
        onDescriptionChange={setDescription}
        nameError={nameError}
        createError={createError}
        creating={creating}
        selection={selection}
        onSubmit={() => void submit()}
      />
    </CourseCreationContext.Provider>
  )
}
