import { useEffect, useRef, useState } from 'react'
import { useGameStore } from '../store/gameStore'
import type { Phase } from '../types'
import { prepare, layout } from '@chenglou/pretext'

const T = {
  bg: '#0E0C09', surfaceBorder: '#2E2820',
  accent: '#C4963A', text: '#ECE7DE', textMuted: '#786F62', danger: '#8C1F1F',
}
const SERIF = "'Instrument Serif', Georgia, serif"
const MONO  = "'JetBrains Mono', monospace"

function FullscreenButton() {
  const [isFull, setIsFull] = useState(false)

  useEffect(() => {
    const handler = () => setIsFull(!!document.fullscreenElement)
    document.addEventListener('fullscreenchange', handler)
    return () => document.removeEventListener('fullscreenchange', handler)
  }, [])

  function toggle() {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen()
    } else {
      document.exitFullscreen()
    }
  }

  return (
    <button
      onClick={toggle}
      style={{
        background: 'transparent',
        border: '1px solid #2E2820',
        borderRadius: '2px',
        color: '#786F62',
        fontFamily: MONO,
        fontSize: '14px',
        width: '28px',
        height: '28px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        cursor: 'pointer',
        transition: 'border-color 150ms ease, color 150ms ease',
        flexShrink: 0,
      }}
      title={isFull ? '풀스크린 해제' : '풀스크린'}
    >
      {isFull ? '✕' : '⛶'}
    </button>
  )
}

const PHASE_LABELS: Record<Phase, string> = {
  day_discussion: '낮 — 토론',
  day_vote:       '낮 — 투표',
  night:          '밤',
  result:         '종료',
}

export default function PhaseHeader() {
  const { phase, round, timerRemainingSec } = useGameStore()
  const phaseRef = useRef<HTMLSpanElement>(null)
  const phaseHandle = useRef<ReturnType<typeof prepare> | null>(null)

  useEffect(() => {
    const el = phaseRef.current
    if (!el || !phase) return
    let ro: ResizeObserver | null = null
    document.fonts.ready.then(() => {
      phaseHandle.current = prepare(el.textContent ?? '', getComputedStyle(el).font)
      function relayout() {
        if (!phaseHandle.current || !el) return
        const lh = parseFloat(getComputedStyle(el).lineHeight)
        const { height } = layout(phaseHandle.current, el.clientWidth, lh)
        el.style.height = `${height}px`
      }
      relayout()
      ro = new ResizeObserver(relayout)
      ro.observe(el)
    })
    return () => ro?.disconnect()
  }, [phase])

  if (!phase) return null

  const label = PHASE_LABELS[phase] ?? phase
  const mins = Math.floor(timerRemainingSec / 60)
  const secs = timerRemainingSec % 60
  const timerStr = mins > 0
    ? `${mins}:${secs.toString().padStart(2, '0')}`
    : `${secs}s`
  const isUrgent = timerRemainingSec > 0 && timerRemainingSec <= 10
  const isNight = phase === 'night'

  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      padding: '0 24px', height: '56px', flexShrink: 0,
      borderBottom: `1px solid ${T.surfaceBorder}`,
      background: isNight ? 'rgba(14,12,9,0.98)' : T.bg,
    }}>
      {/* Left: phase + round */}
      <div style={{ display: 'flex', alignItems: 'baseline', gap: '12px' }}>
        <span
          ref={phaseRef}
          style={{ fontFamily: SERIF, fontSize: '20px', color: T.text, lineHeight: '56px', display: 'inline-block' }}
        >
          {label}
        </span>
        <span style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.1em' }}>
          ROUND {round}
        </span>
      </div>

      {/* Right: timer + fullscreen */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        {timerRemainingSec > 0 && (
          <span style={{
            fontFamily: MONO, fontSize: '28px', letterSpacing: '-0.02em',
            color: isUrgent ? T.danger : T.accent,
            transition: 'color 300ms ease',
            fontVariantNumeric: 'tabular-nums',
          }}>
            {timerStr}
          </span>
        )}
        <FullscreenButton />
      </div>
    </div>
  )
}
