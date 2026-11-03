import { useEffect, useState } from 'react'
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom'
import {
  CheckOutlined,
  LeftOutlined,
  RightOutlined,
  SendOutlined,
  SaveOutlined,
} from '@ant-design/icons'
import { Modal, message } from 'antd'

import { useI18n } from '@/core/i18n/useI18n'
import { useAuthStore } from '@/core/auth/store'
import {
  answerStudentQuestion,
  getStudentQuestion,
  type StudentQuestionContent,
  type StudentQuestionDetail,
} from '@/modules/student/api'
import { difficultyLabel } from '@/modules/student/quiz/mapStudentQuiz'
import styles from '@/modules/student/QuestionDetailPage.module.css'

export function QuestionDetailPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const { questionId } = useParams()
  const currentUser = useAuthStore((state) => state.currentUser)
  const [detail, setDetail] = useState<StudentQuestionDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [errorMessage, setErrorMessage] = useState('')
  const [index, setIndex] = useState(0)
  const [pending, setPending] = useState<Record<number, string>>({})
  const [saving, setSaving] = useState(false)
  const [marked, setMarked] = useState<Set<number>>(() => new Set())
  const [elapsedSeconds, setElapsedSeconds] = useState(0)

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_STUDENT' || !questionId) {
      return
    }
    let cancelled = false
    setIndex(0)
    setDetail(null)
    setPending({})
    setMarked(new Set())
    setElapsedSeconds(0)
    setErrorMessage('')
    setLoading(true)
    getStudentQuestion(questionId)
      .then((item) => {
        if (!cancelled) {
          setDetail(item)
        }
      })
      .catch((error: Error) => {
        if (!cancelled) {
          setErrorMessage(error.message)
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [questionId, currentUser?.role])

  useEffect(() => {
    if (loading || !detail || detail.isSubmit) {
      return
    }
    const timer = window.setInterval(() => {
      setElapsedSeconds((value) => value + 1)
    }, 1000)
    return () => window.clearInterval(timer)
  }, [detail, loading])

  if (!currentUser) {
    return null
  }
  if (currentUser.role !== 'ROLE_STUDENT') {
    return <Navigate to="/" replace />
  }

  const questions = detail?.questions ?? []
  const current = questions[index]
  const questionNumber = questions.length === 0 ? 0 : index + 1
  const selected = current ? chosenOption(current, pending) : ''
  const answered = detail ? answeredCount(questions, pending) : 0
  const label = detail ? questionTitleLabel(detail.title, detail.difficulty) : null
  const submitted = detail?.isSubmit ?? false
  const correctCount = questions.filter((question) => question.isCorrect).length

  function flushPending(submit: boolean): Promise<void> {
    if (!questionId || !detail || submitted || saving) {
      return Promise.resolve()
    }
    const toSave = unsavedAnswers(questions, pending)
    setSaving(true)
    setErrorMessage('')

    const runSaves = toSave.reduce(
      (chain, item) =>
        chain.then((snapshot) =>
          answerStudentQuestion(questionId, item.contentId, item.option, false).then((updated) => {
            setDetail(updated)
            setPending((currentPending) => dropPending(currentPending, item.contentId))
            return updated
          }),
        ),
      Promise.resolve(detail),
    )

    return runSaves
      .then(() => {
        if (!submit) {
          if (toSave.length > 0) {
            message.success(t('student.questionDetail.saveSuccess'))
          }
          return
        }
        if (!current) {
          return
        }
        const option = chosenOption(current, pending)
        if (option === '') {
          message.warning(t('student.questionDetail.selectOptionHint'))
          return
        }
        return answerStudentQuestion(questionId, current.id, option, true).then((submittedDetail) => {
          setDetail(submittedDetail)
          setPending((currentPending) => dropPending(currentPending, current.id))
          message.success(t('student.questionDetail.submitSuccess'))
        })
      })
      .catch((error: Error) => {
        setErrorMessage(error.message)
        message.error(error.message)
        throw error
      })
      .finally(() => {
        setSaving(false)
      })
  }

  function goToQuestion(nextIndex: number) {
    if (submitted || saving || !questionId || !current || nextIndex === index) {
      setIndex(nextIndex)
      return
    }
    const option = chosenOption(current, pending)
    if (option === '' || option === current.userOption) {
      setIndex(nextIndex)
      return
    }
    setSaving(true)
    setErrorMessage('')
    answerStudentQuestion(questionId, current.id, option, false)
      .then((updated) => {
        setDetail(updated)
        setPending((currentPending) => dropPending(currentPending, current.id))
        setIndex(nextIndex)
      })
      .catch((error: Error) => {
        setErrorMessage(error.message)
        message.error(error.message)
      })
      .finally(() => {
        setSaving(false)
      })
  }

  function confirmSubmitQuiz() {
    if (!detail || submitted || saving) {
      return
    }
    if (selected === '') {
      message.warning(t('student.questionDetail.selectOptionHint'))
      return
    }
    const remaining = Math.max(0, detail.questionCount - answered)
    Modal.confirm({
      title: t('student.questionDetail.submitConfirmTitle'),
      content:
        remaining > 0
          ? t('student.questionDetail.submitConfirmIncomplete', { remaining: String(remaining) })
          : t('student.questionDetail.submitConfirmAll'),
      okText: t('student.questionDetail.submitConfirmOk'),
      cancelText: t('student.questionDetail.submitConfirmCancel'),
      onOk: () => flushPending(true),
    })
  }

  function saveAndExit() {
    if (submitted) {
      navigate('/student/quizzes')
      return
    }
    void flushPending(false)
      .then(() => navigate('/student/quizzes'))
      .catch(() => {
        // 保留在当前页，便于用户重试
      })
  }

  function toggleMark() {
    if (!current) {
      return
    }
    setMarked((currentMarked) => {
      const next = new Set(currentMarked)
      if (next.has(current.id)) {
        next.delete(current.id)
      } else {
        next.add(current.id)
      }
      return next
    })
  }

  return (
    <div className={styles.shell}>
      <p className={styles.backLine}>
        <Link to="/student/quizzes">{t('student.questionDetail.back')}</Link>
      </p>
      {errorMessage ? <p className={styles.error}>{errorMessage}</p> : null}
      {loading ? <p className={styles.loading}>{t('student.questionDetail.loading')}</p> : null}
      {detail && current && label ? (
        <div className={styles.workspace}>
          <article className={styles.mainCard}>
            <div className={styles.cardTop}>
              <h1 className={styles.title}>{label.title}</h1>
              <span className={styles.questionCounter}>
                {t('student.questionDetail.questionProgress', {
                  current: questionNumber,
                  total: detail.questionCount,
                })}
              </span>
            </div>
            <div className={styles.metaRow}>
              <span className={styles.tag}>{detail.courseTitle}</span>
              {label.difficulty ? (
                <span className={`${styles.tag} ${styles.tagDifficulty}`}>{label.difficulty}</span>
              ) : null}
            </div>
            <div className={styles.segmentBar} aria-hidden="true">
              {questions.map((question) => {
                const done = chosenOption(question, pending) !== ''
                return (
                  <span
                    key={question.id}
                    className={`${styles.segment}${done ? ` ${styles.segmentDone}` : ''}`}
                  />
                )
              })}
            </div>

            <p className={styles.stem}>{current.content}</p>

            <ul className={styles.options} aria-label={t('student.questionDetail.optionsAria')}>
              {current.options.map((option, optionIndex) => {
                const letter = optionLetter(optionIndex)
                const isSelected = option === selected
                const isUserChoice = option === current.userOption
                const isCorrectOption = option === current.correctAnswer
                let optionClass = styles.option
                if (submitted) {
                  if (isCorrectOption) {
                    optionClass = `${styles.option} ${styles.optionCorrect}`
                  } else if (isUserChoice && !current.isCorrect) {
                    optionClass = `${styles.option} ${styles.optionWrong}`
                  }
                } else if (isSelected) {
                  optionClass = `${styles.option} ${styles.optionSelected}`
                }
                return (
                  <li key={`${current.id}-${optionIndex}`}>
                    <button
                      type="button"
                      className={optionClass}
                      aria-pressed={isSelected}
                      disabled={submitted}
                      onClick={() => {
                        setPending((currentPending) => ({ ...currentPending, [current.id]: option }))
                        setErrorMessage('')
                      }}
                    >
                      <span className={styles.optionBadge}>{letter}</span>
                      <span className={styles.optionText}>{option}</span>
                      {isSelected && !submitted ? (
                        <CheckOutlined className={styles.optionMark} aria-hidden="true" />
                      ) : null}
                    </button>
                  </li>
                )
              })}
            </ul>

            {submitted && current.explanation ? (
              <div className={styles.tipBox}>
                <h3>{t('student.questionDetail.knowledgeTip')}</h3>
                <p>{current.explanation}</p>
              </div>
            ) : null}

            <footer className={styles.cardFooter}>
              <button
                type="button"
                className={styles.navBtn}
                disabled={index <= 0 || saving}
                onClick={() => goToQuestion(Math.max(0, index - 1))}
              >
                <LeftOutlined /> {t('student.questionDetail.prevQuestion')}
              </button>
              <button
                type="button"
                className={`${styles.markBtn}${marked.has(current.id) ? ` ${styles.markBtnActive}` : ''}`}
                onClick={toggleMark}
              >
                {marked.has(current.id)
                  ? t('student.questionDetail.marked')
                  : t('student.questionDetail.markQuestion')}
              </button>
              <button
                type="button"
                className={styles.navBtn}
                disabled={index >= questions.length - 1 || saving}
                onClick={() => goToQuestion(Math.min(questions.length - 1, index + 1))}
              >
                {t('student.questionDetail.nextQuestion')} <RightOutlined />
              </button>
            </footer>
          </article>

          <aside className={styles.side}>
            {!submitted ? (
              <div className={styles.sideCard}>
                <p className={styles.sideCardLabel}>{t('student.questionDetail.timeElapsed')}</p>
                <p className={styles.timer}>{formatDuration(elapsedSeconds)}</p>
              </div>
            ) : null}

            {submitted && detail.score != null ? (
              <div className={styles.scoreBanner}>
                {t('student.questionDetail.scoreSummary', {
                  score: detail.score,
                  correct: correctCount,
                  total: detail.questionCount,
                })}
              </div>
            ) : null}

            <div className={styles.sideCard}>
              <div className={styles.progressHead}>
                <span>{t('student.questionDetail.progressPanelTitle')}</span>
                <span>
                  {answered}/{detail.questionCount}
                </span>
              </div>
              <div className={styles.progressGrid} role="list" aria-label={t('student.questionDetail.questionsAria')}>
                {questions.map((question, questionIndex) => {
                  const choice = chosenOption(question, pending)
                  const classes = [styles.gridCell]
                  if (questionIndex === index) {
                    classes.push(styles.gridCellActive)
                  }
                  if (submitted) {
                    if (question.isCorrect) {
                      classes.push(styles.gridCellCorrect)
                    } else if (choice !== '') {
                      classes.push(styles.gridCellWrong)
                    }
                  } else if (choice !== '') {
                    classes.push(styles.gridCellAnswered)
                  }
                  return (
                    <button
                      key={question.id}
                      type="button"
                      className={classes.join(' ')}
                      aria-pressed={questionIndex === index}
                      onClick={() => goToQuestion(questionIndex)}
                    >
                      {questionIndex + 1}
                    </button>
                  )
                })}
              </div>
              <div className={styles.legend}>
                <span className={styles.legendItem}>✓ {t('student.questionDetail.legendAnswered')}</span>
                <span className={styles.legendItem}>◎ {t('student.questionDetail.legendCurrent')}</span>
                <span className={styles.legendItem}>○ {t('student.questionDetail.legendUnanswered')}</span>
              </div>
            </div>

            {!submitted ? (
              <>
                <button
                  type="button"
                  className={styles.submitBtn}
                  disabled={selected === '' || saving}
                  onClick={() => confirmSubmitQuiz()}
                >
                  <SendOutlined /> {t('student.questionDetail.submitQuiz')}
                </button>
                <button type="button" className={styles.exitBtn} disabled={saving} onClick={() => saveAndExit()}>
                  <SaveOutlined /> {t('student.questionDetail.saveAndExit')}
                </button>
              </>
            ) : null}
          </aside>
        </div>
      ) : null}
    </div>
  )
}

function formatDuration(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return [hours, minutes, seconds].map((part) => String(part).padStart(2, '0')).join(':')
}

function optionLetter(index: number) {
  return String.fromCharCode(65 + index)
}

function questionTitleLabel(title: string, difficulty: string) {
  const match = /^(.*) · (easy|medium|hard) · \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.exec(title)
  const visibleTitle = match ? match[1] : title
  if (difficulty) {
    return { title: visibleTitle, difficulty: difficultyLabel(parseDifficulty(difficulty)) }
  }
  if (!match) {
    return { title, difficulty: '' }
  }
  return { title: match[1], difficulty: difficultyLabel(parseDifficulty(match[2])) }
}

function parseDifficulty(value: string): 'easy' | 'medium' | 'hard' | null {
  if (value === 'easy' || value === 'medium' || value === 'hard') {
    return value
  }
  return null
}

function chosenOption(question: StudentQuestionContent, pending: Record<number, string>) {
  if (Object.prototype.hasOwnProperty.call(pending, question.id)) {
    return pending[question.id]
  }
  return question.userOption
}

function answeredCount(questions: StudentQuestionContent[], pending: Record<number, string>) {
  let count = 0
  for (const question of questions) {
    if (chosenOption(question, pending) !== '') {
      count += 1
    }
  }
  return count
}

function unsavedAnswers(questions: StudentQuestionContent[], pending: Record<number, string>) {
  const items: { contentId: number; option: string }[] = []
  for (const question of questions) {
    const option = chosenOption(question, pending)
    if (option !== '' && option !== question.userOption) {
      items.push({ contentId: question.id, option })
    }
  }
  return items
}

function dropPending(pending: Record<number, string>, contentId: number) {
  const next = { ...pending }
  delete next[contentId]
  return next
}
