## Context

프론트엔드는 Tailwind v4 (`@import "tailwindcss"`)를 사용 중이며, 현재 모든 페이지/컴포넌트가 `bg-gray-900`, `bg-gray-800`, `bg-gray-700` 계열로 스타일링되어 있다. 외부 디자인 레퍼런스(`ui/DESIGN.md`, `ui/stitch/screen.png`)로 "Glacier" glassmorphism 시스템이 정의되어 있다: 네이비 배경 `#0a0e1a`, 아이스블루 `#7dd3fc`, 라벤더 `#c8a0f0`, frosted glass 카드.

## Goals / Non-Goals

**Goals:**
- Tailwind v4 `@theme` 방식으로 Glacier 컬러 토큰 통합
- `@layer components`로 `.glass` / `.glass-elevated` 재사용 유틸리티 정의
- 모든 페이지와 게임 컴포넌트에 일관된 Glacier 스타일 적용
- 새 LandingPage 생성 + 라우팅 분리 (`/` → 랜딩, `/lobby` → 로비)

**Non-Goals:**
- 게임 로직, 상태 관리, API 변경
- 애니메이션/트랜지션 심층 구현 (기본 Tailwind transition만 사용)
- 반응형 레이아웃 전면 재설계 (현재 레이아웃 구조 유지, 스타일만 교체)
- 다크/라이트 테마 토글

## Decisions

### Tailwind v4 `@theme` vs CSS 변수 직접 사용

**결정**: `@theme {}` 블록으로 토큰을 정의한다.

```css
@theme {
  --color-background: #0a0e1a;
  --color-primary: #7dd3fc;
  --color-tertiary: #c8a0f0;
  --color-surface: #0f1524;
  --color-surface-container: #141c2e;
  --color-surface-high: #1a2438;
  --color-on-surface: #e0e8f0;
  --color-on-surface-variant: #a0b4c4;
  --color-outline-variant: #2a3a48;
  --font-sans: 'Inter', sans-serif;
}
```

`@theme`에 등록된 색상은 Tailwind가 `bg-*`, `text-*`, `border-*` 유틸리티로 자동 생성한다 (`bg-background`, `text-primary`, `text-on-surface-variant` 등). v3 방식의 `tailwind.config.js` 불필요.

**대안**: CSS 변수를 직접 `var(--color-primary)` 인라인으로 사용.
**왜 기각**: Tailwind 유틸리티 클래스 방식과 일관성이 깨지고, 코드 가독성이 낮아진다.

### `.glass` 유틸리티 — `@layer components` vs 인라인 style

**결정**: `@layer components`에 `.glass` / `.glass-elevated`를 정의한다.

```css
@layer components {
  .glass {
    background: rgba(15, 21, 36, 0.6);
    backdrop-filter: blur(16px);
    border: 1px solid rgba(125, 211, 252, 0.1);
  }
  .glass-elevated {
    background: rgba(15, 21, 36, 0.75);
    backdrop-filter: blur(24px);
    border: 1px solid rgba(125, 211, 252, 0.15);
  }
}
```

`backdrop-filter: blur()`는 CSS 변수나 Tailwind arbitrary value로 표현하기 어색하고, `rgba()`에 Tailwind 토큰을 직접 넣을 수 없으므로 컴포넌트 유틸리티로 분리한다.

**대안**: Tailwind `backdrop-blur-*` + `bg-white/5` 조합으로 JSX에 인라인.
**왜 기각**: glassmorphism 효과는 3개 속성(bg opacity, blur, border)이 항상 묶이므로 `.glass` 단일 클래스가 가독성에 훨씬 유리하다.

### Inter 폰트 로딩

**결정**: `index.css` 상단에 `@import url('https://fonts.googleapis.com/...')` 추가.

`index.html`의 `<link>` 태그 대신 CSS `@import`를 사용하면 스타일시트 한 파일에 폰트 설정이 집중된다. `@theme`의 `--font-family-sans` 재정의로 Tailwind의 `font-sans` 유틸리티가 Inter를 사용하게 된다.

### 라우팅 변경 — `/lobby` 경로

**결정**: `App.tsx`에서 `<Route path="/lobby" element={<LobbyPage />} />`로 변경하고, `LandingPage`를 `<Route path="/" element={<LandingPage />} />`로 추가한다.

`ResultOverlay`의 `navigate('/')` (나가기 버튼)는 랜딩 페이지로 이동하는 것이 자연스럽다. 변경 불필요.

### LandingPage 히어로 배경

**결정**: 실제 이미지 없이 CSS radial gradient로 대기 분위기를 표현한다.

```css
background: radial-gradient(ellipse at 70% 20%, rgba(14, 77, 110, 0.4) 0%, transparent 60%),
            radial-gradient(ellipse at 20% 80%, rgba(61, 32, 96, 0.3) 0%, transparent 50%),
            #0a0e1a;
```

아이스블루(primary)와 라벤더(tertiary) 글로우를 배경에 투영해 Glacier 분위기를 낸다.

**대안**: 외부 이미지 에셋 사용.
**왜 기각**: 에셋 없이도 충분한 분위기가 나오며, 의존성을 줄인다.

### 게임 화면 배경

**결정**: GameRoom과 RoomPage의 전체 배경은 `bg-background`로 교체한다. 개별 패널(PlayerList, ChatLog, VotePanel 등)은 `.glass` 클래스를 적용한다.

## Risks / Trade-offs

- **`backdrop-filter` 브라우저 지원**: 구형 브라우저에서 blur가 적용 안 될 수 있음 → 이미 `rgba` 배경이 있어 graceful degradation 자연스럽게 처리됨.
- **Google Fonts CDN 의존**: 오프라인이나 CDN 차단 환경에서 폰트 미적용 → fallback `sans-serif`로 처리됨.
- **`on-surface-variant` 클래스명 충돌 가능성**: Tailwind v4에서 하이픈 포함 컬러 토큰이 예상대로 생성되는지 확인 필요. `text-on-surface-variant`가 작동하지 않으면 `text-[#a0b4c4]` arbitrary value로 대체.
