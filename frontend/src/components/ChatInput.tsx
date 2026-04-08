import { useEffect, useState } from 'react'
import { useGameStore } from '../store/gameStore'

const T = {
  surfaceHigh: '#221E17', surfaceBorder: '#2E2820',
  accent: '#C4963A', accentDim: 'rgba(196,150,58,0.12)',
  text: '#ECE7DE', textMuted: '#786F62', textDim: '#4A4438',
}
const SANS = "'DM Sans', system-ui, sans-serif"
const MONO = "'JetBrains Mono', monospace"

const INJECTED_ID = 'case-file-chatinput-css'
function injectCSS() {
  if (document.getElementById(INJECTED_ID)) return
  const s = document.createElement('style')
  s.id = INJECTED_ID
  s.textContent = `
    #chat-input-field:focus { border-color: ${T.accent} !important; outline: none; }
    #chat-send-btn:hover:not(:disabled) { background: rgba(196,150,58,0.2) !important; }
    #chat-input-field:disabled { opacity: 0.4; cursor: not-allowed; }
  `
  document.head.appendChild(s)
}

export default function ChatInput() {
  const { phase, sendAction } = useGameStore()
  const [text, setText] = useState('')

  useEffect(() => { injectCSS() }, [])

  const enabled = phase === 'day_discussion' || phase === 'day_vote'

  function handleSend() {
    const trimmed = text.trim()
    if (!trimmed || !enabled) return
    sendAction('chat', { chat: { message: trimmed } })
    setText('')
  }

  return (
    <div style={{
      display: 'flex', gap: '8px', padding: '12px 16px',
      borderTop: `1px solid ${T.surfaceBorder}`,
      flexShrink: 0,
    }}>
      <input
        id="chat-input-field"
        style={{
          flex: 1, background: T.surfaceHigh, color: T.text,
          border: `1px solid ${T.surfaceBorder}`, borderRadius: '2px',
          padding: '9px 12px', fontSize: '13px', fontFamily: SANS,
          transition: 'border-color 150ms ease',
        }}
        placeholder={enabled ? '메시지 입력...' : '이 페이즈에서는 채팅 불가'}
        disabled={!enabled}
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && handleSend()}
      />
      <button
        id="chat-send-btn"
        onClick={handleSend}
        disabled={!enabled || !text.trim()}
        style={{
          background: T.accentDim, color: T.accent,
          border: `1px solid ${T.accent}45`, borderRadius: '2px',
          padding: '9px 16px', fontSize: '12px', fontFamily: MONO,
          textTransform: 'uppercase', letterSpacing: '0.06em',
          cursor: enabled && text.trim() ? 'pointer' : 'not-allowed',
          opacity: enabled && text.trim() ? 1 : 0.4,
          transition: 'all 150ms ease', flexShrink: 0,
        }}
      >
        전송
      </button>
    </div>
  )
}
