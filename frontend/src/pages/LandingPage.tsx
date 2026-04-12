import { useEffect, useRef, useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { prepare, layout } from '@chenglou/pretext'

// ── Design tokens (Case File / Industrial Noir) ─────────────────────────────
const T = {
  bg:            '#0E0C09',
  surface:       '#181410',
  surfaceHigh:   '#221E17',
  surfaceBorder: '#2E2820',
  accent:        '#C4963A',
  accentDim:     'rgba(196,150,58,0.15)',
  accentGlow:    'rgba(196,150,58,0.06)',
  text:          '#ECE7DE',
  textMuted:     '#786F62',
  textDim:       '#4A4438',
  danger:        '#8C1F1F',
  dangerDim:     'rgba(140,31,31,0.20)',
  police:        '#3D7FA8',
  policeDim:     'rgba(61,127,168,0.15)',
} as const

const FONT_SERIF = "'Instrument Serif', Georgia, serif"
const FONT_SANS  = "'DM Sans', system-ui, sans-serif"
const FONT_MONO  = "'JetBrains Mono', monospace"

const GRAIN_SVG = `url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E")`

// ── How it works steps ───────────────────────────────────────────────────────
const STEPS = [
  {
    num:   '01',
    title: '방에 입장하다',
    desc:  '공개 방에 참가하거나 코드를 받아 비밀 테이블에 합류한다. 최대 6인, 익명으로.',
  },
  {
    num:   '02',
    title: '역할을 배정받다',
    desc:  '당신은 마피아인가, 시민인가, 경찰인가. 그리고 당신 옆의 누군가는 AI다. 아무도 알려주지 않는다.',
  },
  {
    num:   '03',
    title: '추론하고 살아남다',
    desc:  '낮에는 토론하고 투표한다. 밤에는 일이 벌어진다. 거짓말을 꿰뚫어라, 또는 완벽하게 연기하라.',
  },
] as const

// ── Live feed entries ─────────────────────────────────────────────────────────
const FEED_ENTRIES = [
  { winner: '시민', round: 3, ago: '2분', players: 'Alex, Ji-ho, Sam +3명' },
  { winner: 'AI 마피아', round: 4, ago: '7분', players: 'Morgan, Riley, Quinn +3명' },
  { winner: '시민', round: 2, ago: '14분', players: 'Dana, Yuna, Chris +3명' },
  { winner: 'AI 마피아', round: 5, ago: '21분', players: 'Jordan, Park, Lee +3명' },
  { winner: '시민', round: 3, ago: '33분', players: 'Kim, Max, Seo +3명' },
  { winner: 'AI 마피아', round: 3, ago: '45분', players: 'Alex, Ji-ho, Sam +3명' },
  { winner: '시민', round: 4, ago: '58분', players: 'Choi, Rin, Tom +3명' },
] as const

// ── AI capabilities ───────────────────────────────────────────────────────────
const AI_CAPS = [
  {
    label: '언어 패턴 모방',
    en:    'Language Mimicry',
    desc:  'AI는 인간의 타이핑 패턴, 말 속도, 감정 표현까지 분석해 동일하게 재현한다. 오타도, 망설임도, 의심스러운 완벽함도 없다.',
  },
  {
    label: '전략적 기만',
    en:    'Strategic Deception',
    desc:  '마피아 역할을 맡으면 AI는 무고한 플레이어를 지목해 의심을 돌리는 전략을 실시간으로 계산한다. 감정이 아니라 확률로.',
  },
  {
    label: '역할 추론',
    en:    'Role Inference',
    desc:  '시민이나 경찰 역할에서도 AI는 대화 흐름을 분석해 누가 마피아일 가능성이 높은지 계속 업데이트한다. 당신보다 빠를 수 있다.',
  },
] as const

// ── Roles ────────────────────────────────────────────────────────────────────
const ROLES = [
  {
    num:   '01',
    label: '마피아',
    en:    'Mafia',
    desc:  '낮에는 무고한 시민인 척 토론에 참여하고, 밤에는 팀원들과 합의해 시민 한 명을 제거한다. 끝까지 들키지 않고 살아남는 것이 목표다.',
    color: T.danger,
    dimBg: T.dangerDim,
  },
  {
    num:   '02',
    label: '시민',
    en:    'Citizen',
    desc:  '토론과 투표만이 무기다. 누가 거짓말을 하는지 논리로 추적하고, 마피아를 찾아 제거하라. 진실을 밝혀내는 것이 유일한 수단이다.',
    color: T.textMuted,
    dimBg: `rgba(120,111,98,0.12)`,
  },
  {
    num:   '03',
    label: '경찰',
    en:    'Detective',
    desc:  '매일 밤 한 명을 조사해 그가 마피아인지 확인할 수 있다. 당신이 얻는 정보는 팀의 생사를 가를 수 있다. 언제 공개하고 언제 숨길지 판단하라.',
    color: T.police,
    dimBg: T.policeDim,
  },
] as const

// ── Stat counter hook ────────────────────────────────────────────────────────
function useCounter(target: number, duration = 1400, started: boolean = false) {
  const [value, setValue] = useState(0)
  useEffect(() => {
    if (!started) return
    let start: number | null = null
    const step = (ts: number) => {
      if (start === null) start = ts
      const progress = Math.min((ts - start) / duration, 1)
      const eased = 1 - Math.pow(1 - progress, 3) // ease-out cubic
      setValue(Math.round(eased * target))
      if (progress < 1) requestAnimationFrame(step)
    }
    requestAnimationFrame(step)
  }, [target, duration, started])
  return value
}

// ── Intersection observer hook ────────────────────────────────────────────────
function useInView(options?: IntersectionObserverInit) {
  const ref = useRef<HTMLDivElement>(null)
  const [inView, setInView] = useState(false)
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const obs = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) {
        setInView(true)
        obs.disconnect()
      }
    }, { threshold: 0.15, ...options })
    obs.observe(el)
    return () => obs.disconnect()
  }, [])
  return { ref, inView }
}

