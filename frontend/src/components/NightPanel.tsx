import { useGameStore } from '../store/gameStore'

const T = {
  bg: '#0E0C09', surfaceBorder: '#2E2820',
  text: '#ECE7DE', textMuted: '#786F62',
  danger: '#8C1F1F', dangerDim: 'rgba(140,31,31,0.15)',
  police: '#3D7FA8', policeDim: 'rgba(61,127,168,0.12)',
}
const SERIF = "'Instrument Serif', Georgia, serif"
const SANS  = "'DM Sans', system-ui, sans-serif"
const MONO  = "'JetBrains Mono', monospace"

function ActionButton({
  label, color: _color, dimBg, onClick,
}: { label: string; color: string; dimBg: string; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      style={{
        width: '100%', display: 'block', textAlign: 'left', cursor: 'pointer',
        padding: '10px 16px', border: 'none',
        borderBottom: `1px solid ${T.surfaceBorder}`,
        background: 'transparent', transition: 'background 100ms ease',
        fontFamily: SANS, fontSize: '13px', color: T.text,
      }}
      onMouseEnter={(e) => { (e.currentTarget as HTMLButtonElement).style.background = dimBg }}
      onMouseLeave={(e) => { (e.currentTarget as HTMLButtonElement).style.background = 'transparent' }}
    >
      {label}
    </button>
  )
}

export default function NightPanel() {
  const { myRole, room, alivePlayerIDs, playerID, sendAction } = useGameStore()
  if (!room) return null

  const aliveSet = new Set(alivePlayerIDs)
  const targets = room.players.filter((p) => aliveSet.has(p.id) && p.id !== playerID)

  if (myRole === 'mafia') {
    function handleKill(targetID: string) {
      sendAction('kill', { night: { action_type: 'kill', target_id: targetID } })
    }

    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{
          fontFamily: MONO, fontSize: '10px', color: T.danger,
          textTransform: 'uppercase', letterSpacing: '0.12em',
          padding: '14px 16px 10px', borderBottom: `1px solid ${T.surfaceBorder}`,
          flexShrink: 0,
        }}>
          마피아 — 제거 대상
        </div>
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {targets.map((p) => (
            <ActionButton
              key={p.id}
              label={p.name}
              color={T.danger}
              dimBg={T.dangerDim}
              onClick={() => handleKill(p.id)}
            />
          ))}
        </div>
      </div>
    )
  }

  if (myRole === 'police') {
    function handleInvestigate(targetID: string) {
      sendAction('investigate', { night: { action_type: 'investigate', target_id: targetID } })
    }

    return (
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        <div style={{
          fontFamily: MONO, fontSize: '10px', color: T.police,
          textTransform: 'uppercase', letterSpacing: '0.12em',
          padding: '14px 16px 10px', borderBottom: `1px solid ${T.surfaceBorder}`,
          flexShrink: 0,
        }}>
          경찰 — 조사 대상
        </div>
        <div style={{ flex: 1, overflowY: 'auto' }}>
          {targets.map((p) => (
            <ActionButton
              key={p.id}
              label={p.name}
              color={T.police}
              dimBg={T.policeDim}
              onClick={() => handleInvestigate(p.id)}
            />
          ))}
        </div>
        <div style={{
          padding: '10px 16px', borderTop: `1px solid ${T.surfaceBorder}`,
          fontFamily: MONO, fontSize: '10px', color: T.textMuted,
          textTransform: 'uppercase', letterSpacing: '0.08em', flexShrink: 0,
        }}>
          조사 결과 → 채팅창
        </div>
      </div>
    )
  }

  // Citizen
  return (
    <div style={{
      display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
      height: '100%', padding: '24px', textAlign: 'center', gap: '12px',
    }}>
      <span style={{ fontFamily: SERIF, fontSize: '28px', color: T.text, letterSpacing: '-0.02em' }}>
        밤
      </span>
      <span style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.1em' }}>
        마피아가 활동 중...
      </span>
    </div>
  )
}
