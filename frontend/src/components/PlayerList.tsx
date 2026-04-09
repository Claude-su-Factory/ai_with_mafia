import { useGameStore } from '../store/gameStore'
import type { Role } from '../types'

const T = {
  bg: '#0E0C09', surface: '#181410', surfaceBorder: '#2E2820',
  accent: '#C4963A', text: '#ECE7DE', textMuted: '#786F62', textDim: '#4A4438',
  danger: '#8C1F1F',
}
const SANS  = "'DM Sans', system-ui, sans-serif"
const MONO  = "'JetBrains Mono', monospace"

const ROLE_LABELS: Record<Role, string> = {
  mafia:   '마피아',
  police:  '경찰',
  citizen: '시민',
  '':      '',
}

export default function PlayerList() {
  const { room, alivePlayerIDs, playerID } = useGameStore()
  if (!room) return null

  const aliveSet = new Set(alivePlayerIDs)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Section label */}
      <div style={{
        fontFamily: MONO, fontSize: '10px', color: T.textMuted,
        textTransform: 'uppercase', letterSpacing: '0.12em',
        padding: '14px 12px 10px',
        borderBottom: `1px solid ${T.surfaceBorder}`,
        flexShrink: 0,
      }}>
        Players · {room.players.length}
      </div>

      {/* Ledger rows */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {room.players.map((p, i) => {
          const alive = aliveSet.has(p.id)
          const isMe = p.id === playerID
          const isHost = p.id === room.host_id
          const deadRole = !alive
            ? (p as unknown as { role?: Role }).role
            : null

          return (
            <div
              key={p.id}
              style={{
                display: 'flex', alignItems: 'flex-start', gap: '8px',
                padding: '10px 12px',
                borderBottom: `1px solid ${T.surfaceBorder}`,
                position: 'relative',
              }}
            >
              {/* Row number */}
              <span style={{
                fontFamily: MONO, fontSize: '10px', color: T.textDim,
                minWidth: '18px', flexShrink: 0, lineHeight: '18px',
                paddingTop: '1px',
              }}>
                {String(i + 1).padStart(2, '0')}
              </span>

              {/* Status dot */}
              <span style={{
                width: '5px', height: '5px', borderRadius: '50%',
                flexShrink: 0, marginTop: '7px',
                background: alive ? '#3A6A3A' : T.danger,
              }} />

              {/* Name + labels */}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}>
                  <span style={{
                    fontFamily: SANS, fontSize: '13px', lineHeight: '18px',
                    color: alive ? T.text : T.textDim,
                    textDecoration: alive ? 'none' : 'line-through',
                  }}>
                    {p.name}
                  </span>
                  {isMe && (
                    <span style={{ fontFamily: MONO, fontSize: '9px', color: T.textMuted, letterSpacing: '0.06em' }}>
                      ME
                    </span>
                  )}
                  {isHost && (
                    <span style={{
                      fontFamily: MONO, fontSize: '9px', color: T.accent,
                      background: 'rgba(196,150,58,0.1)', border: `1px solid ${T.accent}30`,
                      borderRadius: '2px', padding: '0 4px', letterSpacing: '0.05em',
                    }}>
                      HOST
                    </span>
                  )}
                </div>

                {/* Revealed role (dead players) */}
                {!alive && deadRole && (
                  <div style={{ marginTop: '3px' }}>
                    <span style={{
                      fontFamily: MONO, fontSize: '9px', color: T.danger,
                      textTransform: 'uppercase', letterSpacing: '0.08em',
                    }}>
                      {ROLE_LABELS[deadRole]}
                    </span>
                  </div>
                )}
              </div>

              {/* ELIMINATED stamp */}
              {!alive && (
                <span style={{
                  fontFamily: MONO, fontSize: '8px', color: T.danger,
                  border: `1px solid ${T.danger}55`, borderRadius: '2px',
                  padding: '2px 5px', letterSpacing: '0.05em', textTransform: 'uppercase',
                  transform: 'rotate(-5deg)', flexShrink: 0, alignSelf: 'center',
                }}>
                  사망
                </span>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
