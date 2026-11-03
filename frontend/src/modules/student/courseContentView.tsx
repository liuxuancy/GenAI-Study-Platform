import { useMemo } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import markdownStyles from '@/modules/student/courseMarkdown.module.css'

export function splitNoteBlocks(note: string) {
  const parts = note.split(/```(?:[\w-]*\n)?/)
  if (parts.length === 1) {
    return [{ type: 'text' as const, value: note }]
  }
  const blocks: Array<{ type: 'text' | 'code'; value: string }> = []
  for (let index = 0; index < parts.length; index += 1) {
    const chunk = parts[index]
    if (!chunk) {
      continue
    }
    blocks.push({
      type: index % 2 === 1 ? 'code' : 'text',
      value: chunk.trim(),
    })
  }
  return blocks
}

export function CourseMarkdownBody({
  source,
  codeClassName,
}: {
  source: string
  codeClassName?: string
}) {
  if (!source.trim()) {
    return null
  }
  return (
    <>
      {splitNoteBlocks(source).map((block, index) =>
        block.type === 'code' ? (
          <pre key={`code-${index}`} className={codeClassName}>
            <code>{block.value}</code>
          </pre>
        ) : (
          <div key={`md-${index}`} className={markdownStyles.prose}>
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{block.value}</ReactMarkdown>
          </div>
        ),
      )}
    </>
  )
}

export function activityLabel(activityType: string) {
  if (activityType === 'rich_text') {
    return '阅读'
  }
  if (activityType === 'coding') {
    return '编程练习'
  }
  if (activityType === 'mock_interview') {
    return '模拟面试'
  }
  if (activityType === 'quiz_game') {
    return '章末小测'
  }
  return activityType
}

export function chapterBadgeLabel(status: 'completed' | 'learning' | 'pending') {
  if (status === 'completed') {
    return '已完成'
  }
  if (status === 'learning') {
    return '学习中'
  }
  return ''
}

export function useActivityNeighbors(activityIds: string[], activeId: string | null) {
  return useMemo(() => {
    if (!activeId) {
      return { prev: null as string | null, next: null as string | null }
    }
    const index = activityIds.indexOf(activeId)
    if (index < 0) {
      return { prev: null, next: null }
    }
    return {
      prev: index > 0 ? activityIds[index - 1] : null,
      next: index < activityIds.length - 1 ? activityIds[index + 1] : null,
    }
  }, [activeId, activityIds])
}
