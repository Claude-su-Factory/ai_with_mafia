import { useEffect, useState } from 'react'
import { useGameStore } from '../store/gameStore'
import PhaseHeader from './PhaseHeader'
import PlayerList from './PlayerList'
import ChatLog from './ChatLog'
import ChatInput from './ChatInput'
import VotePanel from './VotePanel'
import NightPanel from './NightPanel'

const T = {
  bg: '#0E0C09', surface: '#181410', surfaceHigh: '#221E17', surfaceBorder: '#2E2820',
  danger: '#8C1F1F', dangerDim: 'rgba(140,31,31,0.15)',
  text: '#ECE7DE', textMuted: '#786F62',
}
const SANS = "'DM Sans', system-ui, sans-serif"
const MONO = "'JetBrains Mono', monospace"

const INJECTED_ID = 'case-file-gameroom-css'
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
    #mafia-chat-input:focus { border-color: ${T.danger} !important; outline: none; }
    ::-webkit-scrollbar { width: 4px; }
    ::-webkit-scrollbar-track { background: ${T.surface}; }
    ::-webkit-scrollbar-thumb { background: ${T.surfaceBorder}; border-radius: 2px; }
  `
  document.head.appendChild(s)
}

export default function GameRoom() {
  const { phase, myRole, sendAction } = useGameStore()
  const [mafiaText, setMafiaText] = useState('')

  useEffect(() => { injectCSS() }, [])

  function handleMafiaChat() {
    const trimmed = mafiaText.trim()
    if (!trimmed) return
    sendAction('chat', { message: trimmed, mafia_only: true })
    setMafiaText('')
  }

  const isNight = phase === 'night'

  return (
    <div style={{
      display: 'flex', flexDirection: 'column', height: '100vh',
      background: T.bg, color: T.text, fontFamily: SANS,
    }}>

      {/* ── Phase header ─────────────────────────────────────────── */}
      <PhaseHeader />

      {/* ── Three-column body ────────────────────────────────────── */}
      <div style={{ flex: 1, display: 'flex', overflow: 'hidden', minHeight: 0 }}>

        {/* Left: player ledger (240px) */}
        <aside style={{
          width: '240px', flexShrink: 0,
          borderRight: `1px solid ${T.surfaceBorder}`,
          overflowY: 'auto',
        }}>
          <PlayerList />
        </aside>

        {/* Center: chat (flex 1) */}
        <main style={{
          flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0,
          borderRight: `1px solid ${T.surfaceBorder}`,
          background: isNight ? 'rgba(14,12,9,0.97)' : T.bg,
          transition: 'background 600ms ease',
        }}>
          <ChatLog />
          <ChatInput />

          {/* Mafia night channel */}
          {phase === 'night' && myRole === 'mafia' && (
            <div style={{
              display: 'flex', gap: '8px', padding: '10px 16px',
              borderTop: `1px solid ${T.danger}25`,
              background: T.dangerDim,
            }}>
              <input
                id="mafia-chat-input"
                style={{
                  flex: 1, background: 'rgba(140,31,31,0.12)', color: T.text,
                  border: `1px solid ${T.danger}40`, borderRadius: '2px',
                  padding: '8px 12px', fontSize: '13px', fontFamily: SANS,
                  transition: 'border-color 150ms ease',
                }}
                placeholder="마피아 채널 (마피아끼리만)"
                value={mafiaText}
                onChange={(e) => setMafiaText(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleMafiaChat()}
              />
              <button
                onClick={handleMafiaChat}
                disabled={!mafiaText.trim()}
                style={{
                  background: 'rgba(140,31,31,0.2)', color: T.danger,
                  border: `1px solid ${T.danger}40`, borderRadius: '2px',
                  padding: '8px 14px', fontSize: '11px', fontFamily: MONO,
                  textTransform: 'uppercase', letterSpacing: '0.06em',
                  cursor: mafiaText.trim() ? 'pointer' : 'not-allowed',
                  opacity: mafiaText.trim() ? 1 : 0.4,
                  transition: 'all 150ms ease',
                }}
              >
                전송
              </button>
            </div>
          )}
        </main>

        {/* Right: action panel (280px) */}
        <aside style={{
          width: '280px', flexShrink: 0,
          display: 'flex', flexDirection: 'column',
        }}>
          {phase === 'day_vote' && <VotePanel />}
          {phase === 'night' && <NightPanel />}
          {(phase === 'day_discussion' || !phase) && (
            <div style={{
              display: 'flex', flexDirection: 'column',
              alignItems: 'center', justifyContent: 'center',
              height: '100%', padding: '32px 24px', textAlign: 'center', gap: '8px',
            }}>
              <div style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.1em' }}>
                토론 페이즈
              </div>
              <div style={{ fontFamily: SANS, fontSize: '13px', color: T.textMuted }}>
                채팅을 통해 의심스러운 플레이어를 찾아내세요.
              </div>
            </div>
          )}
        </aside>
      </div>
    </div>
  )
}
