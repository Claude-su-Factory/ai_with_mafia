# UX/Animation Improvements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 뷰포트 침범 수정, 토론 패널 collapse, 풀스크린 버튼, 페이즈 전환/사망 발표용 CinematicOverlay 시스템을 구현한다.

**Architecture:** 순수 CSS `100dvh` 치환 + React 컴포넌트 추가. Zustand store에 `overlayQueue: OverlayItem[]`를 추가하고 WS 이벤트 핸들러(`phase_change`, `kill`)에서 `pushOverlay`를 호출한다. `CinematicOverlay` 컴포넌트가 큐의 첫 번째 아이템을 fullscreen으로 표시하고, `durationMs` 후 자동으로 `shiftOverlay`를 호출한다.

**Tech Stack:** React, TypeScript, Zustand, CSS (`100dvh`, `position: fixed`)

---

## File Map

| 파일 | 변경 | 책임 |
|---|---|---|
| `frontend/src/components/GameRoom.tsx` | 수정 | `100dvh` 치환 + `day_discussion` 패널 collapse |
| `frontend/src/pages/RoomPage.tsx` | 수정 | `100dvh` 치환 + CinematicOverlay 마운트 |
| `frontend/src/components/WaitingRoom.tsx` | 수정 | `100dvh` 치환 |
| `frontend/src/pages/LobbyPage.tsx` | 수정 | `100dvh` 치환 |
| `frontend/src/pages/LandingPage.tsx` | 수정 | `100dvh` 치환 (2곳) |
| `frontend/src/components/PhaseHeader.tsx` | 수정 | 풀스크린 토글 버튼 추가 |
| `frontend/src/store/gameStore.ts` | 수정 | `OverlayItem` 타입 + overlayQueue state + phase_change/kill 핸들러 업데이트 |
| `frontend/src/components/CinematicOverlay.tsx` | 신규 | 큐 기반 fullscreen 시네마틱 오버레이 컴포넌트 |

---

## Task 1: 100dvh 뷰포트 수정

**Files:**
- Modify: `frontend/src/components/GameRoom.tsx:55`
- Modify: `frontend/src/pages/RoomPage.tsx:59,66`
- Modify: `frontend/src/components/WaitingRoom.tsx:53`
- Modify: `frontend/src/pages/LobbyPage.tsx:199`
- Modify: `frontend/src/pages/LandingPage.tsx:286,334`

- [ ] **Step 1: GameRoom.tsx — height 교체**

`frontend/src/components/GameRoom.tsx` line 55에서:
```ts
// 변경 전
height: '100vh',

// 변경 후
height: '100dvh',
```

- [ ] **Step 2: RoomPage.tsx — minHeight 교체 (2곳)**

`frontend/src/pages/RoomPage.tsx`에서:

첫 번째 (CONNECTING 로딩 div, line ~59):
```ts
// 변경 전
minHeight: '100vh', background: '#0E0C09', color: '#786F62',

// 변경 후
minHeight: '100dvh', background: '#0E0C09', color: '#786F62',
```

두 번째 (메인 래퍼 div, line ~66):
```ts
// 변경 전
{ minHeight: '100vh', background: '#0E0C09', position: 'relative' }

// 변경 후
{ minHeight: '100dvh', background: '#0E0C09', position: 'relative' }
```

- [ ] **Step 3: WaitingRoom.tsx — minHeight 교체**

`frontend/src/components/WaitingRoom.tsx` line 53에서:
```ts
// 변경 전
minHeight: '100vh', background: T.bg, color: T.text, fontFamily: SANS,

// 변경 후
minHeight: '100dvh', background: T.bg, color: T.text, fontFamily: SANS,
```

- [ ] **Step 4: LobbyPage.tsx — minHeight 교체**

`frontend/src/pages/LobbyPage.tsx` line 199에서:
```ts
// 변경 전
<div style={{ minHeight: '100vh', background: T.bg, color: T.text, fontFamily: SANS }}>

// 변경 후
<div style={{ minHeight: '100dvh', background: T.bg, color: T.text, fontFamily: SANS }}>
```

