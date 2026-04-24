import { useNavigate, useParams } from 'react-router-dom'
import { useGameStore } from '../store/gameStore'
import { restartGame } from '../api'
import type { Role } from '../types'
import AdBanner from './AdBanner'

const T = {
  bg: '#0E0C09', surface: '#181410', surfaceHigh: '#221E17', surfaceBorder: '#2E2820',
  accent: '#C4963A', accentDim: 'rgba(196,150,58,0.12)',
  text: '#ECE7DE', textMuted: '#786F62', textDim: '#4A4438',
  danger: '#8C1F1F', dangerDim: 'rgba(140,31,31,0.15)',
  police: '#3D7FA8', policeDim: 'rgba(61,127,168,0.12)',
}
const SERIF = "'Instrument Serif', Georgia, serif"
const SANS  = "'DM Sans', system-ui, sans-serif"
const MONO  = "'JetBrains Mono', monospace"

const ROLE_LABELS: Record<Role, string> = {
  mafia:   'MAFIA',
  police:  'POLICE',
  citizen: 'CITIZEN',
  '':      '?',
}

const ROLE_COLOR: Record<Role, { color: string; bg: string }> = {
  mafia:   { color: T.danger,   bg: T.dangerDim  },
  police:  { color: T.police,   bg: T.policeDim  },
  citizen: { color: T.textMuted, bg: 'transparent' },
  '':      { color: T.textMuted, bg: 'transparent' },
}

