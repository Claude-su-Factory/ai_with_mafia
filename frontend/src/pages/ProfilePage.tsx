import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { updateMe, getMyStats, getMyGames } from '../api'
import type { MyStatsResponse, MyGameRecord } from '../api'

const T = {
  bg:            '#0E0C09',
  surface:       '#181410',
  surfaceHigh:   '#221E17',
  surfaceBorder: '#2E2820',
  accent:        '#C4963A',
  accentDim:     'rgba(196,150,58,0.12)',
  text:          '#ECE7DE',
  textMuted:     '#786F62',
  textDim:       '#4A4438',
  danger:        '#8C1F1F',
  dangerDim:     'rgba(140,31,31,0.15)',
  police:        '#3D7FA8',
  policeDim:     'rgba(61,127,168,0.12)',
}
const SERIF = "'Instrument Serif', Georgia, serif"
const SANS  = "'DM Sans', system-ui, sans-serif"
const MONO  = "'JetBrains Mono', monospace"

const ROLE_LABEL: Record<string, string> = {
  mafia:   '마피아',
  citizen: '시민',
  police:  '경찰',
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  const mm = String(d.getMonth() + 1).padStart(2, '0')
  const dd = String(d.getDate()).padStart(2, '0')
  return `${mm}-${dd}`
}

function formatDuration(sec: number): string {
  if (sec <= 0) return '0분'
  return `${Math.max(1, Math.round(sec / 60))}분`
}