- [ ] **Step 5: LandingPage.tsx — minHeight 교체 (2곳)**

`frontend/src/pages/LandingPage.tsx` line 286에서:
```ts
// 변경 전
style={{ background: T.bg, color: T.text, fontFamily: FONT_SANS, minHeight: '100vh', position: 'relative' }}

// 변경 후
style={{ background: T.bg, color: T.text, fontFamily: FONT_SANS, minHeight: '100dvh', position: 'relative' }}
```

`frontend/src/pages/LandingPage.tsx` line 334에서:
```ts
// 변경 전
minHeight: 'calc(100vh - 73px)',

// 변경 후
minHeight: 'calc(100dvh - 73px)',
```

- [ ] **Step 6: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 7: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/components/GameRoom.tsx frontend/src/pages/RoomPage.tsx frontend/src/components/WaitingRoom.tsx frontend/src/pages/LobbyPage.tsx frontend/src/pages/LandingPage.tsx
git commit -m "fix: replace 100vh with 100dvh for dynamic viewport height"
```

---

## Task 2: day_discussion 패널 Collapse

**Files:**
- Modify: `frontend/src/components/GameRoom.tsx`

현재 `day_discussion` 페이즈에서 오른쪽 280px aside가 빈 안내 텍스트를 보여준다. 이를 제거하고 채팅 영역이 전체 폭을 차지하도록 한다.

- [ ] **Step 1: 현재 오른쪽 aside 구조 확인**

`GameRoom.tsx`의 `{/* Right: action panel (280px) */}` aside 블록 확인. 현재 구조:

```tsx
<aside style={{ width: '280px', flexShrink: 0, display: 'flex', flexDirection: 'column' }}>
  {phase === 'day_vote' && <VotePanel />}
  {phase === 'night' && <NightPanel />}
  {(phase === 'day_discussion' || !phase) && (
    <div style={{ ... }}>
      <div>토론 페이즈</div>
      <div>채팅을 통해 의심스러운 플레이어를 찾아내세요.</div>
    </div>
  )}
</aside>
```

- [ ] **Step 2: day_discussion일 때 aside 제거**

오른쪽 aside 전체를 조건부 렌더링으로 감싼다:

```tsx
{/* Right: action panel (280px) — day_discussion 때는 숨김 */}
{phase !== 'day_discussion' && (
  <aside style={{
    width: '280px', flexShrink: 0,
    display: 'flex', flexDirection: 'column',
  }}>
    {phase === 'day_vote' && <VotePanel />}
    {phase === 'night' && <NightPanel />}
  </aside>
)}
```

- [ ] **Step 3: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/components/GameRoom.tsx
git commit -m "feat: collapse right panel in day_discussion phase"
```

---

## Task 3: PhaseHeader 풀스크린 버튼

**Files:**
- Modify: `frontend/src/components/PhaseHeader.tsx`

- [ ] **Step 1: useState import 추가 확인**

`PhaseHeader.tsx`의 현재 import:
```ts
import { useEffect, useRef } from 'react'
```

`useState`를 추가:
```ts
import { useEffect, useRef, useState } from 'react'
```

- [ ] **Step 2: FullscreenButton 컴포넌트 추가**

`PhaseHeader.tsx`의 `const PHASE_LABELS` 상수 선언 바로 위에 추가:

```tsx
const MONO = "'JetBrains Mono', monospace"  // 이미 있으면 중복 추가 않음

function FullscreenButton() {
  const [isFull, setIsFull] = useState(false)

  useEffect(() => {
    const handler = () => setIsFull(!!document.fullscreenElement)
    document.addEventListener('fullscreenchange', handler)
    return () => document.removeEventListener('fullscreenchange', handler)
  }, [])

  function toggle() {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen()
    } else {
      document.exitFullscreen()
    }
  }

  return (
    <button
      onClick={toggle}
      style={{
        background: 'transparent',
        border: '1px solid #2E2820',
        borderRadius: '2px',
        color: '#786F62',
        fontFamily: MONO,
        fontSize: '14px',
        width: '28px',
        height: '28px',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        cursor: 'pointer',
        transition: 'border-color 150ms ease, color 150ms ease',
        flexShrink: 0,
      }}
      title={isFull ? '풀스크린 해제' : '풀스크린'}
    >
      {isFull ? '✕' : '⛶'}
    </button>
  )
}
```

