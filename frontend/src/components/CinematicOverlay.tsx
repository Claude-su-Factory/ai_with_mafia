import { useEffect } from 'react'
import { useGameStore } from '../store/gameStore'

const THEME_STYLES: Record<string, { bg: string; eyebrowColor: string; titleColor: string; stampColor: string }> = {
  day: {
    bg: 'radial-gradient(ellipse at center, #1A1508 0%, #0E0C09 60%, #0A0900 100%)',
    eyebrowColor: '#C4963A',
    titleColor: '#ECE7DE',
    stampColor: '#C4963A',
  },
  vote: {
    bg: 'radial-gradient(ellipse at center, #180808 0%, #100606 60%, #0A0404 100%)',
    eyebrowColor: '#C4963A',
    titleColor: '#ECE7DE',
    stampColor: '#8C1F1F',
  },
  night: {
    bg: 'radial-gradient(ellipse at center, #0C0810 0%, #060406 60%, #030204 100%)',
    eyebrowColor: '#9B8EBF',
    titleColor: '#D8D0F0',
    stampColor: '#9B8EBF',
  },
  elim: {
    bg: 'radial-gradient(ellipse at center, #180808 0%, #0A0606 60%, #050303 100%)',
    eyebrowColor: '#8C1F1F',
    titleColor: '#ECE7DE',
    stampColor: '#8C1F1F',
  },
  killed: {
    bg: '#000000',
    eyebrowColor: '#8C1F1F',
    titleColor: '#C8C0B8',
    stampColor: '#786F62',
  },
}

const ROLE_PILL_COLORS: Record<string, { bg: string; border: string; color: string }> = {
  mafia:   { bg: 'rgba(140,31,31,0.15)',   border: '#8C1F1F', color: '#C87070' },
  police:  { bg: 'rgba(155,142,191,0.15)', border: '#9B8EBF', color: '#C8C0F8' },
  citizen: { bg: 'rgba(120,111,98,0.15)',  border: '#786F62', color: '#B8B0A8' },
}

const SERIF = "'Instrument Serif', Georgia, serif"
const MONO  = "'JetBrains Mono', monospace"

export default function CinematicOverlay() {
  const { overlayQueue, shiftOverlay } = useGameStore()
  const current = overlayQueue[0]

  useEffect(() => {
    if (!current) return
    const t = setTimeout(shiftOverlay, current.durationMs)
    return () => clearTimeout(t)
  }, [current?.type, current?.title, current?.theme])

  if (!current) return null

  const theme = THEME_STYLES[current.theme] ?? THEME_STYLES.day

  return (
    <div
      onClick={shiftOverlay}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 8000,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        background: theme.bg,
        cursor: 'pointer',
        userSelect: 'none',
      }}
    >
      {/* Scanline effect */}
      <div
        style={{
          position: 'absolute',
          inset: 0,
          backgroundImage: 'repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.08) 2px, rgba(0,0,0,0.08) 4px)',
          pointerEvents: 'none',
        }}
      />
      {/* Vignette */}
      <div
        style={{
          position: 'absolute',
          inset: 0,
          background: 'radial-gradient(ellipse at center, transparent 40%, rgba(0,0,0,0.6) 100%)',
          pointerEvents: 'none',
        }}
      />

      {/* Content */}
      <div
        style={{
          position: 'relative',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: '16px',
          padding: '48px 64px',
          textAlign: 'center',
        }}
      >
        {/* Eyebrow */}
        {current.eyebrow && (
          <div
            style={{
              fontFamily: MONO,
              fontSize: '11px',
              textTransform: 'uppercase',
              letterSpacing: '0.2em',
              color: theme.eyebrowColor,
            }}
          >
            {current.eyebrow}
          </div>
        )}

        {/* Title */}
        <div
          style={{
            fontFamily: SERIF,
            fontSize: 'clamp(48px, 8vw, 96px)',
            color: theme.titleColor,
            lineHeight: 1.1,
          }}
        >
          {current.title}
        </div>

        {/* Hint */}
        {current.hint && (
          <div
            style={{
              fontFamily: MONO,
              fontSize: '12px',
              color: theme.eyebrowColor,
              opacity: 0.7,
              letterSpacing: '0.05em',
            }}
          >
            {current.hint}
          </div>
        )}

        {/* Role pills */}
        {current.rolePills && current.rolePills.length > 0 && (
          <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap', justifyContent: 'center', marginTop: '8px' }}>
            {current.rolePills.map((pill, i) => {
              const pillStyle = ROLE_PILL_COLORS[pill.role] ?? ROLE_PILL_COLORS.citizen
              return (
                <div
                  key={i}
                  style={{
                    fontFamily: MONO,
                    fontSize: '10px',
                    textTransform: 'uppercase',
                    letterSpacing: '0.1em',
                    padding: '4px 12px',
                    borderRadius: '2px',
                    background: pillStyle.bg,
                    border: `1px solid ${pillStyle.border}`,
                    color: pillStyle.color,
                  }}
                >
                  {pill.label}
                </div>
              )
            })}
          </div>
        )}

        {/* Stamp */}
        {current.stamp && (
          <div
            style={{
              fontFamily: MONO,
              fontSize: '13px',
              textTransform: 'uppercase',
              letterSpacing: '0.15em',
              color: theme.stampColor,
              marginTop: '8px',
              opacity: 0.85,
            }}
          >
            {current.stamp}
          </div>
        )}

        {/* Skip hint */}
        <div
          style={{
            position: 'absolute',
            bottom: '-32px',
            fontFamily: MONO,
            fontSize: '9px',
            textTransform: 'uppercase',
            letterSpacing: '0.12em',
            color: '#2E2820',
          }}
        >
          클릭하여 건너뛰기
        </div>
      </div>
    </div>
  )
}
