import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { getTeacherCourses, type TeacherCourse } from '@/modules/teacher/api'
import { AiWorkspace } from '@/modules/workspace/AiWorkspace'

import styles from '@/modules/teacher/TeacherAiPage.module.css'

export function TeacherAiPage() {
  const [params] = useSearchParams()
  const [courses, setCourses] = useState<TeacherCourse[]>([])

  useEffect(() => {
    getTeacherCourses()
      .then(setCourses)
      .catch(() => setCourses([]))
  }, [])

  return (
    <div className={styles.root}>
      <AiWorkspace
        role="teacher"
        courses={courses.map((course) => ({ id: course.id, title: course.title }))}
        initialConversationId={Number(params.get('conversation')) || null}
        initialCourseId={Number(params.get('courseId')) || null}
      />
    </div>
  )
}
