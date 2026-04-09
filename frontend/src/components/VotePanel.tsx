import { useGameStore } from '../store/gameStore'

const T = {
  surface: '#181410', surfaceHigh: '#221E17', surfaceBorder: '#2E2820',
  accent: '#C4963A', accentDim: 'rgba(196,150,58,0.12)',
  text: '#ECE7DE', textMuted: '#786F62', textDim: '#4A4438',
  danger: '#8C1F1F', dangerDim: 'rgba(140,31,31,0.18)',
}
const SANS  = "'DM Sans', system-ui, sans-serif"
const MONO  = "'JetBrains Mono', monospace"

export default function VotePanel() {
  const { room, alivePlayerIDs, votes, playerID, sendAction } = useGameStore()
  if (!room) return null

  const aliveSet = new Set(alivePlayerIDs)
  const alivePlayers = room.players.filter((p) => aliveSet.has(p.id) && p.id !== playerID)
  const myVote = votes[playerID]

  const voteCounts: Record<string, number> = {}
  Object.values(votes).forEach((targetID) => {
    voteCounts[targetID] = (voteCounts[targetID] ?? 0) + 1
  })
  const totalVotes = Object.values(voteCounts).reduce((a, b) => a + b, 0)

  function handleVote(targetID: string) {
    sendAction('vote', { vote: { target_id: targetID } })
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      {/* Section label */}
      <div style={{
        fontFamily: MONO, fontSize: '10px', color: T.textMuted,
        textTransform: 'uppercase', letterSpacing: '0.12em',
        padding: '14px 16px 10px', borderBottom: `1px solid ${T.surfaceBorder}`,
        flexShrink: 0,
      }}>
        투표 — 처형 대상
      </div>

      {/* Vote list */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '8px 0' }}>
        {alivePlayers.map((p) => {
          const count = voteCounts[p.id] ?? 0
          const pct = totalVotes > 0 ? (count / totalVotes) : 0
          const isMyVote = myVote === p.id

          return (
            <button
              key={p.id}
              onClick={() => handleVote(p.id)}
              style={{
                width: '100%', display: 'block', textAlign: 'left', cursor: 'pointer',
                padding: '10px 16px', border: 'none', borderBottom: `1px solid ${T.surfaceBorder}`,
                background: isMyVote ? T.dangerDim : 'transparent',
                transition: 'background 100ms ease', position: 'relative', overflow: 'hidden',
              }}
            >
              {/* Vote bar (thin, behind content) */}
              <div style={{
                position: 'absolute', bottom: 0, left: 0,
                height: '2px', background: isMyVote ? T.danger : T.surfaceBorder,
                width: `${Math.round(pct * 100)}%`,
                transition: 'width 300ms ease',
              }} />

              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <span style={{
                  fontFamily: SANS, fontSize: '13px',
                  color: isMyVote ? '#ECE7DE' : T.text,
                }}>
                  {p.name}
                </span>
                <span style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted }}>
                  {count}표
                </span>
              </div>
            </button>
          )
        })}
      </div>

      {/* Status footer */}
      {myVote && (
        <div style={{
          padding: '10px 16px', borderTop: `1px solid ${T.surfaceBorder}`,
          fontFamily: MONO, fontSize: '10px', color: T.textMuted,
          textTransform: 'uppercase', letterSpacing: '0.08em', flexShrink: 0,
        }}>
          투표 완료 — 결과 대기 중
        </div>
      )}
    </div>
  )
}
