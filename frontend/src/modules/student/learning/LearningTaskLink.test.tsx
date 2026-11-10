import { renderToStaticMarkup } from 'react-dom/server'
import { StaticRouter } from 'react-router-dom/server'
import { describe, expect, it, vi } from 'vitest'

import type { LearningDashboardTask } from '@/modules/student/api'
import { LearningTaskLink } from './LearningTaskLink'
import { learningTaskDestination } from './learningTaskDestination'

vi.mock('@/core/i18n/useI18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('LearningTaskLink', () => {
  it.each([
    ['course-12', '学习', '/student/courses/12'],
    ['quiz-23', '测验', '/student/questions/23'],
    ['review-34', '复习', '/student/questions/34'],
  ] as const)('renders %s as a link to its page', (id, type, path) => {
    const task: LearningDashboardTask = { id, type, title: 'Test', course: 'Course', completed: false }
    const destination = learningTaskDestination(id)
    if (!destination) throw new Error('Invalid test task')
    const html = renderToStaticMarkup(<StaticRouter location="/student"><LearningTaskLink task={task} destination={destination} /></StaticRouter>)
    expect(html).toContain(`href="${path}"`)
    expect(html).toContain('student.dashboard.taskIncomplete')
    expect(html).toMatch(/<span[^>]*role="img"[^>]*aria-label="student\.dashboard\.taskIncomplete"[^>]*><\/span><a /)
    expect(html).not.toContain('<input')
  })

  it('keeps the completed status indicator outside the navigation link', () => {
    const task: LearningDashboardTask = { id: 'course-12', type: '学习', title: 'Test', course: 'Course', completed: true }
    const html = renderToStaticMarkup(<StaticRouter location="/student"><LearningTaskLink task={task} destination="/student/courses/12" /></StaticRouter>)
    expect(html).toMatch(/<span[^>]*role="img"[^>]*aria-label="student\.dashboard\.taskCompleted"[^>]*>✓<\/span><a /)
    expect(html).toContain('href="/student/courses/12"')
  })
})
