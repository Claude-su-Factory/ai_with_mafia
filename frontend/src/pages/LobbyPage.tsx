import { useEffect, useRef, useState, type CSSProperties } from 'react'
import { useNavigate } from 'react-router-dom'
import { listRooms, joinRoom, createRoom, joinByCode } from '../api'
import type { Room } from '../types'
import { prepare, layout } from '@chenglou/pretext'

// ─── Design tokens ────────────────────────────────────────────────────────────
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

const STATUS_CFG = {
  waiting:  { label: 'WAITING',  color: T.accent,    bg: T.accentDim  },
  playing:  { label: 'PLAYING',  color: T.police,    bg: T.policeDim  },
  finished: { label: 'FINISHED', color: T.textMuted, bg: 'transparent' },
} as const

// ─── Shared style fragments ───────────────────────────────────────────────────
const labelSt: CSSProperties = {
  fontFamily: MONO, fontSize: '10px', color: T.textMuted,
  textTransform: 'uppercase', letterSpacing: '0.12em',
  marginBottom: '14px', display: 'block',
}
const inputSt: CSSProperties = {
  width: '100%', display: 'block',
  background: T.surfaceHigh, color: T.text, fontFamily: SANS,
  border: `1px solid ${T.surfaceBorder}`, borderRadius: '2px',
  padding: '10px 12px', fontSize: '13px', marginBottom: '8px',
  transition: 'border-color 150ms ease',
  outline: 'none',
}
const errSt: CSSProperties = {
  fontFamily: MONO, fontSize: '11px', color: T.danger, marginBottom: '8px', display: 'block',
}

// ─── CSS injected once ────────────────────────────────────────────────────────
const INJECTED_ID = 'case-file-lobby-css'
function injectCSS() {
  if (document.getElementById(INJECTED_ID)) return
  const s = document.createElement('style')
  s.id = INJECTED_ID
  s.textContent = `
    body { margin: 0; }
    body::before {
      content: ''; position: fixed; inset: 0; pointer-events: none; z-index: 9999;
      opacity: 0.04;
      background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
      background-size: 128px;
    }
    #lobby-code-input { text-transform: uppercase; text-align: center; font-family: ${MONO}; letter-spacing: 0.2em; font-size: 15px; }
    input:focus { border-color: ${T.accent} !important; }
    .lobby-row:hover { background: ${T.surfaceHigh} !important; }
    .lobby-btn-refresh:hover { color: ${T.text} !important; border-color: ${T.surfaceBorder} !important; }
    .lobby-btn-join:hover { background: rgba(196,150,58,0.2) !important; }
    .lobby-btn-create:hover { background: rgba(196,150,58,0.2) !important; }
    .lobby-btn-code:hover { border-color: ${T.textMuted} !important; color: ${T.text} !important; }
    .modal-cancel:hover { border-color: ${T.textMuted} !important; color: ${T.text} !important; }
    .modal-join:hover { background: rgba(196,150,58,0.2) !important; }
    @keyframes fadeRow { from { opacity: 0; transform: translateX(-4px); } to { opacity: 1; transform: translateX(0); } }
    @keyframes slideModal { from { opacity: 0; transform: scale(0.97) translateY(10px); } to { opacity: 1; transform: scale(1) translateY(0); } }
    ::-webkit-scrollbar { width: 4px; }
    ::-webkit-scrollbar-track { background: ${T.surface}; }
    ::-webkit-scrollbar-thumb { background: ${T.surfaceBorder}; border-radius: 2px; }
  `
  document.head.appendChild(s)
}