- [ ] **Step 3: FullscreenButton을 헤더 오른쪽에 삽입**

`PhaseHeader.tsx`의 return 블록에서 `{/* Right: timer */}` 영역을 타이머와 풀스크린 버튼을 감싸는 형태로 변경:

```tsx
{/* Right: timer + fullscreen */}
<div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
  {timerRemainingSec > 0 && (
    <span style={{
      fontFamily: MONO, fontSize: '28px', letterSpacing: '-0.02em',
      color: isUrgent ? T.danger : T.accent,
      transition: 'color 300ms ease',
      fontVariantNumeric: 'tabular-nums',
    }}>
      {timerStr}
    </span>
  )}
  <FullscreenButton />
</div>
```

- [ ] **Step 4: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/components/PhaseHeader.tsx
git commit -m "feat: add fullscreen toggle button to PhaseHeader"
```

---

## Task 4: gameStore — OverlayItem 타입 + 큐 state + WS 핸들러 업데이트

**Files:**
- Modify: `frontend/src/store/gameStore.ts`

- [ ] **Step 1: OverlayItem 타입 추가**

`gameStore.ts` 파일 상단 import 블록 바로 다음, `interface GameStore` 선언 위에 타입 추가:

```ts
export interface OverlayItem {
  type: 'phase' | 'kill' | 'elim'
  title: string
  eyebrow?: string
  stamp?: string
  hint?: string
  rolePills?: { label: string; role: 'mafia' | 'police' | 'citizen' }[]
  theme: 'day' | 'vote' | 'night' | 'elim' | 'killed'
  durationMs: number
}
```

- [ ] **Step 2: GameStore interface에 overlayQueue 관련 필드 추가**

`interface GameStore` 블록에 아래 3개 필드 추가 (기존 필드 아래에):

```ts
  overlayQueue: OverlayItem[]
  pushOverlay: (item: OverlayItem) => void
  shiftOverlay: () => void
```

- [ ] **Step 3: 초기 state에 overlayQueue 추가**

`create<GameStore>()(...)` 안의 초기 상태 객체에 추가:

```ts
overlayQueue: [],
```

- [ ] **Step 4: pushOverlay, shiftOverlay 액션 추가**

초기 state 객체 다음의 액션 함수들과 함께 추가:

```ts
pushOverlay: (item) =>
  set((s) => ({ overlayQueue: [...s.overlayQueue, item] })),

shiftOverlay: () =>
  set((s) => ({ overlayQueue: s.overlayQueue.slice(1) })),
