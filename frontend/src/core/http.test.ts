import { afterEach, expect, it, vi } from 'vitest'

import { http } from './http'

afterEach(() => vi.unstubAllGlobals())

it('passes the cancellation signal to fetch and preserves the abort error', async () => {
  const controller = new AbortController()
  const abortError = new DOMException('Request cancelled', 'AbortError')
  const fetchMock = vi.fn((_url: string, init: RequestInit) => new Promise<Response>((_resolve, reject) => {
    init.signal?.addEventListener('abort', () => reject(abortError))
  }))
  vi.stubGlobal('fetch', fetchMock)
  const request = http.post('/student/courses', { title: 'Course' }, controller.signal)
  expect(fetchMock.mock.calls[0][1].signal).toBe(controller.signal)
  controller.abort()
  await expect(request).rejects.toBe(abortError)
})