// ── Fade-up style helper ──────────────────────────────────────────────────────
function fadeUp(visible: boolean, delay = 0): React.CSSProperties {
  return {
    opacity:         visible ? 1 : 0,
    transform:       visible ? 'translateY(0)' : 'translateY(12px)',
    transition:      `opacity 500ms ${delay}ms ease-out, transform 500ms ${delay}ms ease-out`,
  }
}

// ── CSS keyframes injected once ───────────────────────────────────────────────
const KEYFRAMES = `
  @keyframes blink {
    0%, 100% { opacity: 1; }
    50%       { opacity: 0; }
  }
  @keyframes scanline {
    0%   { background-position: 0 0; }
    100% { background-position: 0 100%; }
  }
  @keyframes ambientShift {
    0%   { opacity: 0.04; transform: scale(1) translate(0, 0); }
    33%  { opacity: 0.06; transform: scale(1.05) translate(2%, -1%); }
    66%  { opacity: 0.03; transform: scale(0.98) translate(-1%, 2%); }
    100% { opacity: 0.04; transform: scale(1) translate(0, 0); }
  }
  @keyframes fadeIn {
    from { opacity: 0; }
    to   { opacity: 1; }
  }
`

export default function LandingPage() {
  const navigate = useNavigate()
  const { user, loading, signInWithGoogle } = useAuthStore()

  useEffect(() => {
    if (!loading && user) {
      navigate('/lobby')
    }
  }, [user, loading, navigate])

  function handleCTA() {
    if (user) {
      navigate('/lobby')
    } else {
      void signInWithGoogle()
    }
  }

  const heroRef    = useRef<HTMLHeadingElement>(null)
  const [mounted, setMounted]   = useState(false)
  const [statsVisible, setStatsVisible] = useState(false)
  const statsRef   = useRef<HTMLDivElement>(null)
  const mouseRef   = useRef({ x: 0.5, y: 0.5 })
  const glowRef    = useRef<HTMLDivElement>(null)
  const { ref: rolesRef, inView: rolesInView } = useInView()
  const { ref: stepsRef, inView: stepsInView } = useInView()
  const { ref: feedRef,  inView: feedInView  } = useInView()
  const { ref: aiRef,    inView: aiInView    } = useInView()
  const { ref: ctaRef,   inView: ctaInView   } = useInView()
  const [feedOffset, setFeedOffset] = useState(0)
  const feedTickRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // Counters
  const games    = useCounter(1247, 1600, statsVisible)
  const winRate  = useCounter(58,   1200, statsVisible)
  const rounds   = useCounter(32,   1000, statsVisible)   // ×0.1 → 3.2

  // ── Inject keyframes once ─────────────────────────────────────────────────
  useEffect(() => {
    const id = 'case-file-keyframes'
    if (!document.getElementById(id)) {
      const style = document.createElement('style')
      style.id = id
      style.textContent = KEYFRAMES
      document.head.appendChild(style)
    }
  }, [])

  // ── Load fonts ────────────────────────────────────────────────────────────
  useEffect(() => {
    const id = 'case-file-fonts'
    if (document.getElementById(id)) return
    const link = document.createElement('link')
    link.id   = id
    link.rel  = 'stylesheet'
    link.href = 'https://fonts.googleapis.com/css2?family=Instrument+Serif:ital@0;1&family=DM+Sans:ital,opsz,wght@0,9..40,300;0,9..40,400;0,9..40,500;0,9..40,600;1,9..40,300&family=JetBrains+Mono:wght@400;500&display=swap'
    document.head.appendChild(link)
  }, [])

  // ── Mount entrance (stagger trigger) ─────────────────────────────────────
  useEffect(() => {
    const t = setTimeout(() => setMounted(true), 80)
    return () => clearTimeout(t)
  }, [])

  // ── Stats visibility (IntersectionObserver) ───────────────────────────────
  useEffect(() => {
    const el = statsRef.current
    if (!el) return
    const obs = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) { setStatsVisible(true); obs.disconnect() }
    }, { threshold: 0.3 })
    obs.observe(el)
    return () => obs.disconnect()
  }, [])

  // ── Cursor-aware ambient glow ─────────────────────────────────────────────
  const handleMouseMove = useCallback((e: React.MouseEvent<HTMLElement>) => {
    const rect = e.currentTarget.getBoundingClientRect()
    mouseRef.current = {
      x: (e.clientX - rect.left) / rect.width,
      y: (e.clientY - rect.top)  / rect.height,
    }
    if (glowRef.current) {
      const { x, y } = mouseRef.current
      glowRef.current.style.background = `radial-gradient(ellipse 55% 40% at ${x * 100}% ${y * 100}%, rgba(196,150,58,0.07) 0%, transparent 70%)`
    }
  }, [])

  // ── Feed auto-scroll ─────────────────────────────────────────────────────
  useEffect(() => {
    if (!feedInView) return
    feedTickRef.current = setInterval(() => {
      setFeedOffset(prev => (prev + 1) % FEED_ENTRIES.length)
    }, 2800)
    return () => { if (feedTickRef.current) clearInterval(feedTickRef.current) }
  }, [feedInView])

  // ── Pretext: hero headline ─────────────────────────────────────────────────
  useEffect(() => {
    const el = heroRef.current
    if (!el) return
    let handle: ReturnType<typeof prepare> | null = null
    let ro: ResizeObserver | null = null

    function relayout() {
      if (!el || !handle) return
      const lh = parseFloat(getComputedStyle(el).lineHeight) || 66
      const { height } = layout(handle, el.clientWidth, lh)
      el.style.height = `${height}px`
    }

    async function init() {
      await document.fonts.ready
      if (!el) return
      const font = getComputedStyle(el).font
      handle = prepare(el.textContent ?? '', font)
      ro = new ResizeObserver(relayout)
      ro.observe(el)
      relayout()
    }

    init()
    return () => ro?.disconnect()
  }, [])

  return (
    <div
      style={{ background: T.bg, color: T.text, fontFamily: FONT_SANS, minHeight: '100dvh', position: 'relative' }}
      onMouseMove={handleMouseMove}
    >

      {/* Grain overlay */}
      <div aria-hidden="true" style={{
        position: 'fixed', inset: 0, pointerEvents: 'none', zIndex: 9999,
        opacity: 0.04, backgroundImage: GRAIN_SVG,
        backgroundSize: '128px 128px', backgroundRepeat: 'repeat',
      }} />

      {/* ── Animated scanline (very subtle, moves slowly top→bottom) */}
      <div aria-hidden="true" style={{
        position:        'fixed',
        inset:           0,
        pointerEvents:   'none',
        zIndex:          9998,
        backgroundImage: 'linear-gradient(transparent 50%, rgba(0,0,0,0.03) 50%)',
        backgroundSize:  '100% 4px',
        animation:       'scanline 12s linear infinite',
      }} />

      {/* ── Nav ─────────────────────────────────────────────────────────────── */}
      <nav style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '24px 48px', borderBottom: `1px solid ${T.surfaceBorder}`,
        animation: 'fadeIn 400ms ease-out',
      }}>
        <span style={{ fontFamily: FONT_SERIF, fontSize: 20, color: T.accent, letterSpacing: '0.12em', textTransform: 'uppercase' }}>
          AI Mafia
        </span>
        <button
          onClick={handleCTA}
          style={{
            fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.15em',
            textTransform: 'uppercase', color: T.textMuted,
            background: 'none', border: 'none', cursor: 'pointer', padding: '6px 0',
            transition: 'color 150ms ease-out',
          }}
          onMouseEnter={e => (e.currentTarget.style.color = T.text)}
          onMouseLeave={e => (e.currentTarget.style.color = T.textMuted)}
        >
          게임 하기
        </button>
      </nav>

      {/* ── Hero ─────────────────────────────────────────────────────────────── */}
      <section style={{
        minHeight:     'calc(100dvh - 73px)',
        display:       'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
        padding:       '80px 48px 0', textAlign: 'center', position: 'relative',
        overflow:      'hidden',
      }}>

        {/* Cursor-tracking ambient glow */}
        <div ref={glowRef} aria-hidden="true" style={{
          position:      'absolute', inset: 0, pointerEvents: 'none',
          background:    `radial-gradient(ellipse 55% 40% at 50% 45%, ${T.accentGlow} 0%, transparent 70%)`,
          transition:    'background 800ms ease-out',
        }} />

        {/* Slow ambient blob (auto-animates independently of cursor) */}
        <div aria-hidden="true" style={{
          position:    'absolute',
          top:         '20%', left: '50%',
          width:       600, height: 400,
          marginLeft:  -300,
          background:  `radial-gradient(ellipse at center, rgba(196,150,58,0.05) 0%, transparent 70%)`,
          animation:   'ambientShift 18s ease-in-out infinite',
          pointerEvents: 'none',
        }} />

        {/* Eyebrow */}
        <div style={{
          fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.25em',
          textTransform: 'uppercase', color: T.accent,
          marginBottom: 32, position: 'relative', display: 'flex', alignItems: 'center', gap: 10,
          ...fadeUp(mounted, 0),
        }}>
          {/* Blinking dot */}
          <span style={{
            width: 6, height: 6, borderRadius: '50%', background: T.accent, flexShrink: 0,
            animation: 'blink 2s ease-in-out infinite',
          }} />
          세션 진행 중 · AI 마피아 플랫폼
        </div>

        {/* Hero headline */}
        <h1
          ref={heroRef}
          style={{
            fontFamily: FONT_SERIF,
            fontSize: 'clamp(40px, 6vw, 64px)',
            lineHeight: 1.05,
            letterSpacing: '-0.02em',
            color: T.text,
            maxWidth: 700,
            marginBottom: 24,
            position: 'relative',
            overflow: 'hidden',
            ...fadeUp(mounted, 120),
          }}
        >
          이 자리에 앉은 누군가는,{' '}
          <em style={{ color: T.accent, fontStyle: 'italic' }}>인간이 아니다.</em>
        </h1>

        {/* Subtext */}
        <p style={{
          fontSize: 15, lineHeight: 1.7, color: T.textMuted,
          maxWidth: 400, marginBottom: 48, position: 'relative',
          ...fadeUp(mounted, 240),
        }}>
          여섯 명의 플레이어. 하나의 진실. 알고리즘과 본능, 어느 쪽이 더 설득력 있는가?
        </p>

        {/* CTAs */}
        <div style={{ display: 'flex', gap: 16, position: 'relative', ...fadeUp(mounted, 360) }}>
          <button
            onClick={handleCTA}
            style={{
              fontFamily: FONT_MONO, fontSize: 12, letterSpacing: '0.18em',
              textTransform: 'uppercase', color: T.bg, background: T.accent,
              border: 'none', padding: '14px 40px', borderRadius: 2,
              cursor: 'pointer', transition: 'opacity 150ms ease-out',
            }}
            onMouseEnter={e => (e.currentTarget.style.opacity = '0.85')}
            onMouseLeave={e => (e.currentTarget.style.opacity = '1')}
          >
            방 입장하기
          </button>
          <button
            onClick={handleCTA}
            style={{
              fontFamily: FONT_MONO, fontSize: 12, letterSpacing: '0.18em',
              textTransform: 'uppercase', color: T.textMuted, background: 'transparent',
              border: `1px solid ${T.surfaceBorder}`, padding: '14px 40px', borderRadius: 2,
              cursor: 'pointer', transition: 'border-color 150ms ease-out, color 150ms ease-out',
            }}
            onMouseEnter={e => { e.currentTarget.style.borderColor = T.accent; e.currentTarget.style.color = T.accent }}
            onMouseLeave={e => { e.currentTarget.style.borderColor = T.surfaceBorder; e.currentTarget.style.color = T.textMuted }}
          >
            코드로 참가
          </button>
        </div>

        {/* Stats — count up on scroll into view */}
        <div
          ref={statsRef}
          style={{
            display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)',
            marginTop: 80, width: '100%', maxWidth: 600,
            borderTop: `1px solid ${T.surfaceBorder}`,
            background: T.surface, borderRadius: 4, overflow: 'hidden',
            position: 'relative', ...fadeUp(mounted, 480),
          }}
        >
          {[
            { num: games.toLocaleString(),         label: '게임 완료' },
            { num: `${winRate}%`,                  label: 'AI 승률' },
            { num: `${(rounds / 10).toFixed(1)}`,  label: '평균 라운드' },
          ].map((s, i) => (
            <div key={s.label} style={{
              padding: '20px 24px', textAlign: 'left',
              borderRight: i < 2 ? `1px solid ${T.surfaceBorder}` : 'none',
              borderBottom: `1px solid ${T.surfaceBorder}`,
            }}>
              <span style={{ fontFamily: FONT_MONO, fontSize: 28, color: T.accent, display: 'block', marginBottom: 4 }}>
                {s.num}
              </span>
              <span style={{ fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.1em', textTransform: 'uppercase', color: T.textMuted }}>
                {s.label}
              </span>
            </div>
          ))}
        </div>
      </section>

      {/* ── Role dossier entries ─────────────────────────────────────────────── */}
      <section
        ref={rolesRef}
        style={{ maxWidth: 1200, margin: '80px auto 0', padding: '0 48px 96px' }}
      >
        {/* Section label */}
        <div style={{
          fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.2em',
          textTransform: 'uppercase', color: T.textMuted, marginBottom: 40,
          display: 'flex', alignItems: 'center', gap: 16,
          ...fadeUp(rolesInView, 0),
        }}>
          역할 파일
          <div style={{ flex: 1, height: 1, background: T.surfaceBorder }} />
        </div>

        {/* Role rows */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
          {ROLES.map((role, i) => (
            <div
              key={role.num}
              style={{
                display: 'grid', gridTemplateColumns: '64px 140px 1fr',
                background: T.surface,
                borderBottom: i < ROLES.length - 1 ? `1px solid ${T.surfaceBorder}` : 'none',
                borderRadius: i === 0 ? '4px 4px 0 0' : i === ROLES.length - 1 ? '0 0 4px 4px' : 0,
                overflow: 'hidden',
                transition: 'background 200ms ease-out',
                ...fadeUp(rolesInView, i * 100),
              }}
              onMouseEnter={e => (e.currentTarget.style.background = T.surfaceHigh)}
              onMouseLeave={e => (e.currentTarget.style.background = T.surface)}
            >
              {/* Row number */}
              <div style={{
                padding: '24px 16px', borderRight: `1px solid ${T.surfaceBorder}`,
                display: 'flex', alignItems: 'flex-start', justifyContent: 'center',
              }}>
                <span style={{ fontFamily: FONT_MONO, fontSize: 10, color: T.textDim }}>{role.num}</span>
              </div>

              {/* Role label */}
              <div style={{ padding: '24px', borderRight: `1px solid ${T.surfaceBorder}`, display: 'flex', flexDirection: 'column', gap: 6 }}>
                <span style={{ fontFamily: FONT_SERIF, fontSize: 20, color: role.color }}>{role.label}</span>
                <span style={{ fontFamily: FONT_MONO, fontSize: 10, letterSpacing: '0.15em', textTransform: 'uppercase', color: T.textDim }}>
                  {role.en}
                </span>
              </div>

              {/* Description */}
              <div style={{ padding: '24px 32px', display: 'flex', alignItems: 'center' }}>
                <p style={{ fontSize: 14, lineHeight: 1.65, color: T.textMuted, margin: 0 }}>{role.desc}</p>
              </div>
            </div>
          ))}
        </div>

        {/* Footer note */}
        <p style={{
          fontFamily: FONT_MONO, fontSize: 11, color: T.textDim,
          letterSpacing: '0.08em', marginTop: 24, textAlign: 'right',
          ...fadeUp(rolesInView, 400),
        }}>
          AI 플레이어는 게임 종료 후 공개됩니다.
        </p>
      </section>

      {/* ── How it works ──────────────────────────────────────────────────────── */}
      <section
        ref={stepsRef}
        style={{
          maxWidth: 1200, margin: '0 auto', padding: '96px 48px',
          borderTop: `1px solid ${T.surfaceBorder}`,
        }}
      >
        <div style={{
          fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.2em',
          textTransform: 'uppercase', color: T.textMuted, marginBottom: 56,
          display: 'flex', alignItems: 'center', gap: 16,
          ...fadeUp(stepsInView, 0),
        }}>
          진행 방식
          <div style={{ flex: 1, height: 1, background: T.surfaceBorder }} />
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 1 }}>
          {STEPS.map((step, i) => (
            <div
              key={step.num}
              style={{
                padding: '32px',
                borderRight: i < 2 ? `1px solid ${T.surfaceBorder}` : 'none',
                background: T.surface,
                borderRadius: i === 0 ? '4px 0 0 4px' : i === 2 ? '0 4px 4px 0' : 0,
                transition: 'background 200ms ease-out',
                ...fadeUp(stepsInView, i * 100),
              }}
              onMouseEnter={e => (e.currentTarget.style.background = T.surfaceHigh)}
              onMouseLeave={e => (e.currentTarget.style.background = T.surface)}
            >
              <span style={{ fontFamily: FONT_MONO, fontSize: 10, color: T.textDim, display: 'block', marginBottom: 16 }}>
                {step.num}
              </span>
              <h3 style={{
                fontFamily: FONT_SERIF, fontSize: 22, color: T.text,
                marginBottom: 16, lineHeight: 1.2,
              }}>
                {step.title}
              </h3>
              <p style={{ fontSize: 14, lineHeight: 1.7, color: T.textMuted, margin: 0 }}>
                {step.desc}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* ── Live game feed ────────────────────────────────────────────────────── */}
      <section
        ref={feedRef}
        style={{
          borderTop: `1px solid ${T.surfaceBorder}`,
          padding: '64px 0',
          overflow: 'hidden',
        }}
      >
        <div style={{ maxWidth: 1200, margin: '0 auto', padding: '0 48px' }}>
          <div style={{
            fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.2em',
            textTransform: 'uppercase', color: T.textMuted, marginBottom: 32,
            display: 'flex', alignItems: 'center', gap: 16,
            ...fadeUp(feedInView, 0),
          }}>
            <span style={{
              width: 6, height: 6, borderRadius: '50%', background: '#3A8A4A', flexShrink: 0,
              animation: 'blink 2s ease-in-out infinite',
            }} />
            실시간 게임 기록
            <div style={{ flex: 1, height: 1, background: T.surfaceBorder }} />
          </div>

          {/* Scrolling feed — shows a window of 4 entries, auto-advances */}
          <div style={{
            display: 'flex', flexDirection: 'column', gap: 1,
            ...fadeUp(feedInView, 100),
          }}>
            {Array.from({ length: 4 }, (_, i) => {
              const entry = FEED_ENTRIES[(feedOffset + i) % FEED_ENTRIES.length]
              const isMafiaWin = entry.winner.includes('마피아')
              return (
                <div
                  key={i}
                  style={{
                    display: 'grid', gridTemplateColumns: '160px 80px 1fr auto',
                    alignItems: 'center', gap: 0,
                    background: T.surface,
                    borderBottom: i < 3 ? `1px solid ${T.surfaceBorder}` : 'none',
                    borderRadius: i === 0 ? '4px 4px 0 0' : i === 3 ? '0 0 4px 4px' : 0,
                    opacity: i === 0 ? 1 : i === 1 ? 0.9 : i === 2 ? 0.75 : 0.5,
                    transition: 'opacity 400ms ease-out',
                    overflow: 'hidden',
                  }}
                >
                  {/* Winner */}
                  <div style={{ padding: '16px 20px', borderRight: `1px solid ${T.surfaceBorder}` }}>
                    <span style={{
                      fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.1em',
                      textTransform: 'uppercase',
                      color: isMafiaWin ? '#C45A5A' : T.accent,
                      background: isMafiaWin ? T.dangerDim : T.accentDim,
                      padding: '3px 8px', borderRadius: 2,
                    }}>
                      {entry.winner} 승리
                    </span>
                  </div>

                  {/* Round */}
                  <div style={{ padding: '16px 16px', borderRight: `1px solid ${T.surfaceBorder}` }}>
                    <span style={{ fontFamily: FONT_MONO, fontSize: 12, color: T.textMuted }}>
                      {entry.round}R
                    </span>
                  </div>

                  {/* Players */}
                  <div style={{ padding: '16px 20px' }}>
                    <span style={{ fontSize: 13, color: T.textMuted }}>{entry.players}</span>
                  </div>

                  {/* Time ago */}
                  <div style={{ padding: '16px 20px', borderLeft: `1px solid ${T.surfaceBorder}` }}>
                    <span style={{ fontFamily: FONT_MONO, fontSize: 11, color: T.textDim }}>
                      {entry.ago} 전
                    </span>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      </section>

      {/* ── AI capability callout ─────────────────────────────────────────────── */}
      <section
        ref={aiRef}
        style={{
          borderTop: `1px solid ${T.surfaceBorder}`,
          padding: '96px 48px',
          maxWidth: 1200, margin: '0 auto',
        }}
      >
        <div style={{
          fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.2em',
          textTransform: 'uppercase', color: T.textMuted, marginBottom: 48,
          display: 'flex', alignItems: 'center', gap: 16,
          ...fadeUp(aiInView, 0),
        }}>
          AI 플레이어
          <div style={{ flex: 1, height: 1, background: T.surfaceBorder }} />
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 48, alignItems: 'start', ...fadeUp(aiInView, 80) }}>
          {/* Left: big statement */}
          <div>
            <h2 style={{
              fontFamily: FONT_SERIF,
              fontSize: 'clamp(32px, 4vw, 48px)',
              lineHeight: 1.1,
              letterSpacing: '-0.02em',
              color: T.text,
              marginBottom: 24,
            }}>
              AI는 당신과{' '}
              <em style={{ color: T.accent, fontStyle: 'italic' }}>동일한 방식으로</em>{' '}
              생각한다.
            </h2>
            <p style={{ fontSize: 15, lineHeight: 1.7, color: T.textMuted, maxWidth: 400 }}>
              단순한 규칙 기반 봇이 아니다. 대화를 분석하고, 전략을 계산하고,
              심리적으로 압박한다. 마피아 역할에서는 무고한 척 연기하고,
              시민 역할에서는 진짜 단서를 추적한다.
            </p>
            <div style={{ marginTop: 32 }}>
              <span style={{
                fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.15em',
                textTransform: 'uppercase', color: T.textDim, display: 'block', marginBottom: 8,
              }}>
                탐지율 (인간 기준)
              </span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                <div style={{ flex: 1, height: 2, background: T.surfaceBorder, borderRadius: 1, overflow: 'hidden' }}>
                  <div style={{
                    height: 2, background: T.accent, borderRadius: 1,
                    width: aiInView ? '42%' : '0%',
                    transition: 'width 1200ms 400ms ease-out',
                  }} />
                </div>
                <span style={{ fontFamily: FONT_MONO, fontSize: 13, color: T.accent }}>42%</span>
              </div>
              <span style={{ fontFamily: FONT_MONO, fontSize: 11, color: T.textDim, marginTop: 6, display: 'block' }}>
                플레이어 중 42%만이 AI를 제때 간파했다
              </span>
            </div>
          </div>

          {/* Right: capability rows */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {AI_CAPS.map((cap, i) => (
              <div
                key={cap.label}
                style={{
                  padding: '24px',
                  background: T.surface,
                  borderRadius: i === 0 ? '4px 4px 0 0' : i === 2 ? '0 0 4px 4px' : 0,
                  borderBottom: i < 2 ? `1px solid ${T.surfaceBorder}` : 'none',
                  transition: 'background 200ms ease-out',
                  ...fadeUp(aiInView, 150 + i * 80),
                }}
                onMouseEnter={e => (e.currentTarget.style.background = T.surfaceHigh)}
                onMouseLeave={e => (e.currentTarget.style.background = T.surface)}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', marginBottom: 10 }}>
                  <span style={{ fontFamily: FONT_SERIF, fontSize: 17, color: T.text }}>{cap.label}</span>
                  <span style={{ fontFamily: FONT_MONO, fontSize: 10, color: T.textDim, letterSpacing: '0.12em', textTransform: 'uppercase' }}>{cap.en}</span>
                </div>
                <p style={{ fontSize: 13, lineHeight: 1.65, color: T.textMuted, margin: 0 }}>{cap.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ── CTA footer ───────────────────────────────────────────────────────── */}
      <section
        ref={ctaRef}
        style={{
          borderTop: `1px solid ${T.surfaceBorder}`,
          padding: '120px 48px',
          textAlign: 'center',
          background: T.surface,
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        {/* Ambient glow behind CTA */}
        <div aria-hidden="true" style={{
          position: 'absolute', inset: 0, pointerEvents: 'none',
          background: `radial-gradient(ellipse 50% 60% at 50% 50%, ${T.accentGlow} 0%, transparent 70%)`,
        }} />

        <div style={{ position: 'relative', ...fadeUp(ctaInView, 0) }}>
          <div style={{
            fontFamily: FONT_MONO, fontSize: 11, letterSpacing: '0.25em',
            textTransform: 'uppercase', color: T.accent, marginBottom: 32,
          }}>
            지금 시작
          </div>
          <h2 style={{
            fontFamily: FONT_SERIF,
            fontSize: 'clamp(36px, 5vw, 56px)',
            lineHeight: 1.05,
            letterSpacing: '-0.02em',
            color: T.text,
            marginBottom: 24,
          }}>
            <em style={{ fontStyle: 'italic', color: T.accent }}>진짜</em>를 가려낼 수 있겠는가?
          </h2>
          <p style={{ fontSize: 15, color: T.textMuted, marginBottom: 48, maxWidth: 380, margin: '0 auto 48px' }}>
            방을 만들고, 플레이어를 초대하거나, 공개 테이블에 합류하라. 게임은 이미 시작됐다.
          </p>
          <button
            onClick={handleCTA}
            style={{
              fontFamily: FONT_MONO, fontSize: 13, letterSpacing: '0.2em',
              textTransform: 'uppercase', color: T.bg, background: T.accent,
              border: 'none', padding: '18px 56px', borderRadius: 2,
              cursor: 'pointer', transition: 'opacity 150ms ease-out',
            }}
            onMouseEnter={e => (e.currentTarget.style.opacity = '0.85')}
            onMouseLeave={e => (e.currentTarget.style.opacity = '1')}
          >
            게임 시작하기
          </button>
        </div>

        {/* Bottom signature */}
        <div style={{
          fontFamily: FONT_MONO, fontSize: 10, color: T.textDim,
          letterSpacing: '0.12em', marginTop: 64, position: 'relative',
        }}>
          AI MAFIA · AI 플레이어는 게임 종료 후 공개됩니다
        </div>
      </section>

    </div>
  )
}
