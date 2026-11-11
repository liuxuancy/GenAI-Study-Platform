import { useEffect, useMemo, useRef, useState } from 'react'
import {
  CaretDownOutlined,
  CloseOutlined,
  EyeInvisibleOutlined,
  FastForwardOutlined,
  HistoryOutlined,
  RollbackOutlined,
} from '@ant-design/icons'

import { useI18n } from '@/core/i18n/useI18n'
import { GalChapterComplete } from '@/modules/student/gal/GalChapterComplete'
import { galBackgroundUrl, resolveGalBackground } from '@/modules/student/gal/galBackgrounds'
import { spriteForEmotion } from '@/modules/student/gal/characterSprites'
import type { GalEmotion, GalScriptItem } from '@/modules/student/gal/galTypes'
import styles from '@/modules/student/gal/GalGameScreen.module.css'

const typingSpeedMs = 28
const autoDelayMs = 2200

export function GalGameScreen({
  script,
  title,
  nextChapterKey,
  courseRunMode,
  onChapterComplete,
  onNextChapter,
  onReplay,
  onExit,
}: {
  script: GalScriptItem[]
  title: string
  nextChapterKey?: string | null
  courseRunMode: boolean
  onChapterComplete: () => void
  onNextChapter: (chapterKey: string) => void
  onReplay: () => void
  onExit: () => void
}) {
  const { t } = useI18n()
  const [currentIndex, setCurrentIndex] = useState(0)
  const [displayedText, setDisplayedText] = useState('')
  const [isTyping, setIsTyping] = useState(false)
  const [isAuto, setIsAuto] = useState(false)
  const [showLog, setShowLog] = useState(false)
  const [hideUI, setHideUI] = useState(false)
  const [jumpKey, setJumpKey] = useState(0)
  const [choiceFeedback, setChoiceFeedback] = useState<string | null>(null)
  const [choiceEmotion, setChoiceEmotion] = useState<GalEmotion>('normal')
  const [pickedChoiceIndex, setPickedChoiceIndex] = useState<number | null>(null)
  const [finished, setFinished] = useState(false)
  const [completeNotified, setCompleteNotified] = useState(false)
  const timerRef = useRef<number | null>(null)
  const autoTimerRef = useRef<number | null>(null)

  const currentItem = script[currentIndex]
  const isChoice = currentItem?.kind === 'choice'
  const isDialogue = currentItem?.kind === 'dialogue'

  const sceneBackgroundUrl = useMemo(
    () =>
      galBackgroundUrl(
        resolveGalBackground({
          script,
          currentIndex,
          finished,
          currentItem,
        }),
      ),
    [script, currentIndex, finished, currentItem],
  )

  useEffect(() => {
    setJumpKey((value) => value + 1)
    setChoiceFeedback(null)
    setChoiceEmotion('normal')
    setPickedChoiceIndex(null)
  }, [currentIndex])

  useEffect(() => {
    if (!isDialogue || !currentItem || choiceFeedback) {
      return
    }
    setDisplayedText('')
    setIsTyping(true)
    let charIndex = 0
    if (timerRef.current) {
      window.clearInterval(timerRef.current)
    }
    timerRef.current = window.setInterval(() => {
      charIndex += 1
      setDisplayedText(currentItem.text.slice(0, charIndex))
      if (charIndex >= currentItem.text.length) {
        setIsTyping(false)
        if (timerRef.current) {
          window.clearInterval(timerRef.current)
        }
      }
    }, typingSpeedMs)
    return () => {
      if (timerRef.current) {
        window.clearInterval(timerRef.current)
      }
    }
  }, [currentIndex, currentItem, isDialogue, choiceFeedback])

  function finishChapter() {
    setIsAuto(false)
    if (!completeNotified) {
      onChapterComplete()
      setCompleteNotified(true)
    }
    setFinished(true)
  }

  function handleNext() {
    if (finished || !currentItem) {
      return
    }
    if (isChoice && !choiceFeedback) {
      return
    }
    if (isDialogue && isTyping) {
      if (timerRef.current) {
        window.clearInterval(timerRef.current)
      }
      setDisplayedText(currentItem.text)
      setIsTyping(false)
      return
    }
    if (currentIndex < script.length - 1) {
      setCurrentIndex((value) => value + 1)
      return
    }
    finishChapter()
  }

  function pickChoice(optionIndex: number) {
    if (!currentItem || currentItem.kind !== 'choice' || choiceFeedback) {
      return
    }
    const correct = optionIndex === currentItem.correctIndex
    setPickedChoiceIndex(optionIndex)
    setChoiceFeedback(correct ? currentItem.feedbackCorrect : currentItem.feedbackWrong)
    setChoiceEmotion(correct ? currentItem.emotionOnCorrect ?? 'happy' : currentItem.emotionOnWrong ?? 'shy')
  }

  useEffect(() => {
    if (!isAuto || finished || isTyping || (isChoice && !choiceFeedback)) {
      return
    }
    if (currentIndex >= script.length - 1 && choiceFeedback) {
      autoTimerRef.current = window.setTimeout(() => finishChapter(), autoDelayMs)
      return () => {
        if (autoTimerRef.current) {
          window.clearTimeout(autoTimerRef.current)
        }
      }
    }
    if (currentIndex < script.length - 1 && (!isChoice || choiceFeedback)) {
      autoTimerRef.current = window.setTimeout(() => handleNext(), autoDelayMs)
    }
    return () => {
      if (autoTimerRef.current) {
        window.clearTimeout(autoTimerRef.current)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuto, isTyping, currentIndex, script.length, choiceFeedback, finished, isChoice])

  if (finished) {
    return (
      <div className={styles.galRoot}>
        <img className={styles.sceneBackground} src={sceneBackgroundUrl} alt="" aria-hidden="true" />
        <div className={styles.sceneVignette} aria-hidden="true" />
        <GalChapterComplete
          hasNextChapter={Boolean(nextChapterKey)}
          courseRunMode={courseRunMode}
          onNextChapter={() => {
            if (nextChapterKey) {
              onNextChapter(nextChapterKey)
            }
          }}
          onReplay={onReplay}
          onExit={onExit}
        />
      </div>
    )
  }

  if (!currentItem) {
    return <div className={styles.endText}>{t('student.gal.endChapter')}</div>
  }

  const spriteEmotion =
    isChoice && choiceFeedback
      ? choiceEmotion
      : isDialogue
        ? currentItem.emotion
        : 'normal'

  const dialogSpeaker = isDialogue ? currentItem.speaker : t('student.gal.choiceHost')
  const dialogBody =
    isChoice && !choiceFeedback
      ? currentItem.prompt
      : isChoice && choiceFeedback
        ? choiceFeedback
        : displayedText

  const progressPercent = script.length > 0 ? ((currentIndex + 1) / script.length) * 100 : 0

  const dialogBoxClass = [
    styles.dialogBox,
    isChoice && !choiceFeedback ? styles.dialogBoxChoice : '',
    isChoice &&
    choiceFeedback &&
    pickedChoiceIndex !== null &&
    currentItem.kind === 'choice' &&
    pickedChoiceIndex === currentItem.correctIndex
      ? styles.dialogBoxFeedbackOk
      : '',
    isChoice &&
    choiceFeedback &&
    pickedChoiceIndex !== null &&
    currentItem.kind === 'choice' &&
    pickedChoiceIndex !== currentItem.correctIndex
      ? styles.dialogBoxFeedbackWarn
      : '',
  ]
    .filter(Boolean)
    .join(' ')

  const choiceItem = currentItem.kind === 'choice' ? currentItem : null

  return (
    <div
      className={`${styles.galRoot}${isChoice && !choiceFeedback ? ` ${styles.galRootChoice}` : ''}`}
      onClick={isChoice && !choiceFeedback ? undefined : handleNext}
      role="presentation"
    >
      <img className={styles.sceneBackground} src={sceneBackgroundUrl} alt="" aria-hidden="true" />
      <div className={styles.sceneVignette} aria-hidden="true" />
      <div className={styles.progressBar} aria-hidden="true">
        <div className={styles.progressFill} style={{ width: `${progressPercent}%` }} />
      </div>
      <div className={styles.progressLabel} aria-hidden="true">
        {currentIndex + 1} / {script.length}
      </div>
      {!hideUI ? (
        <div key={jumpKey} className={styles.spriteWrap}>
          <img className={styles.sprite} src={spriteForEmotion(spriteEmotion)} alt="" />
        </div>
      ) : null}

      {!hideUI ? (
        <div className={styles.uiLayer}>
          <div className={styles.scriptColumn}>
            {isDialogue && currentItem.note ? (
              <div className={styles.notePop}>
                <div className={styles.noteLabel}>{t('student.gal.termNote')}</div>
                <div>{currentItem.note}</div>
              </div>
            ) : null}
            <div className={styles.controls}>
              <button
                type="button"
                className={isAuto ? `${styles.controlBtn} ${styles.controlBtnActive}` : styles.controlBtn}
                onClick={(event) => {
                  event.stopPropagation()
                  setIsAuto((value) => !value)
                }}
              >
                <FastForwardOutlined /> <span className={styles.controlLabel}>{t('student.gal.auto')}</span>
              </button>
              <button
                type="button"
                className={styles.controlBtn}
                onClick={(event) => {
                  event.stopPropagation()
                  setShowLog(true)
                }}
              >
                <HistoryOutlined /> <span className={styles.controlLabel}>{t('student.gal.log')}</span>
              </button>
              <button
                type="button"
                className={styles.controlBtn}
                onClick={(event) => {
                  event.stopPropagation()
                  setHideUI(true)
                }}
              >
                <EyeInvisibleOutlined /> <span className={styles.controlLabel}>{t('student.gal.hideUi')}</span>
              </button>
              <button
                type="button"
                className={styles.controlBtn}
                onClick={(event) => {
                  event.stopPropagation()
                  onExit()
                }}
              >
                <RollbackOutlined /> <span className={styles.controlLabel}>{t('student.gal.exitPlay')}</span>
              </button>
            </div>
            {choiceItem ? (
              <div className={styles.choiceList}>
                {choiceItem.options.map((option, index) => {
                  const answered = Boolean(choiceFeedback)
                  const isPicked = pickedChoiceIndex === index
                  const isCorrectOption = index === choiceItem.correctIndex
                  const pickedWrong = answered && pickedChoiceIndex !== null && pickedChoiceIndex !== choiceItem.correctIndex
                  const choiceClasses = [styles.choiceBtn]
                  if (answered && isPicked && isCorrectOption) {
                    choiceClasses.push(styles.choiceBtnPickedOk)
                  } else if (answered && isPicked && !isCorrectOption) {
                    choiceClasses.push(styles.choiceBtnPickedWarn)
                  } else if (answered && pickedWrong && isCorrectOption) {
                    choiceClasses.push(styles.choiceBtnPickedOk)
                  }
                  return (
                    <button
                      key={`${index}-${option}`}
                      type="button"
                      className={choiceClasses.join(' ')}
                      disabled={answered}
                      onClick={(event) => {
                        event.stopPropagation()
                        pickChoice(index)
                      }}
                    >
                      <span className={styles.choiceIndex}>{String.fromCharCode(65 + index)}</span>
                      <span>{option}</span>
                    </button>
                  )
                })}
              </div>
            ) : null}
            <div className={dialogBoxClass}>
              <div className={styles.nameTag}>
                <span className={styles.speakerBadge}>{dialogSpeaker}</span>
                <span className={styles.chapterBadge} title={title}>
                  {title}
                </span>
              </div>
              <div className={`${styles.dialogText}${isChoice ? ` ${styles.dialogTextChoice}` : ''}`}>
                {dialogBody}
                {isDialogue && isTyping ? <span className={styles.caret} aria-hidden="true" /> : null}
              </div>
              {!isTyping && (!isChoice || choiceFeedback) ? (
                <span className={styles.nextHint} aria-hidden="true">
                  <CaretDownOutlined />
                </span>
              ) : null}
            </div>
          </div>
        </div>
      ) : (
        <div
          className={styles.hideTap}
          onClick={(event) => {
            event.stopPropagation()
            setHideUI(false)
          }}
          role="presentation"
        />
      )}

      {showLog ? (
        <div className={styles.logOverlay} onClick={(event) => event.stopPropagation()} role="presentation">
          <div className={styles.logPanel}>
            <div className={styles.logHead}>
              <span>{t('student.gal.logTitle')}</span>
              <button type="button" className={styles.controlBtn} onClick={() => setShowLog(false)}>
                <CloseOutlined />
              </button>
            </div>
            <div className={styles.logBody}>
              {script.slice(0, currentIndex + 1).map((line, index) => (
                <div key={`${index}-log`} className={styles.logLine}>
                  <div className={styles.logSpeaker}>
                    {line.kind === 'dialogue' ? line.speaker : t('student.gal.choiceHost')}
                  </div>
                  <div>{line.kind === 'dialogue' ? line.text : line.prompt}</div>
                </div>
              ))}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}
