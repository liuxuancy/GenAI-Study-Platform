import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it, vi } from 'vitest'

import { CourseCover } from './studentCourseVisuals'

vi.mock('@/core/i18n/store', () => ({ getAppLocale: () => 'en' }))
vi.mock('@/core/i18n/translateApp', () => ({
  translateApp: (_locale: string, key: string) => key,
}))

describe('course cover', () => {
  it('hides both labels when displaying a pixel character', () => {
    const html = renderToStaticMarkup(
      <CourseCover title="Git" sprite={{ palette: ['#000000'], pixels: ['0'] }} />,
    )
    expect(html).toContain('<rect')
    expect(html).not.toContain('DEVELOPER TOOLS')
    expect(html).not.toContain('student.coverTopics.git')
  })

  it('retains both labels and the original artwork without a pixel character', () => {
    const html = renderToStaticMarkup(<CourseCover title="Git" sprite={null} />)
    expect(html).toContain('DEVELOPER TOOLS')
    expect(html).toContain('student.coverTopics.git')
    expect(html).toContain('<svg')
  })

  it('adds visible hearts to course cover art only at maximum affection', () => {
    const sprite = { palette: ['#000000'], pixels: ['0'] }
    const loved = renderToStaticMarkup(<CourseCover title="Git" sprite={sprite} affection={100} />)
    const notMax = renderToStaticMarkup(<CourseCover title="Git" sprite={sprite} affection={99} />)
    expect(loved).toContain('data-affection-hearts="true"')
    expect(loved.match(/<path/g)).toHaveLength(2)
    expect(notMax).not.toContain('data-affection-hearts')
  })
})