```

- [ ] **Step 5: buildPhaseOverlay 헬퍼 함수 추가**

`gameStore.ts`의 import 블록 바로 아래, `create<GameStore>()` 호출 위에 추가:

```ts
function buildPhaseOverlay(
  phase: string,
  round: number | undefined,
  myRole: string,
): OverlayItem | null {
  const r = round !== undefined ? `Round ${round}` : undefined

  if (phase === 'day_discussion') {
    return {
      type: 'phase',
      title: '낮 — 토론',
      eyebrow: r,
      hint: '의심스러운 플레이어를 찾아내세요',
      theme: 'day',
      durationMs: 1800,
    }
  }

  if (phase === 'day_vote') {
    return {
      type: 'phase',
      title: '투표',
      eyebrow: r,
      hint: '처형할 플레이어에 투표하세요',
      theme: 'vote',
      durationMs: 1800,
    }
  }

  if (phase === 'night') {
    const pills: OverlayItem['rolePills'] =
      myRole === 'mafia'
        ? [
            { label: '처치 대상 선택', role: 'mafia' },
            { label: '조사 중', role: 'police' },
            { label: '대기', role: 'citizen' },
          ]
        : myRole === 'police'
        ? [
            { label: '움직임', role: 'mafia' },
            { label: '조사 가능', role: 'police' },
            { label: '대기', role: 'citizen' },
          ]
        : [
            { label: '움직임', role: 'mafia' },
            { label: '조사 중', role: 'police' },
            { label: '대기', role: 'citizen' },
          ]

    return {
      type: 'phase',
      title: '밤',
      eyebrow: r,
      rolePills: pills,
      theme: 'night',
      durationMs: 1800,
    }
  }

  return null
}
```

- [ ] **Step 6: phase_change 핸들러에 pushOverlay 호출 추가**

현재 `case 'phase_change':` 블록:

```ts
case 'phase_change': {
  const { phase, round, duration, alive_players } = event.payload
  const updates: Partial<GameStore> = { phase }
  if (round !== undefined) updates.round = round
  if (alive_players !== undefined) updates.alivePlayerIDs = alive_players
  updates.votes = {}
  set((s) => ({ ...updates, room: s.room ? { ...s.room, status: 'playing' } : s.room }))
  if (duration !== undefined) {
    startTimer(get, set, duration)
  }
  break
}
```

다음으로 교체:

```ts
case 'phase_change': {
  const { phase, round, duration, alive_players } = event.payload
  const updates: Partial<GameStore> = { phase }
  if (round !== undefined) updates.round = round
  if (alive_players !== undefined) updates.alivePlayerIDs = alive_players
  updates.votes = {}
  set((s) => ({ ...updates, room: s.room ? { ...s.room, status: 'playing' } : s.room }))
  if (duration !== undefined) {
    startTimer(get, set, duration)
  }
  const overlayItem = buildPhaseOverlay(phase, round, get().myRole)
  if (overlayItem) get().pushOverlay(overlayItem)
  break
}
```

- [ ] **Step 7: kill 핸들러에 pushOverlay 호출 추가**

현재 `case 'kill':` 블록:

```ts
case 'kill': {
  const { player_id, role } = event.payload
  set((s) => ({
    alivePlayerIDs: s.alivePlayerIDs.filter((id) => id !== player_id),
    room: s.room
      ? {
          ...s.room,
          players: s.room.players.map((p) =>
            p.id === player_id ? { ...p, is_alive: false } : p
          ),
        }
      : null,
    messages: [
      ...s.messages,
      {
        id: `${Date.now()}-${Math.random()}`,
        player_id: 'system',
        message: role
          ? `플레이어가 사망했습니다. (역할: ${role})`
          : '플레이어가 사망했습니다.',
        mafia_only: false,
        is_system: true,
      },
    ],
  }))
  break
}
```

다음으로 교체:

```ts
case 'kill': {
  const { player_id, role, reason } = event.payload
  const playerName =
    get().room?.players.find((p) => p.id === player_id)?.name ?? player_id
  const roleLabel = role ?? '알 수 없음'

  set((s) => ({
    alivePlayerIDs: s.alivePlayerIDs.filter((id) => id !== player_id),
    room: s.room
      ? {
          ...s.room,
          players: s.room.players.map((p) =>
            p.id === player_id ? { ...p, is_alive: false } : p
          ),
        }
      : null,
    messages: [
      ...s.messages,
      {
        id: `${Date.now()}-${Math.random()}`,
        player_id: 'system',
        message: role
          ? `플레이어가 사망했습니다. (역할: ${role})`
          : '플레이어가 사망했습니다.',
        mafia_only: false,
        is_system: true,
      },
    ],
  }))

  const isVote = reason === 'vote'
  get().pushOverlay({
    type: isVote ? 'elim' : 'kill',
    title: playerName,
    eyebrow: isVote ? '투표 결과' : '밤 사이에',
    stamp: `${roleLabel} ${isVote ? '탈락' : '사망'}`,
    theme: isVote ? 'elim' : 'killed',
    durationMs: 2500,
  })
  break
}
```

- [ ] **Step 8: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 9: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/store/gameStore.ts
git commit -m "feat: add overlayQueue to gameStore with phase_change and kill overlay triggers"
```