export default function ProfilePage() {
  const navigate = useNavigate()
  const { user, displayName, loading: authLoading } = useAuthStore()

  const [stats, setStats] = useState<MyStatsResponse | null>(null)
  const [games, setGames] = useState<MyGameRecord[]>([])
  const [statsLoading, setStatsLoading] = useState(true)

  // Nickname edit state
  const [editing, setEditing] = useState(false)
  const [editValue, setEditValue] = useState('')
  const [editError, setEditError] = useState('')
  const [saving, setSaving] = useState(false)

  // Redirect if not authenticated
  useEffect(() => {
    if (!authLoading && !user) {
      navigate('/')
    }
  }, [authLoading, user, navigate])

  // Fetch stats and games
  useEffect(() => {
    if (!user) return
    void (async () => {
      setStatsLoading(true)
      try {
        const [s, g] = await Promise.all([getMyStats(), getMyGames()])
        setStats(s)
        setGames(g)
      } catch {
        // leave null — show zeros
      } finally {
        setStatsLoading(false)
      }
    })()
  }, [user])

  function startEdit() {
    setEditValue(displayName)
    setEditError('')
    setEditing(true)
  }

  function cancelEdit() {
    setEditing(false)
    setEditError('')
  }

  async function saveEdit() {
    const trimmed = editValue.trim()
    if (trimmed.length === 0 || trimmed.length > 50) {
      setEditError('닉네임은 1~50자여야 합니다.')
      return
    }
    setSaving(true)
    setEditError('')
    try {
      await updateMe(trimmed)
      useAuthStore.setState({ displayName: trimmed })
      setEditing(false)
    } catch {
      setEditError('저장에 실패했습니다.')
    } finally {
      setSaving(false)
    }
  }

  async function handleSignOut() {
    await useAuthStore.getState().signOut()
    navigate('/')
  }

  const avatarUrl = user?.user_metadata?.avatar_url as string | undefined
  const initials = displayName ? displayName[0].toUpperCase() : '?'

  const roleEntries = stats
    ? Object.entries(stats.by_role).filter(([, v]) => v.games > 0)
    : []

  return (
    <div style={{ minHeight: '100dvh', background: T.bg, color: T.text, fontFamily: SANS }}>

      {/* Nav */}
      <nav style={{
        position: 'fixed', top: 0, left: 0, right: 0, zIndex: 40,
        height: '60px', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '0 32px', background: T.bg,
        borderBottom: `1px solid ${T.surfaceBorder}`,
      }}>
        <button
          onClick={() => navigate('/lobby')}
          style={{ fontFamily: SERIF, fontSize: '20px', color: T.accent, letterSpacing: '-0.02em', background: 'none', border: 'none', cursor: 'pointer', padding: 0 }}
        >
          AI Mafia
        </button>
        <div style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted, letterSpacing: '0.1em', textTransform: 'uppercase' }}>
          PROFILE
        </div>
      </nav>

      <main style={{ maxWidth: '720px', margin: '0 auto', padding: '88px 32px 64px' }}>

        {/* ── Header: avatar + nickname + email ── */}
        <section style={{ display: 'flex', alignItems: 'flex-start', gap: '24px', marginBottom: '40px' }}>
          {/* Avatar */}
          {avatarUrl ? (
            <img
              src={avatarUrl}
              alt="profile"
              style={{ width: '64px', height: '64px', borderRadius: '50%', flexShrink: 0, border: `1px solid ${T.surfaceBorder}` }}
            />
          ) : (
            <div style={{
              width: '64px', height: '64px', borderRadius: '50%', flexShrink: 0,
              background: T.accentDim, border: `1px solid ${T.accent}40`,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontFamily: SERIF, fontSize: '24px', color: T.accent,
            }}>
              {initials}
            </div>
          )}

          {/* Name + email */}
          <div style={{ flex: 1, minWidth: 0 }}>
            {editing ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <input
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') void saveEdit(); if (e.key === 'Escape') cancelEdit() }}
                    style={{
                      flex: 1, background: T.surfaceHigh, color: T.text, fontFamily: SANS,
                      border: `1px solid ${T.accent}60`, borderRadius: '2px',
                      padding: '8px 12px', fontSize: '20px', outline: 'none',
                    }}
                    autoFocus
                    maxLength={50}
                  />
                  <button
                    onClick={() => void saveEdit()}
                    disabled={saving}
                    style={{
                      padding: '8px 16px', cursor: saving ? 'not-allowed' : 'pointer',
                      background: T.accentDim, color: T.accent,
                      border: `1px solid ${T.accent}50`, borderRadius: '2px',
                      fontFamily: SANS, fontSize: '13px', fontWeight: 600,
                      opacity: saving ? 0.6 : 1,
                    }}
                  >
                    {saving ? '저장 중...' : '저장'}
                  </button>
                  <button
                    onClick={cancelEdit}
                    style={{
                      padding: '8px 16px', cursor: 'pointer',
                      background: 'transparent', color: T.textMuted,
                      border: `1px solid ${T.surfaceBorder}`, borderRadius: '2px',
                      fontFamily: SANS, fontSize: '13px',
                    }}
                  >
                    취소
                  </button>
                </div>
                {editError && (
                  <span style={{ fontFamily: MONO, fontSize: '11px', color: T.danger }}>
                    {editError}
                  </span>
                )}
              </div>
            ) : (
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '6px' }}>
                <span style={{ fontFamily: SERIF, fontSize: '28px', letterSpacing: '-0.02em', color: T.text }}>
                  {displayName || '—'}
                </span>
                <button
                  onClick={startEdit}
                  title="닉네임 수정"
                  style={{
                    background: 'none', border: 'none', cursor: 'pointer',
                    color: T.textMuted, padding: '4px', borderRadius: '2px',
                    display: 'flex', alignItems: 'center',
                    transition: 'color 150ms ease',
                  }}
                  onMouseEnter={(e) => (e.currentTarget.style.color = T.accent)}
                  onMouseLeave={(e) => (e.currentTarget.style.color = T.textMuted)}
                >
                  {/* Pencil icon */}
                  <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                    <path d="M9.5 1.5L12.5 4.5L4.5 12.5H1.5V9.5L9.5 1.5Z" stroke="currentColor" strokeWidth="1.2" strokeLinejoin="round"/>
                    <path d="M7.5 3.5L10.5 6.5" stroke="currentColor" strokeWidth="1.2"/>
                  </svg>
                </button>
              </div>
            )}
            <span style={{ fontFamily: MONO, fontSize: '12px', color: T.textMuted }}>
              {user?.email ?? ''}
            </span>
          </div>
        </section>

        {/* ── Stats cards ── */}
        <section style={{ marginBottom: '32px' }}>
          <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.12em', display: 'block', marginBottom: '14px' }}>
            전적
          </span>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '12px' }}>
            {[
              { label: '총 게임', value: statsLoading ? '—' : String(stats?.total_games ?? 0) },
              { label: '승',      value: statsLoading ? '—' : String(stats?.wins ?? 0) },
              { label: '패',      value: statsLoading ? '—' : String(stats?.losses ?? 0) },
              { label: '승률',    value: statsLoading ? '—' : `${((stats?.win_rate ?? 0) * 100).toFixed(1)}%` },
            ].map(({ label, value }) => (
              <div key={label} style={{
                background: T.surface, border: `1px solid ${T.surfaceBorder}`,
                borderRadius: '4px', padding: '16px',
                display: 'flex', flexDirection: 'column', gap: '8px',
              }}>
                <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.1em' }}>
                  {label}
                </span>
                <span style={{ fontFamily: SERIF, fontSize: '28px', color: T.text, letterSpacing: '-0.02em' }}>
                  {value}
                </span>
              </div>
            ))}
          </div>
        </section>

        {/* ── Role stats table ── */}
        {!statsLoading && roleEntries.length > 0 && (
          <section style={{ marginBottom: '32px' }}>
            <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.12em', display: 'block', marginBottom: '14px' }}>
              역할별 통계
            </span>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr style={{ borderBottom: `1px solid ${T.surfaceBorder}` }}>
                  {['역할', '게임', '승', '패', '승률'].map((h) => (
                    <th key={h} style={{
                      fontFamily: MONO, fontSize: '10px', color: T.textMuted,
                      textTransform: 'uppercase', letterSpacing: '0.1em',
                      textAlign: 'left', padding: '8px 12px', fontWeight: 400,
                    }}>
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {roleEntries.map(([role, rs]) => (
                  <tr key={role} style={{ borderBottom: `1px solid ${T.surfaceBorder}` }}>
                    <td style={{ fontFamily: SANS, fontSize: '13px', color: T.text, padding: '12px' }}>
                      {ROLE_LABEL[role] ?? role}
                    </td>
                    <td style={{ fontFamily: MONO, fontSize: '13px', color: T.text, padding: '12px' }}>{rs.games}</td>
                    <td style={{ fontFamily: MONO, fontSize: '13px', color: T.accent, padding: '12px' }}>{rs.wins}</td>
                    <td style={{ fontFamily: MONO, fontSize: '13px', color: T.textMuted, padding: '12px' }}>{rs.games - rs.wins}</td>
                    <td style={{ fontFamily: MONO, fontSize: '13px', color: T.text, padding: '12px' }}>{(rs.win_rate * 100).toFixed(1)}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>
        )}

        {/* ── Recent games table ── */}
        <section style={{ marginBottom: '48px' }}>
          <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.12em', display: 'block', marginBottom: '14px' }}>
            최근 게임
          </span>
          {!statsLoading && games.length === 0 ? (
            <p style={{ fontFamily: MONO, fontSize: '11px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em' }}>
              게임 기록 없음
            </p>
          ) : (
            <div style={{ maxHeight: '320px', overflowY: 'auto', border: `1px solid ${T.surfaceBorder}`, borderRadius: '4px' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead style={{ position: 'sticky', top: 0, background: T.surface }}>
                  <tr style={{ borderBottom: `1px solid ${T.surfaceBorder}` }}>
                    {['날짜', '역할', '결과', '생존', '라운드', '게임시간'].map((h) => (
                      <th key={h} style={{
                        fontFamily: MONO, fontSize: '10px', color: T.textMuted,
                        textTransform: 'uppercase', letterSpacing: '0.1em',
                        textAlign: 'left', padding: '10px 12px', fontWeight: 400,
                      }}>
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {games.map((g) => (
                    <tr key={g.game_id} style={{ borderBottom: `1px solid ${T.surfaceBorder}` }}>
                      <td style={{ fontFamily: MONO, fontSize: '12px', color: T.textMuted, padding: '10px 12px' }}>{formatDate(g.played_at)}</td>
                      <td style={{ fontFamily: SANS, fontSize: '13px', color: T.text, padding: '10px 12px' }}>{ROLE_LABEL[g.role] ?? g.role}</td>
                      <td style={{ fontFamily: MONO, fontSize: '13px', color: g.won ? T.accent : T.danger, padding: '10px 12px', fontWeight: 600 }}>{g.won ? '승' : '패'}</td>
                      <td style={{ fontFamily: MONO, fontSize: '13px', color: g.survived ? T.text : T.textMuted, padding: '10px 12px' }}>{g.survived ? 'O' : 'X'}</td>
                      <td style={{ fontFamily: MONO, fontSize: '12px', color: T.textMuted, padding: '10px 12px' }}>{g.round_count}R</td>
                      <td style={{ fontFamily: MONO, fontSize: '12px', color: T.textMuted, padding: '10px 12px' }}>{formatDuration(g.duration_sec)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>

        {/* ── Bottom buttons ── */}
        <div style={{ display: 'flex', gap: '12px' }}>
          <button
            onClick={() => navigate('/lobby')}
            style={{
              padding: '11px 24px', cursor: 'pointer',
              background: T.accentDim, color: T.accent,
              border: `1px solid ${T.accent}50`, borderRadius: '2px',
              fontFamily: SANS, fontSize: '13px', fontWeight: 600,
              transition: 'background 150ms ease',
            }}
          >
            로비로 돌아가기
          </button>
          <button
            onClick={() => void handleSignOut()}
            style={{
              padding: '11px 24px', cursor: 'pointer',
              background: 'transparent', color: T.textMuted,
              border: `1px solid ${T.surfaceBorder}`, borderRadius: '2px',
              fontFamily: SANS, fontSize: '13px', fontWeight: 500,
              transition: 'all 150ms ease',
            }}
          >
            로그아웃
          </button>
        </div>
      </main>
    </div>
  )
}