// ─── Component ────────────────────────────────────────────────────────────────
export default function LobbyPage() {
  const navigate = useNavigate()

  // Room list
  const [rooms, setRooms] = useState<Room[]>([])
  const [loading, setLoading] = useState(true)
  const [listError, setListError] = useState('')

  // Join modal
  const [joiningRoom, setJoiningRoom] = useState<Room | null>(null)
  const [joinName, setJoinName] = useState('')
  const [joinError, setJoinError] = useState('')

  // Create room form
  const [createName, setCreateName] = useState('')
  const [createVisibility, setCreateVisibility] = useState<'public' | 'private'>('public')
  const [createPlayerName, setCreatePlayerName] = useState('')
  const [createError, setCreateError] = useState('')

  // Code join form
  const [codeInput, setCodeInput] = useState('')
  const [codePlayerName, setCodePlayerName] = useState('')
  const [codeError, setCodeError] = useState('')

  // Pretext — section heading
  const headingRef = useRef<HTMLHeadingElement>(null)
  const headingHandle = useRef<ReturnType<typeof prepare> | null>(null)

  // CSS + Pretext init
  useEffect(() => {
    injectCSS()

    const el = headingRef.current
    if (!el) return
    let ro: ResizeObserver | null = null

    document.fonts.ready.then(() => {
      headingHandle.current = prepare(el.textContent ?? '', getComputedStyle(el).font)
      function relayout() {
        if (!headingHandle.current || !el) return
        const lh = parseFloat(getComputedStyle(el).lineHeight)
        const { height } = layout(headingHandle.current, el.clientWidth, lh)
        el.style.height = `${height}px`
      }
      relayout()
      ro = new ResizeObserver(relayout)
      ro.observe(el)
    })

    return () => ro?.disconnect()
  }, [])

  // Initial fetch
  useEffect(() => { fetchRooms() }, [])

  async function fetchRooms() {
    setLoading(true)
    setListError('')
    try {
      const data = await listRooms()
      setRooms(data ?? [])
    } catch {
      setListError('방 목록을 불러오지 못했습니다.')
    } finally {
      setLoading(false)
    }
  }

  async function handleJoinRoom() {
    if (!joiningRoom || !joinName.trim()) return
    setJoinError('')
    try {
      const res = await joinRoom({ room_id: joiningRoom.id, player_name: joinName.trim() })
      localStorage.setItem(`player_id_${joiningRoom.id}`, res.player_id)
      navigate(`/rooms/${joiningRoom.id}`)
    } catch (e: unknown) {
      setJoinError(e instanceof Error ? e.message : '참가에 실패했습니다.')
    }
  }

  async function handleCreateRoom() {
    if (!createName.trim() || !createPlayerName.trim()) return
    setCreateError('')
    try {
      const res = await createRoom({
        name: createName.trim(),
        visibility: createVisibility,
        player_name: createPlayerName.trim(),
      })
      localStorage.setItem(`player_id_${res.id}`, res.player_id)
      navigate(`/rooms/${res.id}`)
    } catch (e: unknown) {
      setCreateError(e instanceof Error ? e.message : '방 생성에 실패했습니다.')
    }
  }

  async function handleJoinByCode() {
    if (!codeInput.trim() || !codePlayerName.trim()) return
    setCodeError('')
    try {
      const res = await joinByCode({ code: codeInput.trim(), player_name: codePlayerName.trim() })
      localStorage.setItem(`player_id_${res.id}`, res.player_id)
      navigate(`/rooms/${res.id}`)
    } catch (e: unknown) {
      setCodeError(e instanceof Error ? e.message : '코드 참가에 실패했습니다.')
    }
  }

  function openJoin(room: Room) {
    setJoiningRoom(room)
    setJoinName('')
    setJoinError('')
  }

  return (
    <div style={{ minHeight: '100vh', background: T.bg, color: T.text, fontFamily: SANS }}>

      {/* ── Nav ────────────────────────────────────────────────────────── */}
      <nav style={{
        position: 'fixed', top: 0, left: 0, right: 0, zIndex: 40,
        height: '60px', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '0 32px', background: T.bg,
        borderBottom: `1px solid ${T.surfaceBorder}`,
      }}>
        <button
          onClick={() => navigate('/')}
          style={{ fontFamily: SERIF, fontSize: '20px', color: T.accent, letterSpacing: '-0.02em', background: 'none', border: 'none', cursor: 'pointer', padding: 0 }}
        >
          AI Mafia
        </button>
        <div style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted, letterSpacing: '0.1em', textTransform: 'uppercase' }}>
          LOBBY
        </div>
      </nav>

      {/* ── Page layout ────────────────────────────────────────────────── */}
      <main style={{
        maxWidth: '1200px', margin: '0 auto',
        padding: '88px 32px 64px',
        display: 'flex', gap: '48px', alignItems: 'flex-start',
      }}>

        {/* ── Left sidebar ─────────────────────────────────────────────── */}
        <aside style={{ width: '320px', flexShrink: 0 }}>

          {/* ── Create room ──────────────────────────────────────────── */}
          <section>
            <span style={labelSt}>새 방 만들기</span>

            <input
              style={inputSt}
              placeholder="방 이름"
              value={createName}
              onChange={(e) => setCreateName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCreateRoom()}
            />
            <input
              style={inputSt}
              placeholder="닉네임"
              value={createPlayerName}
              onChange={(e) => setCreatePlayerName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleCreateRoom()}
            />

            {/* Visibility pills */}
            <div style={{ display: 'flex', gap: '6px', marginBottom: '12px' }}>
              {(['public', 'private'] as const).map((v) => (
                <button
                  key={v}
                  onClick={() => setCreateVisibility(v)}
                  style={{
                    flex: 1, padding: '7px 0', cursor: 'pointer',
                    borderRadius: '2px', transition: 'all 150ms ease',
                    fontFamily: MONO, fontSize: '10px', textTransform: 'uppercase', letterSpacing: '0.08em',
                    background: createVisibility === v ? T.accentDim : 'transparent',
                    color: createVisibility === v ? T.accent : T.textMuted,
                    border: `1px solid ${createVisibility === v ? `${T.accent}60` : T.surfaceBorder}`,
                  }}
                >
                  {v === 'public' ? '공개' : '비공개'}
                </button>
              ))}
            </div>

            {createError && <span style={errSt}>{createError}</span>}
            <button
              className="lobby-btn-create"
              onClick={handleCreateRoom}
              style={{
                width: '100%', padding: '11px 0', cursor: 'pointer',
                borderRadius: '2px', transition: 'all 150ms ease',
                background: T.accentDim, color: T.accent,
                border: `1px solid ${T.accent}50`,
                fontFamily: SANS, fontSize: '13px', fontWeight: 600,
              }}
            >
              방 만들기
            </button>
          </section>

          <div style={{ height: '1px', background: T.surfaceBorder, margin: '28px 0' }} />

          {/* ── Code join ────────────────────────────────────────────── */}
          <section>
            <span style={labelSt}>초대 코드로 참가</span>

            <input
              id="lobby-code-input"
              style={inputSt}
              placeholder="XXXXXX"
              value={codeInput}
              onChange={(e) => setCodeInput(e.target.value.toUpperCase())}
              maxLength={8}
              onKeyDown={(e) => e.key === 'Enter' && handleJoinByCode()}
            />
            <input
              style={inputSt}
              placeholder="닉네임"
              value={codePlayerName}
              onChange={(e) => setCodePlayerName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleJoinByCode()}
            />

            {codeError && <span style={errSt}>{codeError}</span>}
            <button
              className="lobby-btn-code"
              onClick={handleJoinByCode}
              style={{
                width: '100%', padding: '11px 0', cursor: 'pointer',
                borderRadius: '2px', transition: 'all 150ms ease',
                background: 'transparent', color: T.textMuted,
                border: `1px solid ${T.surfaceBorder}`,
                fontFamily: SANS, fontSize: '13px', fontWeight: 500,
              }}
            >
              코드로 참가
            </button>
          </section>
        </aside>

        {/* ── Room list ────────────────────────────────────────────────── */}
        <section style={{ flex: 1, minWidth: 0 }}>

          {/* Header row */}
          <div style={{
            display: 'flex', alignItems: 'flex-end', justifyContent: 'space-between',
            marginBottom: '0', paddingBottom: '16px',
            borderBottom: `1px solid ${T.surfaceBorder}`,
          }}>
            <div>
              <span style={labelSt}>공개 게임</span>
              <h2
                ref={headingRef}
                style={{ fontFamily: SERIF, fontSize: '28px', color: T.text, margin: 0, lineHeight: '1.25', letterSpacing: '-0.02em' }}
              >
                방을 선택하거나, 직접 만드세요.
              </h2>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: '14px', paddingBottom: '4px' }}>
              <span style={{ fontFamily: MONO, fontSize: '12px', color: T.textDim }}>
                {loading ? '—' : `${rooms.length} ROOMS`}
              </span>
              <button
                className="lobby-btn-refresh"
                onClick={fetchRooms}
                style={{
                  fontFamily: MONO, fontSize: '10px', textTransform: 'uppercase', letterSpacing: '0.1em',
                  color: T.textMuted, background: 'none',
                  border: `1px solid ${T.surfaceBorder}`, borderRadius: '2px',
                  padding: '5px 10px', cursor: 'pointer', transition: 'all 100ms ease',
                }}
              >
                새로고침
              </button>
            </div>
          </div>

          {/* Loading state */}
          {loading && (
            <div style={{ padding: '48px 0', fontFamily: MONO, fontSize: '11px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em' }}>
              LOADING...
            </div>
          )}

          {/* Error state */}
          {!loading && listError && (
            <div style={{
              margin: '16px 0', padding: '12px 16px', borderRadius: '2px',
              background: 'rgba(140,31,31,0.08)', border: `1px solid rgba(140,31,31,0.25)`,
              fontFamily: MONO, fontSize: '12px', color: T.danger,
            }}>
              {listError}
            </div>
          )}

          {/* Empty state */}
          {!loading && !listError && rooms.length === 0 && (
            <div style={{ padding: '48px 0' }}>
              <p style={{ fontFamily: MONO, fontSize: '11px', color: T.textDim, textTransform: 'uppercase', letterSpacing: '0.1em', marginBottom: '8px' }}>
                공개된 방 없음
              </p>
              <p style={{ fontFamily: SANS, fontSize: '13px', color: T.textMuted, margin: 0 }}>
                첫 번째 방을 만들어 게임을 시작해보세요.
              </p>
            </div>
          )}

          {/* Room ledger */}
          {!loading && rooms.length > 0 && (
            <div>
              {rooms.map((room, i) => {
                const s = STATUS_CFG[room.status] ?? STATUS_CFG.waiting
                return (
                  <div
                    key={room.id}
                    className="lobby-row"
                    onClick={() => openJoin(room)}
                    style={{
                      display: 'flex', alignItems: 'center', gap: '12px',
                      padding: '14px 8px',
                      borderBottom: `1px solid ${T.surfaceBorder}`,
                      background: 'transparent', cursor: 'pointer',
                      transition: 'background 80ms ease',
                      animation: `fadeRow 200ms ${i * 40}ms ease both`,
                    }}
                  >
                    {/* Row number */}
                    <span style={{ fontFamily: MONO, fontSize: '11px', color: T.textDim, minWidth: '22px', flexShrink: 0 }}>
                      {String(i + 1).padStart(2, '0')}
                    </span>

                    {/* Status dot */}
                    <span style={{
                      width: '5px', height: '5px', borderRadius: '50%', flexShrink: 0,
                      background: room.status === 'waiting' ? '#3A6A3A' : room.status === 'playing' ? T.police : T.textDim,
                    }} />

                    {/* Room name */}
                    <span style={{ flex: 1, fontFamily: SANS, fontSize: '14px', color: T.text, letterSpacing: '-0.01em', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {room.name}
                    </span>

                    {/* Player count */}
                    <span style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted, flexShrink: 0 }}>
                      {room.players.length}명
                    </span>

                    {/* Status badge */}
                    <span style={{
                      fontFamily: MONO, fontSize: '10px', textTransform: 'uppercase', letterSpacing: '0.08em',
                      color: s.color, background: s.bg,
                      padding: '3px 8px', borderRadius: '2px',
                      border: `1px solid ${s.color}35`,
                      flexShrink: 0,
                    }}>
                      {s.label}
                    </span>

                    {/* Join button — only for waiting rooms */}
                    {room.status === 'waiting' ? (
                      <button
                        className="lobby-btn-join"
                        onClick={(e) => { e.stopPropagation(); openJoin(room) }}
                        style={{
                          fontFamily: MONO, fontSize: '10px', textTransform: 'uppercase', letterSpacing: '0.08em',
                          color: T.accent, background: T.accentDim,
                          border: `1px solid ${T.accent}45`,
                          padding: '5px 12px', borderRadius: '2px', cursor: 'pointer',
                          flexShrink: 0, transition: 'all 100ms ease',
                        }}
                      >
                        참가
                      </button>
                    ) : (
                      <span style={{ minWidth: '57px', flexShrink: 0 }} />
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </section>
      </main>

      {/* ── Join modal ──────────────────────────────────────────────────── */}
      {joiningRoom && (
        <div
          role="dialog"
          aria-modal="true"
          style={{
            position: 'fixed', inset: 0, zIndex: 50,
            background: 'rgba(14,12,9,0.88)',
            backdropFilter: 'blur(6px)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}
          onClick={(e) => e.target === e.currentTarget && setJoiningRoom(null)}
        >
          <div style={{
            background: T.surface,
            border: `1px solid ${T.surfaceBorder}`,
            borderRadius: '4px',
            padding: '32px',
            width: '340px',
            animation: 'slideModal 200ms ease both',
          }}>
            <span style={{ fontFamily: MONO, fontSize: '10px', color: T.textMuted, textTransform: 'uppercase', letterSpacing: '0.12em', display: 'block', marginBottom: '10px' }}>
              방 참가
            </span>
            <h3 style={{ fontFamily: SERIF, fontSize: '22px', color: T.text, margin: '0 0 4px', lineHeight: 1.25 }}>
              {joiningRoom.name}
            </h3>
            <p style={{ fontFamily: MONO, fontSize: '11px', color: T.textMuted, margin: '0 0 24px' }}>
              {joiningRoom.players.length}명 참가 중
            </p>

            <input
              style={inputSt}
              placeholder="닉네임을 입력하세요"
              value={joinName}
              onChange={(e) => setJoinName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleJoinRoom()}
              autoFocus
            />
            {joinError && <span style={{ ...errSt, marginTop: '4px' }}>{joinError}</span>}

            <div style={{ display: 'flex', gap: '8px', marginTop: '16px' }}>
              <button
                className="modal-cancel"
                onClick={() => setJoiningRoom(null)}
                style={{
                  flex: 1, padding: '10px 0', cursor: 'pointer',
                  borderRadius: '2px', transition: 'all 150ms ease',
                  background: 'transparent', color: T.textMuted,
                  border: `1px solid ${T.surfaceBorder}`,
                  fontFamily: SANS, fontSize: '13px', fontWeight: 500,
                }}
              >
                취소
              </button>
              <button
                className="modal-join"
                onClick={handleJoinRoom}
                style={{
                  flex: 1, padding: '10px 0', cursor: 'pointer',
                  borderRadius: '2px', transition: 'all 150ms ease',
                  background: T.accentDim, color: T.accent,
                  border: `1px solid ${T.accent}50`,
                  fontFamily: SANS, fontSize: '13px', fontWeight: 600,
                }}
              >
                입장
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
