## Why

현재 프론트엔드 UI는 기본적인 Tailwind gray 계열로만 구성되어 있어 게임의 분위기와 맞지 않는다. 랜딩 페이지도 없어 접속하자마자 방 목록이 바로 표시된다. Glacier 디자인 시스템(glassmorphism, 네이비-아이스블루 팔레트)을 적용해 게임 전체의 시각적 완성도를 높이고, 분리된 랜딩 페이지로 첫인상을 개선한다.

## What Changes

- **새 랜딩 페이지** (`/`): AI Mafia 히어로 섹션 + 역할 소개 카드 3개 + "게임 시작하기" CTA
- **라우팅 변경**: `/` → LandingPage, `/lobby` → LobbyPage (기존 `/` 로비를 `/lobby`로 이동)
- **Glacier 디자인 시스템 도입**: `index.css`에 Tailwind v4 `@theme` 토큰 + `.glass` / `.glass-elevated` 컴포넌트 유틸 + Inter 폰트
- **전체 페이지/컴포넌트 재스타일**: LobbyPage, RoomPage, WaitingRoom, GameRoom, PhaseHeader, PlayerList, ChatLog, ChatInput, VotePanel, NightPanel, ResultOverlay 모두 Glacier 팔레트로 교체

## Capabilities

### New Capabilities

- `landing-page`: AI Mafia 게임 소개 랜딩 페이지. 히어로 섹션, 역할 설명 카드, 로비 진입 CTA 포함.

### Modified Capabilities

- `lobby-ui`: 라우트 `/lobby`로 변경, Glacier glass 스타일 전면 적용
- `game-ui`: 게임 화면(GameRoom, 하위 컴포넌트 전체) Glacier 재스타일
- `waiting-room-ui`: 대기실 화면 Glacier 재스타일

## Impact

- `frontend/src/index.css`: Tailwind v4 `@theme` 토큰, `.glass` 유틸리티, Inter 폰트 임포트
- `frontend/src/App.tsx`: 라우팅 변경 (새 LandingPage 추가, LobbyPage 경로 변경)
- `frontend/src/pages/LandingPage.tsx`: 신규 파일
- `frontend/src/pages/LobbyPage.tsx`: 전면 재스타일 + 라우트 `/lobby`
- `frontend/src/pages/RoomPage.tsx`: 배경 컬러 교체
- `frontend/src/components/*`: 전체 컴포넌트 Glacier 재스타일 (11개 파일)
- 외부 의존성 없음 (Inter는 Google Fonts CDN, `@import url(...)`)
