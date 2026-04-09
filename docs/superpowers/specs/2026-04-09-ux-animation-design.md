# UX/Animation Improvements Design

**Date:** 2026-04-09
**Status:** Approved
**Sub-project:** A (UX/Animation) — B(실시간 플레이어 현황), C(광고) 는 별도 스펙

---

## Context

현재 게임 UI에 4가지 문제가 있다:

1. **뷰포트 침범** — `100vh`를 사용해서 Chrome 주소창이 게임 화면 위를 가림. 하단 버튼이 잘리는 현상 발생.
2. **오른쪽 패널 낭비** — 토론 페이즈(day_discussion)에서 투표/밤 패널이 없는데 280px 공간이 빈 텍스트만 표시하며 낭비됨.
3. **페이즈 전환 컨텍스트 부족** — 낮↔밤 전환 시 배경색만 바뀜. "지금 내가 무엇을 할 수 있나"를 명확히 알 수 없음.
4. **투표 결과/사망 발표가 밋밋함** — 시스템 메시지 한 줄로만 처리되어 긴장감 없음.

---

## Goals / Non-Goals

**Goals:**
- 브라우저 주소창에 의한 뷰포트 침범 수정
- 풀스크린 모드 지원
- 토론 페이즈에서 오른쪽 패널 collapse → 채팅 전체폭 활용
- 페이즈 전환마다 풀스크린 시네마틱 오버레이 (역할별 액션 힌트 포함)
- 투표 처형 / 마피아 킬 사망 발표에 시네마틱 오버레이 적용

**Non-Goals:**
- 모바일 반응형 레이아웃 (별도 작업)
- 사운드 효과
- `game_over` 결과 화면 변경 (기존 `ResultOverlay` 유지)

---

## Decisions

### 1. 뷰포트 수정 (`100vh` → `100dvh`)

모든 컴포넌트에서 `height: '100vh'`를 `height: '100dvh'`로 교체한다.
`100dvh`(dynamic viewport height)는 브라우저 주소창이 보이든 숨겨지든 항상 실제 가시 영역에 맞게 계산된다.

변경 대상: `GameRoom.tsx`, `WaitingRoom.tsx`, `RoomPage.tsx`, `LobbyPage.tsx`, `LandingPage.tsx`

---

### 2. 풀스크린 토글 버튼

`PhaseHeader.tsx` 오른쪽 끝에 풀스크린 토글 버튼을 추가한다.

```tsx
// PhaseHeader.tsx 추가
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
    <button onClick={toggle} style={{ /* 현재 MONO 스타일 유지 */ }}>
      {isFull ? '✕' : '⛶'}
    </button>
  )
}
```

---

### 3. 토론 페이즈 오른쪽 패널 Collapse

`GameRoom.tsx`에서 `phase === 'day_discussion'` 일 때 오른쪽 `aside`를 렌더링하지 않는다.
채팅 `main` 영역이 자동으로 전체 폭을 차지한다.

```tsx
// GameRoom.tsx
{phase === 'day_vote' && <VotePanel />}
{phase === 'night' && <NightPanel />}
// day_discussion 일 때는 aside 자체를 제거
```

---

### 4. CinematicOverlay 시스템

#### 4-1. Zustand store 확장

```ts
// gameStore.ts 추가
interface OverlayItem {
  type: 'phase' | 'kill' | 'elim'
  title: string           // 크게 표시될 텍스트 (페이즈명 or 플레이어 이름)
  eyebrow?: string        // 상단 작은 텍스트 (예: "Round 2", "투표 결과")
  stamp?: string          // 하단 스탬프 (예: "시민 탈락", "투표 가능")
  hint?: string           // 부제 (예: "의심스러운 플레이어를 찾아내세요")
  rolePills?: { label: string; role: 'mafia' | 'police' | 'citizen' }[]
  theme: 'day' | 'vote' | 'night' | 'elim' | 'killed'
  durationMs: number      // 자동 닫힘 시간 (페이즈: 1800ms, 탈락/사망: 2500ms)
}

interface GameStore {
  // ... 기존 필드
  overlayQueue: OverlayItem[]
  pushOverlay: (item: OverlayItem) => void
  shiftOverlay: () => void
}
```

#### 4-2. WS 이벤트 핸들러에서 pushOverlay 호출

