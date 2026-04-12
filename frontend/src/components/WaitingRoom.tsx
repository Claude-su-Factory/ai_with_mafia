import { useEffect } from 'react'
import { useGameStore } from '../store/gameStore'
import { startGame } from '../api'
import AdBanner from './AdBanner'

const T = {
  bg: '#0E0C09', surface: '#181410', surfaceHigh: '#221E17', surfaceBorder: '#2E2820',
  accent: '#C4963A', accentDim: 'rgba(196,150,58,0.12)',
  text: '#ECE7DE', textMuted: '#786F62', textDim: '#4A4438',
}
const SERIF = "'Instrument Serif', Georgia, serif"
const SANS  = "'DM Sans', system-ui, sans-serif"
const MONO  = "'JetBrains Mono', monospace"

const INJECTED_ID = 'case-file-waiting-css'
function injectCSS() {
  if (document.getElementById(INJECTED_ID)) return
  const s = document.createElement('style')
  s.id = INJECTED_ID
  s.textContent = `
    body::before {
      content: ''; position: fixed; inset: 0; pointer-events: none; z-index: 9999;
      opacity: 0.04;
      background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
      background-size: 128px;
    }
    #waiting-start-btn:hover:not(:disabled) { background: rgba(196,150,58,0.2) !important; }
    @keyframes waitPulse { 0%,100% { opacity: 1; } 50% { opacity: 0.5; } }
  `
  document.head.appendChild(s)
}

export default function WaitingRoom() {
  const { room, playerID } = useGameStore()

  useEffect(() => { injectCSS() }, [])

  if (!room) return null

  const isHost = room.host_id === playerID
  const canStart = room.players.length >= 1

  async function handleStart() {
    if (!room) return
    try {
      await startGame(room.id)
    } catch (e) {
      console.error('게임 시작 실패:', e)
    }
  }

  return (
    <div style={{
      minHeight: '100dvh', background: T.bg, color: T.text, fontFamily: SANS,
      display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
      padding: '32px',
    }}>
      <div style={{ width: '100%', maxWidth: '480px' }}>

        {/* Section label */}
        <div style={{
          fontFamily: MONO, fontSize: '10px', color: T.textMuted,
          textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: '12px',
        }}>
          대기 중 — Waiting Room
        </div>

        {/* Room name */}
        <h1 style={{
          fontFamily: SERIF, fontSize: '36px', color: T.text,
          margin: '0 0 32px', lineHeight: 1.2, letterSpacing: '-0.02em',
        }}>
          {room.name}
        </h1>

        {/* Join code */}
        {room.visibility === 'private' && room.join_code && (
          <div style={{ marginBottom: '32px' }}>
            <div style={{
              fontFamily: MONO, fontSize: '10px', color: T.textMuted,
              textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: '8px',
            }}>
              초대 코드
            </div>
            <div style={{
              fontFamily: MONO, fontSize: '32px', letterSpacing: '0.3em', color: T.accent,
              padding: '16px 20px',
              background: 'rgba(196,150,58,0.06)', border: `1px solid ${T.accent}30`,
              borderRadius: '4px', display: 'inline-block',
            }}>
              {room.join_code}
            </div>
          </div>
        )}

        {/* Player ledger */}
        <div style={{ marginBottom: '32px' }}>
          <div style={{
            fontFamily: MONO, fontSize: '10px', color: T.textMuted,
            textTransform: 'uppercase', letterSpacing: '0.12em',
            paddingBottom: '10px', borderBottom: `1px solid ${T.surfaceBorder}`,
            marginBottom: '0',
          }}>
            플레이어 · {room.players.length}명
          </div>

          {room.players.map((p, i) => (
            <div key={p.id} style={{
              display: 'flex', alignItems: 'center', gap: '10px',
              padding: '11px 0', borderBottom: `1px solid ${T.surfaceBorder}`,
            }}>
              <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textDim, minWidth: '20px' }}>
                {String(i + 1).padStart(2, '0')}
              </span>
              <span style={{ width: '5px', height: '5px', borderRadius: '50%', background: '#3A6A3A', flexShrink: 0 }} />
              <span style={{ flex: 1, fontFamily: SANS, fontSize: '14px', color: T.text }}>
                {p.name}
              </span>
              <div style={{ display: 'flex', gap: '6px' }}>
                {p.id === room.host_id && (
                  <span style={{
                    fontFamily: MONO, fontSize: '9px', color: T.accent,
                    background: 'rgba(196,150,58,0.1)', border: `1px solid ${T.accent}30`,
                    borderRadius: '2px', padding: '2px 5px', textTransform: 'uppercase', letterSpacing: '0.06em',
                  }}>
                    HOST
                  </span>
                )}
                {p.id === playerID && (
                  <span style={{ fontFamily: MONO, fontSize: '9px', color: T.textMuted, letterSpacing: '0.05em' }}>
                    ME
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>

        <AdBanner
          slotId={import.meta.env.VITE_ADSENSE_SLOT_WAITING}
          style={{ marginBottom: '24px' }}
        />

        {/* Start button / waiting text */}
        {isHost ? (
          <button
            id="waiting-start-btn"
            onClick={handleStart}
            disabled={!canStart}
            style={{
              width: '100%', padding: '14px 0', cursor: canStart ? 'pointer' : 'not-allowed',
              borderRadius: '2px', transition: 'all 200ms ease',
              background: T.accentDim, color: T.accent,
              border: `1px solid ${T.accent}50`,
              fontFamily: SANS, fontSize: '15px', fontWeight: 600,
              opacity: canStart ? 1 : 0.4,
            }}
          >
            게임 시작
          </button>
        ) : (
          <div style={{
            textAlign: 'center', fontFamily: MONO, fontSize: '11px', color: T.textMuted,
            textTransform: 'uppercase', letterSpacing: '0.1em',
            animation: 'waitPulse 2.4s ease-in-out infinite',
          }}>
            방장이 게임을 시작할 때까지 대기 중...
          </div>
        )}
      </div>
    </div>
  )
}