---

## Task 5: CinematicOverlay 컴포넌트 신규 작성 + RoomPage 마운트

**Files:**
- Create: `frontend/src/components/CinematicOverlay.tsx`
- Modify: `frontend/src/pages/RoomPage.tsx`

- [ ] **Step 1: CinematicOverlay.tsx 파일 생성**

`frontend/src/components/CinematicOverlay.tsx`:

```tsx
import { useEffect } from 'react'
import { useGameStore } from '../store/gameStore'

const THEME_STYLES: Record<string, { bg: string; eyebrowColor: string; titleColor: string; stampColor: string }> = {
  day: {
    bg: 'radial-gradient(ellipse at center, #1A1508 0%, #0E0C09 60%, #0A0900 100%)',
    eyebrowColor: '#C4963A',
    titleColor: '#ECE7DE',
    stampColor: '#C4963A',
  },
  vote: {
    bg: 'radial-gradient(ellipse at center, #180808 0%, #100606 60%, #0A0404 100%)',
    eyebrowColor: '#C4963A',
    titleColor: '#ECE7DE',
    stampColor: '#8C1F1F',
  },
  night: {
    bg: 'radial-gradient(ellipse at center, #0C0810 0%, #060406 60%, #030204 100%)',
    eyebrowColor: '#9B8EBF',
    titleColor: '#D8D0F0',
    stampColor: '#9B8EBF',
  },
  elim: {
    bg: 'radial-gradient(ellipse at center, #180808 0%, #0A0606 60%, #050303 100%)',
    eyebrowColor: '#8C1F1F',
    titleColor: '#ECE7DE',
    stampColor: '#8C1F1F',
  },
  killed: {
    bg: '#000000',
    eyebrowColor: '#8C1F1F',
    titleColor: '#C8C0B8',
    stampColor: '#786F62',
  },
}

const ROLE_PILL_COLORS: Record<string, { bg: string; border: string; color: string }> = {
  mafia:   { bg: 'rgba(140,31,31,0.15)',   border: '#8C1F1F', color: '#C87070' },
  police:  { bg: 'rgba(155,142,191,0.15)', border: '#9B8EBF', color: '#C8C0F8' },
  citizen: { bg: 'rgba(120,111,98,0.15)',  border: '#786F62', color: '#B8B0A8' },
}

const SERIF = "'Instrument Serif', Georgia, serif"
const MONO  = "'JetBrains Mono', monospace"

export default function CinematicOverlay() {
  const { overlayQueue, shiftOverlay } = useGameStore()
  const current = overlayQueue[0]

  useEffect(() => {
    if (!current) return
    const t = setTimeout(shiftOverlay, current.durationMs)
    return () => clearTimeout(t)
  }, [current?.type, current?.title, current?.theme])

  if (!current) return null

  const theme = THEME_STYLES[current.theme] ?? THEME_STYLES.day

  return (
    <div
      onClick={shiftOverlay}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 8000,
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        background: theme.bg,
        cursor: 'pointer',
        userSelect: 'none',
      }}
    >
      {/* Scanline effect */}
      <div
        style={{
          position: 'absolute',
          inset: 0,
          backgroundImage: 'repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.08) 2px, rgba(0,0,0,0.08) 4px)',
          pointerEvents: 'none',
        }}
      />
      {/* Vignette */}
      <div
        style={{
          position: 'absolute',
          inset: 0,
          background: 'radial-gradient(ellipse at center, transparent 40%, rgba(0,0,0,0.6) 100%)',
          pointerEvents: 'none',
        }}
      />

      {/* Content */}
      <div
        style={{
          position: 'relative',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: '16px',
          padding: '48px 64px',
          textAlign: 'center',
        }}
      >
        {/* Eyebrow */}
        {current.eyebrow && (
          <div
            style={{
              fontFamily: MONO,
              fontSize: '11px',
              textTransform: 'uppercase',
              letterSpacing: '0.2em',
              color: theme.eyebrowColor,
            }}
          >
            {current.eyebrow}
          </div>
        )}

        {/* Title */}
        <div
          style={{
            fontFamily: SERIF,
            fontSize: 'clamp(48px, 8vw, 96px)',
            color: theme.titleColor,
            lineHeight: 1.1,
          }}
        >
          {current.title}
        </div>

        {/* Hint */}
        {current.hint && (
          <div
            style={{
              fontFamily: MONO,
              fontSize: '12px',
              color: theme.eyebrowColor,
              opacity: 0.7,
              letterSpacing: '0.05em',
            }}
          >
            {current.hint}
          </div>
        )}

        {/* Role pills */}
        {current.rolePills && current.rolePills.length > 0 && (
          <div style={{ display: 'flex', gap: '10px', flexWrap: 'wrap', justifyContent: 'center', marginTop: '8px' }}>
            {current.rolePills.map((pill, i) => {
              const pillStyle = ROLE_PILL_COLORS[pill.role] ?? ROLE_PILL_COLORS.citizen
              return (
                <div
                  key={i}
                  style={{
                    fontFamily: MONO,
                    fontSize: '10px',
                    textTransform: 'uppercase',
                    letterSpacing: '0.1em',
                    padding: '4px 12px',
                    borderRadius: '2px',
                    background: pillStyle.bg,
                    border: `1px solid ${pillStyle.border}`,
                    color: pillStyle.color,
                  }}
                >
                  {pill.label}
                </div>
              )
            })}
          </div>
        )}

        {/* Stamp */}
        {current.stamp && (
          <div
            style={{
              fontFamily: MONO,
              fontSize: '13px',
              textTransform: 'uppercase',
              letterSpacing: '0.15em',
              color: theme.stampColor,
              marginTop: '8px',
              opacity: 0.85,
            }}
          >
            {current.stamp}
          </div>
        )}

        {/* Skip hint */}
        <div
          style={{
            position: 'absolute',
            bottom: '-32px',
            fontFamily: MONO,
            fontSize: '9px',
            textTransform: 'uppercase',
            letterSpacing: '0.12em',
            color: '#2E2820',
          }}
        >
          클릭하여 건너뛰기
        </div>
      </div>
    </div>
  )
}
```