```ts
// gameStore.ts — phase_change 핸들러 내
case 'phase_change': {
  // 기존 상태 업데이트 로직 유지 ...

  // 오버레이 추가
  const overlayItem = buildPhaseOverlay(phase, round, myRole)
  if (overlayItem) get().pushOverlay(overlayItem)
  break
}

case 'kill': {
  // 기존 상태 업데이트 로직 유지 ...

  const reason = event.payload.reason  // 'vote' | 'mafia'
  get().pushOverlay({
    type: reason === 'vote' ? 'elim' : 'kill',
    title: playerName,
    eyebrow: reason === 'vote' ? '투표 결과' : '밤 사이에',
    stamp: `${roleLabel} ${reason === 'vote' ? '탈락' : '사망'}`,
    theme: reason === 'vote' ? 'elim' : 'killed',
    durationMs: 2500,
  })
  break
}
```

`buildPhaseOverlay` 함수는 페이즈와 역할에 따라 다른 OverlayItem을 반환한다:

| phase | myRole | title | rolePills | theme | durationMs |
|---|---|---|---|---|---|
| day_discussion | any | 낮 — 토론 | 없음 | day | 1800 |
| day_vote | any | 투표 | 없음 | vote | 1800 |
| night | mafia | 밤 | [{마피아: 처치 선택}, {경찰: 조사 중}, {시민: 대기}] | night | 1800 |
| night | police | 밤 | [{마피아: 움직임}, {경찰: 조사 가능}, {시민: 대기}] | night | 1800 |
| night | citizen | 밤 | [{마피아: 움직임}, {경찰: 조사 중}, {시민: 대기}] | night | 1800 |

#### 4-3. CinematicOverlay 컴포넌트

```tsx
// frontend/src/components/CinematicOverlay.tsx (신규)

export default function CinematicOverlay() {
  const { overlayQueue, shiftOverlay } = useGameStore()
  const current = overlayQueue[0]

  useEffect(() => {
    if (!current) return
    const t = setTimeout(shiftOverlay, current.durationMs)
    return () => clearTimeout(t)
  }, [current])

  if (!current) return null

  return (
    <div
      onClick={shiftOverlay}   // 클릭 시 즉시 스킵
      style={{
        position: 'fixed', inset: 0, zIndex: 8000,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        // theme별 배경색 적용
      }}
    >
      {/* scanline, vignette, eyebrow, title, stamp, rolePills */}
    </div>
  )
}
```

`CinematicOverlay`는 `RoomPage.tsx`에서 `<GameRoom />` 위에 항상 마운트된다.

#### 4-4. 테마별 색상

| theme | 배경 | eyebrow 색 | title 색 | stamp 색 |
|---|---|---|---|---|
| day | `#0E0C09` (어두운 골드 그라디언트) | `#C4963A` | `#ECE7DE` | `#C4963A` |
| vote | `#100808` (레드 틴트) | `#C4963A` | `#ECE7DE` | `#8C1F1F` |
| night | `#060406` (퍼플 방사형) | `#9B8EBF` | `#D8D0F0` | — |
| elim | `#0A0606` (레드 방사형) | `#8C1F1F` | `#ECE7DE` | `#8C1F1F` |
| killed | `#000000` (완전 블랙) | `#8C1F1F` | `#C8C0B8` | `#786F62` |

모든 테마에 공통으로 scanline(미세 수평선 오버레이)과 vignette(가장자리 어둠) CSS를 적용한다.

---

## Files Changed

| 파일 | 변경 유형 |
|---|---|
| `frontend/src/components/CinematicOverlay.tsx` | 신규 |
| `frontend/src/store/gameStore.ts` | 수정 (overlayQueue, pushOverlay, shiftOverlay, WS 핸들러) |
| `frontend/src/components/GameRoom.tsx` | 수정 (패널 collapse, 100dvh) |
| `frontend/src/components/PhaseHeader.tsx` | 수정 (풀스크린 버튼) |
| `frontend/src/pages/RoomPage.tsx` | 수정 (100dvh, CinematicOverlay 마운트) |
| `frontend/src/pages/WaitingRoom.tsx` | 수정 (100dvh) |
| `frontend/src/pages/LobbyPage.tsx` | 수정 (100dvh) |
| `frontend/src/pages/LandingPage.tsx` | 수정 (100dvh) |

---

## Risks / Trade-offs

- **`100dvh` 브라우저 지원**: Chrome 108+, Safari 15.4+, Firefox 101+ 이상 지원. 구형 브라우저는 `100vh` fallback으로 처리. 현재 타깃 사용자 기준 무방.
- **오버레이 큐잉**: 빠른 연속 이벤트(예: 투표 결과 직후 페이즈 전환)는 큐에 쌓여 순서대로 표시됨. 최대 큐 길이 제한 없음 — 이벤트가 5개 이상 쌓이는 경우는 발생하지 않음.
- **`game_over` 제외**: ResultOverlay가 이미 잘 동작하므로 변경하지 않음.
