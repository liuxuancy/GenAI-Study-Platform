import { renderToStaticMarkup } from 'react-dom/server'
import { StaticRouter } from 'react-router-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { CourseBreadcrumb } from './CourseBreadcrumb'

vi.mock('@/core/i18n/useI18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('CourseBreadcrumb', () => {
  it('links to home and the course list before the current course', () => {
    const html = renderToStaticMarkup(
      <StaticRouter location="/student/courses/42"><CourseBreadcrumb title="Current course" /></StaticRouter>,
    )
    expect(html).toMatch(/<nav[^>]*aria-label="student\.bubble\.breadcrumbAria"/)
    expect(html).toMatch(/href="\/student">student\.bubble\.home<\/a>.*href="\/student\/courses">nav\.student\.myCourses<\/a>.*aria-current="page"[^>]*>Current course<\/span>/)
    expect(html.match(/aria-hidden="true"/g)).toHaveLength(2)
  })
})