- [ ] **Step 2: RoomPage.tsx에 CinematicOverlay import + 마운트**

`frontend/src/pages/RoomPage.tsx` import 블록에 추가:

```ts
import CinematicOverlay from '../components/CinematicOverlay'
```

return 블록의 `<LeaveConfirmModal ... />` 바로 앞에 추가:

```tsx
<CinematicOverlay />
<LeaveConfirmModal
  isOpen={blocker.state === 'blocked'}
  onConfirm={handleLeaveConfirm}
  onCancel={handleLeaveCancel}
/>
```

- [ ] **Step 3: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/components/CinematicOverlay.tsx frontend/src/pages/RoomPage.tsx
git commit -m "feat: add CinematicOverlay component with phase and kill themes"
```

---

## 최종 검증

- [ ] `npm run build` 에러 없음
- [ ] 로비/랜딩 페이지에서 모바일 Chrome 주소창이 콘텐츠를 가리지 않음 (`100dvh`)
- [ ] `day_discussion` 페이즈에서 오른쪽 패널이 사라지고 채팅이 전체 폭을 차지
- [ ] PhaseHeader 우측에 `⛶` 버튼 노출, 클릭 시 풀스크린 진입, `✕` 클릭 시 복귀
- [ ] 페이즈 전환 시 CinematicOverlay가 1.8초간 표시 후 자동 닫힘
- [ ] 투표 결과 `kill` 이벤트 시 `elim` 테마 오버레이 2.5초 표시
- [ ] 마피아 킬 `kill` 이벤트 시 `killed` 테마 오버레이 2.5초 표시
- [ ] 오버레이 클릭 시 즉시 닫힘 (건너뛰기)
- [ ] 연속 이벤트가 큐에 쌓혀 순서대로 표시됨
