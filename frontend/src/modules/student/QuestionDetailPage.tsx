// 套题详情：点选只改本页进度，提交才写入。查看答案在右侧打开内容栏。
import { useEffect, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'

import { useAuthStore } from '@/core/auth/store'
import { answerStudentQuestion, getStudentQuestion, type StudentQuestionContent, type StudentQuestionDetail } from '@/modules/student/api'
import styles from '@/modules/student/QuestionDetailPage.module.css'

export function QuestionDetailPage() {
  const { questionId } = useParams()
  const currentUser = useAuthStore((state) => state.currentUser)
  const [detail, setDetail] = useState<StudentQuestionDetail | null>(null)
  const [errorMessage, setErrorMessage] = useState('')
  const [index, setIndex] = useState(0)
  const [pending, setPending] = useState<Record<number, string>>({})
  const [saving, setSaving] = useState(false)
  const [answerOpen, setAnswerOpen] = useState(false)

  useEffect(() => {
    if (currentUser?.role !== 'ROLE_STUDENT' || !questionId) {
      return
    }
    let cancelled = false
    setIndex(0)
    setDetail(null)
    setPending({})
    setErrorMessage('')
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
    return () => {
      cancelled = true
    }
  }, [questionId, currentUser?.role])

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

  const saveAnswer = (submit: boolean) => {
    if (!questionId || !current || selected === '' || saving) {
      return
    }
    const contentId = current.id
    setSaving(true)
    setErrorMessage('')
    answerStudentQuestion(questionId, contentId, selected, submit)
      .then((updated) => {
        setDetail(updated)
        setPending((currentPending) => dropPending(currentPending, contentId))
      })
      .catch((error: Error) => {
        setErrorMessage(error.message)
      })
      .finally(() => {
        setSaving(false)
      })
  }

  return (
    <>
      <p className={styles.backLine}>
        <Link to="/student/quizzes">← 返回我的测验</Link>
      </p>
      {errorMessage ? <p className={styles.error}>{errorMessage}</p> : null}
      {detail && current && label ? (
        <article className={answerOpen ? styles.withAnswer : undefined}>
          <h1 className={styles.title}>{label.title}</h1>
          <p className={styles.courseLine}>
            <span className={styles.course}>{detail.courseTitle}</span>
            {label.difficulty ? <span className={styles.course}>{label.difficulty}</span> : null}
          </p>
          <div className={styles.progress}>
            <span className={styles.track} aria-hidden="true">
              <span className={styles.fill} style={{ width: `${progressWidth(answered, detail.questionCount)}%` }} />
            </span>
            <span className={styles.percent}>
              {answered}/{detail.questionCount}
            </span>
          </div>
          <p className={styles.meta}>
            Question {questionNumber} of {detail.questionCount}
          </p>
          <div className={styles.question}>
            <p className={styles.content}>{current.content}</p>
            <div className={styles.options} role="group" aria-label="Options">
              {current.options.map((option, optionIndex) => (
                <button
                  key={option}
                  type="button"
                  className={styles.option}
                  aria-pressed={option === selected}
                  onClick={() => {
                    setPending((currentPending) => ({ ...currentPending, [current.id]: option }))
                    setErrorMessage('')
                  }}
                >
                  {optionLetter(optionIndex)}. {option}
                </button>
              ))}
            </div>
          </div>
          <div className={styles.actions}>
            <button className={styles.save} type="button" disabled={selected === '' || saving} onClick={() => saveAnswer(false)}>
              Save
            </button>
            <button className={styles.submit} type="button" disabled={selected === '' || saving} onClick={() => saveAnswer(true)}>
              Submit
            </button>
          </div>
          <div className={styles.numbers} role="list" aria-label="Questions">
            {questions.map((question, questionIndex) => (
              <button
                key={question.id}
                type="button"
                className={styles.number}
                aria-pressed={questionIndex === index}
                onClick={() => setIndex(questionIndex)}
              >
                {questionIndex + 1}
              </button>
            ))}
          </div>
        </article>
      ) : null}
      {current && !answerOpen ? (
        <button className={styles.viewAnswer} type="button" onClick={() => setAnswerOpen(true)}>
          View answer
        </button>
      ) : null}
      {answerOpen && current ? (
        <aside className={styles.detail}>
          <button className={styles.close} type="button" aria-label="Close" onClick={() => setAnswerOpen(false)}>
            ×
          </button>
          <div className={styles.detailBody}>
            <p className={styles.level}>Answer</p>
            <h2 className={styles.detailTitle}>{current.correctAnswer}</h2>
            {current.explanation ? <p className={styles.detailNote}>{current.explanation}</p> : null}
          </div>
          <AnswerChat />
        </aside>
      ) : null}
    </>
  )
}

function optionLetter(index: number) {
  return String.fromCharCode(65 + index)
}

function questionTitleLabel(title: string, difficulty: string) {
  const match = /^(.*) · (easy|medium|hard) · \d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.exec(title)
  const visibleTitle = match ? match[1] : title
  if (difficulty) {
    return { title: visibleTitle, difficulty: difficultyName(difficulty) }
  }
  if (!match) {
    return { title, difficulty: '' }
  }
  return { title: match[1], difficulty: difficultyName(match[2]) }
}

function difficultyName(value: string) {
  if (value === 'easy') {
    return 'Easy'
  }
  if (value === 'medium') {
    return 'Medium'
  }
  if (value === 'hard') {
    return 'Hard'
  }
  return value
}

function chosenOption(question: StudentQuestionContent, pending: Record<number, string>) {
  if (Object.prototype.hasOwnProperty.call(pending, question.id)) {
    return pending[question.id]
  }
  return question.userOption
}

function answeredCount(questions: StudentQuestionContent[], pending: Record<number, string>) {
  let answered = 0
  for (const question of questions) {
    if (chosenOption(question, pending) !== '') {
      answered += 1
    }
  }
  return answered
}

function progressWidth(answered: number, questionCount: number) {
  if (questionCount <= 0 || answered <= 0) {
    return 0
  }
  const percent = Math.floor((answered * 100) / questionCount)
  if (percent > 100) {
    return 100
  }
  return percent
}

function dropPending(pending: Record<number, string>, contentId: number) {
  const next = { ...pending }
  delete next[contentId]
  return next
}

interface ChatMessage {
  role: 'user' | 'reply'
  text: string
}

function AnswerChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [draft, setDraft] = useState('')

  function sendMessage() {
    const text = draft.trim()
    if (!text) {
      return
    }
    setMessages((current) => [...current, { role: 'user', text }, { role: 'reply', text: `Noted: ${text}` }])
    setDraft('')
  }

  return (
    <section className={styles.chat} aria-label="Ask">
      <div className={styles.chatLog}>
        {messages.map((message, index) => (
          <p key={`${message.role}-${index}`} className={message.role === 'user' ? styles.chatUser : styles.chatReply}>
            {message.text}
          </p>
        ))}
      </div>
      <form
        className={styles.chatForm}
        onSubmit={(event) => {
          event.preventDefault()
          sendMessage()
        }}
      >
        <input aria-label="Message" value={draft} onChange={(event) => setDraft(event.target.value)} />
        <button type="submit">Send</button>
      </form>
    </section>
  )
}
