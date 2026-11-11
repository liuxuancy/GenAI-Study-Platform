export function learningTaskDestination(id: string): string | null {
  const match = /^(course|quiz|review)-([1-9]\d*)$/.exec(id)
  if (!match) {
    return null
  }
  const [, kind, resourceId] = match
  return kind === 'course'
    ? `/student/courses/${resourceId}`
    : `/student/questions/${resourceId}`
}
