import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

import { getStudentCourses, type StudentCourse } from '@/modules/student/api'
import { AiWorkspace } from '@/modules/workspace/AiWorkspace'

import styles from '@/modules/student/learning/StudentAiPage.module.css'

export function StudentAiPage() {
  const [params] = useSearchParams()
  const [courses, setCourses] = useState<StudentCourse[]>([])

  useEffect(() => {
    getStudentCourses()
      .then(setCourses)
      .catch(() => setCourses([]))
  }, [])

  return (
    <div className={styles.root}>
      <AiWorkspace
        role="student"
        courses={courses.map((course) => ({ id: course.id, title: course.title }))}
        initialConversationId={Number(params.get('conversation')) || null}
        initialCourseId={Number(params.get('courseId')) || null}
      />
    </div>
  )
}
