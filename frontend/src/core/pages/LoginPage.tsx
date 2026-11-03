// 登录页：样式对齐 docs/UI/login.html。须先选身份再登录，账号角色须与所选身份一致。
import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'

import { homePathFor } from '@/core/auth/homePath'
import {
  audienceMismatchMessage,
  roleMatchesAudience,
  type LoginAudience,
} from '@/core/auth/loginAudience'
import { pathAfterLogin } from '@/core/auth/postLoginPath'
import { useAuthStore } from '@/core/auth/store'
import { LanguageSwitcher } from '@/core/i18n/LanguageSwitcher'
import { useI18n } from '@/core/i18n/useI18n'
import styles from '@/core/pages/LoginPage.module.css'

const audienceKeys: Record<LoginAudience, { label: string; note: string }> = {
  student: { label: 'login.form.audienceStudent', note: 'login.form.noteStudent' },
  teacher: { label: 'login.form.audienceTeacher', note: 'login.form.noteTeacher' },
  admin: { label: 'login.form.audienceAdmin', note: 'login.form.noteAdmin' },
}

const toastDurationMs = 3400

export function LoginPage() {
  const { t } = useI18n()
  const token = useAuthStore((state) => state.token)
  const currentUser = useAuthStore((state) => state.currentUser)
  const login = useAuthStore((state) => state.login)
  const loadCurrentUser = useAuthStore((state) => state.loadCurrentUser)
  const clearSession = useAuthStore((state) => state.clearSession)
  const navigate = useNavigate()
  const location = useLocation()

  const [audience, setAudience] = useState<LoginAudience>('student')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [remember, setRemember] = useState(true)
  const [passwordVisible, setPasswordVisible] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [toastMessage, setToastMessage] = useState('')
  const [toastVisible, setToastVisible] = useState(false)
  const toastTimer = useRef<number | undefined>(undefined)

  useEffect(() => () => window.clearTimeout(toastTimer.current), [])

  useEffect(() => {
    if (!token || currentUser || submitting) {
      return
    }
    void loadCurrentUser().catch(() => {
      clearSession()
    })
  }, [token, currentUser, submitting, loadCurrentUser, clearSession])

  if (token && !submitting && currentUser) {
    return <Navigate to={homePathFor(currentUser.role)} replace />
  }

  if (token && !submitting && !currentUser) {
    return null
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
      showToast(t('login.error.credentialsRequired'))
      return
    }
    setSubmitting(true)
    try {
      await login({ username: username.trim(), password }, remember)
      const signedInUser = useAuthStore.getState().currentUser
      if (!signedInUser) {
        clearSession()
        showToast(t('login.error.profileLoadFailed'))
        setSubmitting(false)
        return
      }
      if (!roleMatchesAudience(signedInUser.role, audience)) {
        clearSession()
        showToast(audienceMismatchMessage(audience))
        setSubmitting(false)
        return
      }
      const from = (location.state as { from?: string } | null)?.from
      const redirectPath = pathAfterLogin(signedInUser.role, from)
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
            <strong className={styles.brandName}>{t('common.brandName')}</strong>
            <small className={styles.brandTagline}>{t('login.brandTagline')}</small>
          </span>
        </div>
        <div className={styles.topbarRight}>
          <p className={styles.topNote}>{t('login.topNote')}</p>
          <LanguageSwitcher compact />
        </div>
      </header>

      <main className={styles.stage}>
        <section className={styles.intro}>
          <p className={styles.eyebrow}>{t('login.intro.eyebrow')}</p>
          <h1>{t('login.intro.title')}</h1>
          <p className={styles.lede}>{t('login.intro.lede')}</p>

          <ol className={styles.points}>
            <li>
              <span className={styles.pointNumber}>01</span>
              <div>
                <h3>{t('login.intro.point1Title')}</h3>
                <p>{t('login.intro.point1Body')}</p>
              </div>
            </li>
            <li>
              <span className={styles.pointNumber}>02</span>
              <div>
                <h3>{t('login.intro.point2Title')}</h3>
                <p>{t('login.intro.point2Body')}</p>
              </div>
            </li>
            <li>
              <span className={styles.pointNumber}>03</span>
              <div>
                <h3>{t('login.intro.point3Title')}</h3>
                <p>{t('login.intro.point3Body')}</p>
              </div>
            </li>
          </ol>

          <ul className={styles.specs}>
            <li>{t('login.intro.spec1')}</li>
            <li>{t('login.intro.spec2')}</li>
            <li>{t('login.intro.spec3')}</li>
          </ul>
        </section>

        <section className={styles.panelWrap}>
          <form className={styles.panel} onSubmit={handleSubmit} noValidate>
            <p className={styles.eyebrow}>{t('login.form.eyebrow')}</p>
            <h2>{t('login.form.signIn')}</h2>
            <p className={styles.panelSub}>{t('login.form.panelSub')}</p>

            <div className={`${styles.segment} ${styles.segmentThree}`} role="tablist" aria-label={t('login.form.roleTabsAria')}>
              {(['student', 'teacher', 'admin'] as const).map((key) => (
                <button
                  key={key}
                  type="button"
                  role="tab"
                  aria-selected={audience === key}
                  className={audience === key ? styles.segmentActive : undefined}
                  onClick={() => setAudience(key)}
                >
                  {t(audienceKeys[key].label)}
                </button>
              ))}
            </div>
            <p className={styles.roleNote}>{t(audienceKeys[audience].note)}</p>

            <label className={styles.field}>
              <span className={styles.fieldLabel}>{t('login.form.usernameLabel')}</span>
              <input
                type="text"
                name="username"
                placeholder={t('login.form.usernamePlaceholder')}
                autoComplete="username"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
              />
            </label>

            <label className={styles.field}>
              <span className={styles.fieldLabel}>{t('login.form.passwordLabel')}</span>
              <input
                type={passwordVisible ? 'text' : 'password'}
                name="password"
                placeholder={t('login.form.passwordPlaceholder')}
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
              <button
                className={styles.peek}
                type="button"
                aria-label={passwordVisible ? t('login.form.hidePassword') : t('login.form.showPassword')}
                onClick={() => setPasswordVisible(!passwordVisible)}
              >
                {passwordVisible ? t('login.form.hide') : t('login.form.show')}
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
                <span>{t('login.form.rememberMe')}</span>
              </label>
              <a className={styles.tlink} href="#" onClick={(event) => event.preventDefault()}>
                {t('login.form.forgotPassword')}
              </a>
            </div>

            <button className={styles.submit} type="submit" aria-busy={submitting}>
              {submitting ? t('login.form.signingIn') : t('login.form.signIn')}
            </button>

            <hr className={styles.panelRule} />

            <p className={styles.panelFoot}>
              {audience === 'student' ? (
                <>
                  {t('login.form.footStudentLead')}{' '}
                  <a className={styles.tlink} href="#" onClick={(event) => event.preventDefault()}>
                    {t('login.form.createAccount')}
                  </a>
                </>
              ) : audience === 'teacher' ? (
                <>
                  {t('login.form.footTeacherLead')}{' '}
                  <a className={styles.tlink} href="#" onClick={(event) => event.preventDefault()}>
                    {t('login.form.requestAccess')}
                  </a>
                </>
              ) : (
                <>
                  {t('login.form.footAdminLead')}{' '}
                  <a className={styles.tlink} href="#" onClick={(event) => event.preventDefault()}>
                    {t('login.form.requestAccess')}
                  </a>
                </>
              )}
            </p>
          </form>

          <p className={styles.legal}>
            {t('login.legal.prefix')}{' '}
            <a href="#" onClick={(event) => event.preventDefault()}>
              {t('login.legal.terms')}
            </a>{' '}
            {t('login.legal.and')}{' '}
            <a href="#" onClick={(event) => event.preventDefault()}>
              {t('login.legal.privacy')}
            </a>
            .
          </p>
        </section>
      </main>

      <footer className={styles.foot}>
        <span>{t('login.footer.copyright')}</span>
        <span className={styles.footLinks}>
          <a href="#" onClick={(event) => event.preventDefault()}>
            {t('login.footer.status')}
          </a>
          <a href="#" onClick={(event) => event.preventDefault()}>
            {t('login.footer.help')}
          </a>
          <a href="#" onClick={(event) => event.preventDefault()}>
            {t('login.footer.contact')}
          </a>
        </span>
      </footer>

      <div className={`${styles.toast} ${toastVisible ? styles.toastVisible : ''}`} role="status" aria-live="polite">
        {toastMessage}
      </div>
    </div>
  )
}
