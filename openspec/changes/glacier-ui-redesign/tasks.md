## 1. 디자인 시스템 기반 설정

- [x] 1.1 `frontend/src/index.css`에 Google Fonts Inter `@import url(...)` 추가
- [x] 1.2 `frontend/src/index.css`에 `@theme {}` 블록으로 Glacier 컬러 토큰 추가 (`--color-background`, `--color-primary`, `--color-tertiary`, `--color-surface`, `--color-surface-container`, `--color-surface-high`, `--color-on-surface`, `--color-on-surface-variant`, `--color-outline-variant`, `--font-sans`)
- [x] 1.3 `frontend/src/index.css`에 `@layer components`로 `.glass` / `.glass-elevated` 유틸리티 클래스 추가
- [x] 1.4 `frontend/src/index.css`의 `body` 스타일에 `background-color: #0a0e1a`, `color: #e0e8f0`, `font-family: 'Inter', sans-serif` 적용

## 2. 라우팅 변경

- [x] 2.1 `frontend/src/App.tsx`에 `LandingPage` import 추가, `<Route path="/" element={<LandingPage />} />` 추가
- [x] 2.2 `frontend/src/App.tsx`의 `LobbyPage` route를 `/lobby`로 변경

## 3. LandingPage 신규 생성

- [x] 3.1 `frontend/src/pages/LandingPage.tsx` 파일 생성
- [x] 3.2 `bg-background` + radial gradient 히어로 배경 구현 (아이스블루 + 라벤더 글로우)
- [x] 3.3 상단 glass navbar: "AI Mafia" 로고 + `/lobby` 링크 버튼
- [x] 3.4 히어로 섹션: 게임 제목, 태그라인("속여라. 간파하라. 살아남아라."), "게임 시작하기 ▶" CTA 버튼 → `/lobby`
- [x] 3.5 역할 소개 glass 카드 3개: 마피아(붉은 아이콘), 시민(파란 아이콘), 경찰(보라 아이콘) — 각 역할 이름 + 한 줄 설명

## 4. LobbyPage 재스타일

- [x] 4.1 `frontend/src/pages/LobbyPage.tsx` 상단 navbar: glass 스타일, "AI Mafia" 로고 + 홈(`/`) 링크
- [x] 4.2 페이지 배경을 `bg-background`로 변경
- [x] 4.3 공개 방 목록 영역: 섹션 헤더 + 새로고침 버튼 glass 스타일 적용
- [x] 4.4 방 목록 아이템: `glass` 카드 스타일, hover 시 `border-primary/40` 강조
- [x] 4.5 방 상태 뱃지: 대기중 → `bg-primary/20 text-primary border border-primary/30`, 게임중 → `bg-tertiary/20 text-tertiary border border-tertiary/30`
- [x] 4.6 방 만들기 패널: `glass` 카드, 인풋 `bg-surface-container border border-outline-variant focus:border-primary/40`, 버튼 `bg-primary/20 text-primary border border-primary/30 hover:bg-primary/30`
- [x] 4.7 코드 참가 패널: 동일 glass 스타일 적용
- [x] 4.8 참가 모달: `glass-elevated` 스타일 + `backdrop-filter` 오버레이

## 5. RoomPage 재스타일

- [x] 5.1 `frontend/src/pages/RoomPage.tsx`의 배경 `bg-gray-900` → `bg-background` 교체 (연결 중 화면 포함)
- [x] 5.2 `frontend/src/pages/RoomPage.tsx`의 playerID 미존재 시 `navigate('/')` → `navigate('/lobby')`로 변경

## 6. WaitingRoom 재스타일

- [x] 6.1 `frontend/src/components/WaitingRoom.tsx` 배경 제거, 페이지 전체가 `bg-background`이므로 컨테이너 최대 폭 유지
- [x] 6.2 플레이어 목록 패널: `bg-gray-800` → `glass` 카드
- [x] 6.3 초대 코드: `text-yellow-400` → `text-primary` + `font-mono tracking-widest`
- [x] 6.4 방장 뱃지: `bg-yellow-600` → `bg-primary/20 text-primary border border-primary/30`
- [x] 6.5 AI 뱃지: `bg-blue-700` → `bg-tertiary/20 text-tertiary border border-tertiary/30`
- [x] 6.6 게임 시작 버튼: `bg-green-600` → `bg-primary/20 text-primary border border-primary/30 hover:bg-primary/30`

