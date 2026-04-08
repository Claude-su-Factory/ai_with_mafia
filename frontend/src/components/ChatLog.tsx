import { useEffect, useRef } from 'react'
import { useGameStore } from '../store/gameStore'

const T = {
  surfaceBorder: '#2E2820', accent: '#C4963A',
  text: '#ECE7DE', textMuted: '#786F62', textDim: '#4A4438',
  danger: '#8C1F1F', police: '#3D7FA8',
}
const SANS = "'DM Sans', system-ui, sans-serif"
const MONO = "'JetBrains Mono', monospace"

export default function ChatLog() {
  const { messages, room } = useGameStore()
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  function getPlayerName(playerID: string, playerName?: string) {
    if (playerID === 'system') return ''
    if (playerName) return playerName
    return room?.players.find((p) => p.id === playerID)?.name ?? playerID
  }

  return (
    <div style={{
      flex: 1, overflowY: 'auto', padding: '12px 16px',
      display: 'flex', flexDirection: 'column', gap: '4px',
    }}>
      {messages.length === 0 && (
        <div style={{
          flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
          fontFamily: MONO, fontSize: '11px', color: T.textDim,
          textTransform: 'uppercase', letterSpacing: '0.1em',
        }}>
          채팅 없음
        </div>
      )}

      {messages.map((msg) => {
        if (msg.is_system) {
          return (
            <div key={msg.id} style={{
              textAlign: 'center', padding: '6px 0',
              fontFamily: MONO, fontSize: '11px', color: T.textMuted,
              fontStyle: 'italic', letterSpacing: '0.02em',
              borderTop: `1px solid ${T.surfaceBorder}`,
              borderBottom: `1px solid ${T.surfaceBorder}`,
              margin: '4px 0',
            }}>
              {msg.message}
            </div>
          )
        }

        const senderName = getPlayerName(msg.player_id, msg.player_name)
        const nameColor = msg.mafia_only ? T.danger : T.textMuted

        return (
          <div key={msg.id} style={{ display: 'flex', gap: '8px', alignItems: 'flex-start' }}>
            {/* Sender name */}
            <span style={{
              fontFamily: MONO, fontSize: '11px', color: nameColor,
              flexShrink: 0, paddingTop: '2px', minWidth: '80px', maxWidth: '120px',
              overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
            }}>
              {senderName}
              {msg.mafia_only && (
                <span style={{ fontSize: '9px', marginLeft: '4px', opacity: 0.7 }}>✦</span>
              )}
            </span>

            {/* Separator */}
            <span style={{ color: T.textDim, fontFamily: MONO, fontSize: '11px', paddingTop: '2px', flexShrink: 0 }}>
              —
            </span>

            {/* Message text */}
            <span style={{
              fontFamily: SANS, fontSize: '13px', color: T.text,
              lineHeight: '1.5', flex: 1,
            }}>
              {msg.message}
            </span>
          </div>
        )
      })}

      <div ref={bottomRef} />
    </div>
  )
}
