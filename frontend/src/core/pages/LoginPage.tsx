// 登录页：样式照搬设计稿 login.html。登录成功后跳回原来要去的页面，没有就去首页。
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import styles from '@/core/pages/LoginPage.module.css'

type AudienceKey = 'student' | 'teacher'

// 学生 / 讲师·管理员切换只改说明文字。实际角色由账号决定，登录时不需要选。
const audienceLabels: Record<AudienceKey, string> = {
  student: 'Student',
  teacher: 'Teacher / Admin',
}

const audienceNotes: Record<AudienceKey, string> = {
  student: 'Student view: the course map, chapter list and AI explanations.',
  teacher: 'Teacher view: chapter drafts and explanation editing. Admins also manage users and role permissions.',
}

const toastDurationMs = 3400

export function LoginPage() {
  const token = useAuthStore((state) => state.token)
  const login = useAuthStore((state) => state.login)
  const navigate = useNavigate()
  const location = useLocation()

  const [audience, setAudience] = useState<AudienceKey>('student')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [passwordVisible, setPasswordVisible] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVisible, setToastVisible] = useState(false)
  const toastTimer = useRef<number | undefined>(undefined)

  useEffect(() => () => window.clearTimeout(toastTimer.current), [])

  // 已经登录的人打开登录页，直接进系统
  if (token && !submitting) {
    return <Navigate to="/" replace />
  }

  function showToast(message: string) {
    setToastMessage(message)
    setToastVisible(true)
    window.clearTimeout(toastTimer.current)
    toastTimer.current = window.setTimeout(() => setToastVisible(false), toastDurationMs)
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (submitting) {
      return
    }
    if (username.trim() === '' || password === '') {
      showToast('Please enter your username and password.')
      return
    }
    setSubmitting(true)
    try {
      await login({ username: username.trim(), password }, remember)
      const redirectPath = (location.state as { from?: string } | null)?.from ?? '/'
      navigate(redirectPath, { replace: true })
    } catch (error) {
      showToast((error as Error).message)
      setSubmitting(false)
    }
  }

  return (
    <div className={styles.page}>
      <header className={styles.topbar}>
        <div className={styles.brand}>
          <span className={styles.mark} aria-hidden="true">
            C
          </span>
          <span>
            <strong className={styles.brandName}>CaveClass</strong>
            <small className={styles.brandTagline}>Courses, explained by AI</small>
          </span>
        </div>
        <p className={styles.topNote}>AI course companion · Beta</p>
      </header>

      <main className={styles.stage}>
        <section className={styles.intro}>
          <p className={styles.eyebrow}>How a course reads</p>
          <h1>Every lecture, explained in order.</h1>
          <p className={styles.lede}>
            CaveClass rewrites each course as a short walkthrough — in the order you actually need it.
          </p>

          <ol className={styles.points}>
            <li>
              <span className={styles.pointNumber}>01</span>
              <div>
                <h3>Read it once, in order</h3>
                <p>Each lecture becomes a written walkthrough — no slide deck, no skipping.</p>
              </div>
            </li>
            <li>
              <span className={styles.pointNumber}>02</span>
              <div>
                <h3>Ask on the line you are stuck on</h3>
                <p>Highlight any sentence and ask. The answer stays attached to that line.</p>
              </div>
            </li>
            <li>
              <span className={styles.pointNumber}>03</span>
              <div>
                <h3>Keep your place</h3>
                <p>Your progress is marked chapter by chapter, so you always know the next step.</p>
              </div>
            </li>
          </ol>

          <ul className={styles.specs}>
            <li>12-week courses</li>
            <li>One chapter a week</li>
            <li>AI-guided</li>
          </ul>
        </section>

        <section className={styles.panelWrap}>
          <form className={styles.panel} onSubmit={handleSubmit} noValidate>
            <p className={styles.eyebrow}>Account</p>
            <h2>Sign in</h2>
            <p className={styles.panelSub}>Use the CaveClass account issued by your administrator.</p>

            <div className={styles.segment} role="tablist" aria-label="Choose your view">
              {(['student', 'teacher'] as const).map((key) => (
                <button
                  key={key}
                  type="button"
                  role="tab"
                  aria-selected={audience === key}
                  className={audience === key ? styles.segmentActive : undefined}
                  onClick={() => setAudience(key)}
                >
                  {audienceLabels[key]}
                </button>
              ))}
            </div>
            <p className={styles.roleNote}>{audienceNotes[audience]}</p>

            <label className={styles.field}>
              <span className={styles.fieldLabel}>Username</span>
              <input
                type="text"
                name="username"
                placeholder="e.g. student1"
                autoComplete="username"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
              />
            </label>

            <label className={styles.field}>
              <span className={styles.fieldLabel}>Password</span>
              <input
                type={passwordVisible ? 'text' : 'password'}
                name="password"
                placeholder="Your password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
              <button
                className={styles.peek}
                type="button"
                aria-label={passwordVisible ? 'Hide password' : 'Show password'}
                onClick={() => setPasswordVisible(!passwordVisible)}
              >
                {passwordVisible ? 'Hide' : 'Show'}
              </button>
            </label>

            <div className={styles.row}>
              <label className={styles.check}>
                <input type="checkbox" checked={remember} onChange={(event) => setRemember(event.target.checked)} />
                <span className={styles.checkBox}>
                  <svg viewBox="0 0 12 12" aria-hidden="true">
                    <path d="M2 6.4 L 4.6 9 L 10 3" />
                  </svg>
                </span>
                <span>Remember me</span>
              </label>
            </div>

            <button className={styles.submit} type="submit" aria-busy={submitting}>
              {submitting ? 'Signing in…' : 'Sign in'}
            </button>
          </form>
        </section>
      </main>

      <footer className={styles.foot}>
        <span>© 2026 CaveClass · A GenAI course learning platform</span>
      </footer>

      <div className={`${styles.toast} ${toastVisible ? styles.toastVisible : ''}`} role="status" aria-live="polite">
        {toastMessage}
      </div>
    </div>
  )
}