export default function ResultOverlay() {
  const { id: roomID } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { result, room, playerID } = useGameStore()

  if (!result || !room || !roomID) return null

  const isHost = room.host_id === playerID
  const isAborted = result.winner === 'aborted'
  const isMafiaWin = result.winner === 'mafia'

  const bannerTitle = isAborted
    ? '게임 중단'
    : isMafiaWin
    ? '마피아의 승리'
    : '시민의 승리'
  const bannerMeta = isAborted
    ? result.reason === 'all_humans_left'
      ? '모든 플레이어가 나가 게임이 중단되었습니다.'
      : '게임이 중단되었습니다.'
    : `${result.round} 라운드 · ${Math.floor(result.duration_sec / 60)}분 ${result.duration_sec % 60}초`
  const bannerAccentColor = isAborted ? T.textMuted : isMafiaWin ? T.danger : '#3A6A3A'

  async function handleRestart() {
    try {
      await restartGame(roomID!)
      useGameStore.setState({ result: null })
    } catch (e) {
      console.error('재시작 실패:', e)
    }
  }

  function handleLeave() {
    navigate('/')
  }

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 50,
      background: 'rgba(14,12,9,0.92)', backdropFilter: 'blur(8px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '16px',
    }}>
      <div style={{
        background: T.surface, border: `1px solid ${T.surfaceBorder}`,
        borderRadius: '4px', width: '100%', maxWidth: '480px',
        overflow: 'hidden',
      }}>

        {/* Winner banner */}
        <div style={{
          padding: '32px 32px 24px',
          borderBottom: `1px solid ${T.surfaceBorder}`,
          background: isAborted
            ? 'linear-gradient(180deg, rgba(120,111,98,0.08) 0%, transparent 100%)'
            : isMafiaWin
            ? 'linear-gradient(180deg, rgba(140,31,31,0.12) 0%, transparent 100%)'
            : 'linear-gradient(180deg, rgba(58,106,58,0.10) 0%, transparent 100%)',
        }}>
          <div style={{
            fontFamily: MONO, fontSize: '10px',
            color: bannerAccentColor,
            textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: '8px',
          }}>
            {isAborted ? '게임 중단' : '게임 종료'}
          </div>
          <h2 style={{
            fontFamily: SERIF, fontSize: '36px', color: T.text,
            margin: '0 0 8px', letterSpacing: '-0.02em', lineHeight: 1.2,
          }}>
            {bannerTitle}
          </h2>
          <div style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted }}>
            {bannerMeta}
          </div>
        </div>

        {/* Player reveal ledger */}
        <div style={{ maxHeight: '320px', overflowY: 'auto' }}>
          {/* Column headers */}
          <div style={{
            display: 'flex', gap: '8px', padding: '8px 24px',
            borderBottom: `1px solid ${T.surfaceBorder}`,
          }}>
            <span style={{ fontFamily: MONO, fontSize: '9px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em', minWidth: '20px' }}>#</span>
            <span style={{ fontFamily: MONO, fontSize: '9px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em', flex: 1 }}>NAME</span>
            <span style={{ fontFamily: MONO, fontSize: '9px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em', minWidth: '60px', textAlign: 'right' }}>IDENTITY</span>
            <span style={{ fontFamily: MONO, fontSize: '9px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em', minWidth: '60px', textAlign: 'right' }}>ROLE</span>
            <span style={{ fontFamily: MONO, fontSize: '9px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em', minWidth: '64px', textAlign: 'right' }}>STATUS</span>
          </div>

          {result.players.map((p, i) => {
            const roleColors = ROLE_COLOR[p.role] ?? ROLE_COLOR['']
            return (
              <div key={p.id} style={{
                display: 'flex', alignItems: 'center', gap: '8px',
                padding: '11px 24px',
                borderBottom: `1px solid ${T.surfaceBorder}`,
                background: i % 2 === 0 ? 'transparent' : 'rgba(34,30,23,0.4)',
                opacity: p.survived ? 1 : 0.65,
              }}>
                {/* Row number */}
                <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textDim, minWidth: '20px', flexShrink: 0 }}>
                  {String(i + 1).padStart(2, '0')}
                </span>

                {/* Name */}
                <span style={{
                  fontFamily: SANS, fontSize: '14px',
                  color: p.survived ? T.text : T.textDim,
                  textDecoration: p.survived ? 'none' : 'line-through',
                  flex: 1,
                }}>
                  {p.name}
                </span>

                {/* AI/Human badge */}
                <span style={{
                  fontFamily: MONO, fontSize: '9px', textTransform: 'uppercase', letterSpacing: '0.08em',
                  minWidth: '60px', textAlign: 'right',
                  color: p.is_ai ? T.accent : T.textMuted,
                }}>
                  {p.is_ai ? 'AI' : 'HUMAN'}
                </span>

                {/* Role badge */}
                <span style={{
                  fontFamily: MONO, fontSize: '9px', textTransform: 'uppercase', letterSpacing: '0.06em',
                  color: roleColors.color, background: roleColors.bg,
                  border: `1px solid ${roleColors.color}35`,
                  padding: '2px 6px', borderRadius: '2px',
                  minWidth: '60px', textAlign: 'center', flexShrink: 0,
                }}>
                  {ROLE_LABELS[p.role]}
                </span>

                {/* Status */}
                <span style={{
                  fontFamily: MONO, fontSize: '9px', textTransform: 'uppercase', letterSpacing: '0.06em',
                  color: p.survived ? '#3A6A3A' : T.danger,
                  minWidth: '64px', textAlign: 'right', flexShrink: 0,
                }}>
                  {p.survived ? 'SURVIVED' : 'ELIMINATED'}
                </span>
              </div>
            )
          })}
        </div>

        <div style={{ margin: '24px 24px 16px' }}>
          <AdBanner slot="result" gameID={roomID} />
        </div>

        {/* Action buttons */}
        <div style={{
          display: 'flex', gap: '8px', padding: '20px 24px',
          borderTop: `1px solid ${T.surfaceBorder}`,
        }}>
          <button
            onClick={handleLeave}
            style={{
              flex: 1, padding: '11px 0', cursor: 'pointer',
              borderRadius: '2px', transition: 'all 150ms ease',
              background: 'transparent', color: T.textMuted,
              border: `1px solid ${T.surfaceBorder}`,
              fontFamily: SANS, fontSize: '13px', fontWeight: 500,
            }}
          >
            나가기
          </button>
          {isHost && (
            <button
              onClick={handleRestart}
              style={{
                flex: 1, padding: '11px 0', cursor: 'pointer',
                borderRadius: '2px', transition: 'all 150ms ease',
                background: T.accentDim, color: T.accent,
                border: `1px solid ${T.accent}50`,
                fontFamily: SANS, fontSize: '13px', fontWeight: 600,
              }}
            >
              다시 시작
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