## 7. PhaseHeader 재스타일

- [x] 7.1 `frontend/src/components/PhaseHeader.tsx`의 `bg-gray-800` → `glass` + `border-b border-primary/10`
- [x] 7.2 타이머 10초 이하 색상: `text-red-400` 유지 (위험 신호 명확성)

## 8. PlayerList 재스타일

- [x] 8.1 `frontend/src/components/PlayerList.tsx`의 `bg-gray-800` → `glass`
- [x] 8.2 방장 뱃지: `text-yellow-400` → `text-primary`
- [x] 8.3 AI 뱃지: `text-blue-400` → `text-tertiary`

## 9. ChatLog 재스타일

- [x] 9.1 `frontend/src/components/ChatLog.tsx`의 `bg-gray-900` → `glass` (패널 자체가 glass 카드)
- [x] 9.2 발신자 이름 색상: 일반 `text-blue-300` → `text-primary`, 마피아 전용 `text-red-400` 유지

## 10. ChatInput 재스타일

- [x] 10.1 `frontend/src/components/ChatInput.tsx`의 인풋: `bg-gray-700` → `bg-surface-container border border-outline-variant focus:border-primary/40 focus:outline-none`
- [x] 10.2 전송 버튼: `bg-blue-600` → `bg-primary/20 text-primary border border-primary/30 hover:bg-primary/30`

## 11. VotePanel 재스타일

- [x] 11.1 `frontend/src/components/VotePanel.tsx`의 컨테이너: `bg-gray-800` → `glass`
- [x] 11.2 투표 버튼 기본: `bg-gray-700 hover:bg-gray-600` → `bg-surface-container hover:bg-surface-high border border-outline-variant`
- [x] 11.3 투표 선택된 버튼: `bg-red-700` → `bg-red-900/60 border border-red-500/40 text-red-200` (Glacier tone 유지하되 선택 상태 명확화)

## 12. NightPanel 재스타일

- [x] 12.1 `frontend/src/components/NightPanel.tsx`의 컨테이너: `bg-gray-800` → `glass`
- [x] 12.2 마피아 킬 버튼: `bg-red-900 hover:bg-red-800` → `bg-red-900/60 hover:bg-red-800/60 border border-red-500/30`
- [x] 12.3 경찰 조사 버튼: `bg-blue-900 hover:bg-blue-800` → `bg-blue-900/60 hover:bg-blue-800/60 border border-blue-500/30`
- [x] 12.4 일반 플레이어 대기 패널: `bg-gray-800` → `glass`, 이모지 🌙 제거하고 텍스트만 유지

## 13. GameRoom 재스타일

- [x] 13.1 `frontend/src/components/GameRoom.tsx`의 마피아 야간 채팅 인풋: `bg-red-900/40 border border-red-700` → `bg-red-950/60 border border-red-500/30 focus:border-red-400/50`
- [x] 13.2 마피아 전송 버튼: `bg-red-700 hover:bg-red-600` → `bg-red-900/60 hover:bg-red-800/60 border border-red-500/30`
- [x] 13.3 낮 토론 안내 패널: `bg-gray-800` → `glass`

## 14. ResultOverlay 재스타일

- [x] 14.1 `frontend/src/components/ResultOverlay.tsx`의 오버레이 배경: `bg-black/80` → `bg-background/80 backdrop-blur-sm`
- [x] 14.2 결과 카드: `bg-gray-800` → `glass-elevated`
- [x] 14.3 승리 배너: 마피아 승리 `bg-red-900` → `bg-red-900/60 border border-red-500/30`, 시민 승리 `bg-green-900` → `bg-green-900/60 border border-green-500/30`
- [x] 14.4 플레이어 결과 행: `bg-gray-700` → `bg-surface-container`, `bg-gray-700/40` → `bg-surface-container/40`
- [x] 14.5 역할 뱃지: 마피아 `bg-red-800` → `bg-red-900/60 border border-red-500/30`, 경찰 `bg-blue-800` → `bg-blue-900/60 border border-blue-500/30`, 시민 `bg-gray-600` → `bg-surface-high`
- [x] 14.6 나가기 버튼: `bg-gray-700` → `glass`, 다시 시작 버튼: `bg-green-600` → `bg-primary/20 text-primary border border-primary/30 hover:bg-primary/30`

## 15. 빌드 및 타입 검증

- [x] 15.1 `cd frontend && npx tsc --noEmit`으로 TypeScript 오류 없음 확인
