# Ad Revenue Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Google AdSense 배너 광고를 대기실(WaitingRoom)과 게임 종료 화면(ResultOverlay)에 삽입하고, 개발 환경에서는 광고가 렌더링되지 않도록 처리한다.

**Architecture:** `AdBanner` 컴포넌트가 `VITE_ADSENSE_CLIENT` 환경변수를 읽어 없으면 `null` 반환. `index.html`에 AdSense 스크립트 추가. `.env.production`에 실제 Publisher ID/Slot ID, `.env.development`에 빈 값.

**Tech Stack:** React, TypeScript, Vite 환경변수

---

## File Map

| 파일 | 변경 유형 | 책임 |
|---|---|---|
| `frontend/src/components/AdBanner.tsx` | 신규 | AdSense 배너 컴포넌트 |
| `frontend/index.html` | 수정 | AdSense 비동기 스크립트 추가 |
| `frontend/.env.production` | 수정 | Publisher ID / Slot ID 환경변수 |
| `frontend/.env.development` | 수정 (없으면 생성) | 빈 값으로 dev 환경 비활성화 |
| `frontend/src/components/WaitingRoom.tsx` | 수정 | 플레이어 목록 아래 AdBanner 삽입 |
| `frontend/src/components/ResultOverlay.tsx` | 수정 | 결과 목록 아래 AdBanner 삽입 |

---

## Task 1: AdBanner 컴포넌트 + 환경변수 설정

**Files:**
- Create: `frontend/src/components/AdBanner.tsx`
- Modify: `frontend/index.html`
- Modify/Create: `frontend/.env.production`
- Modify/Create: `frontend/.env.development`

- [ ] **Step 1: AdBanner.tsx 파일 생성**

`frontend/src/components/AdBanner.tsx`:

```tsx
import { useEffect } from 'react'

interface Props {
  slotId: string
  style?: React.CSSProperties
}

export default function AdBanner({ slotId, style }: Props) {
  const client = import.meta.env.VITE_ADSENSE_CLIENT

  useEffect(() => {
    try {
      ;((window as any).adsbygoogle = (window as any).adsbygoogle ?? []).push({})
    } catch {
      // dev 환경에서 AdSense 스크립트 없을 때 무시
    }
  }, [])

  // VITE_ADSENSE_CLIENT 또는 slotId 없으면 렌더링 건너뜀 (dev 환경)
  if (!client || !slotId) return null

  return (
    <ins
      className="adsbygoogle"
      style={{ display: 'block', ...style }}
      data-ad-client={client}
      data-ad-slot={slotId}
      data-ad-format="auto"
      data-full-width-responsive="true"
    />
  )
}
```

- [ ] **Step 2: index.html에 AdSense 스크립트 추가**

`frontend/index.html`의 `<head>` 닫는 태그 바로 앞에 추가 (Publisher ID는 AdSense 대시보드 발급 후 실제 값으로 교체):

```html
<!-- Google AdSense — Publisher ID를 발급받은 후 ca-pub-XXXXXXXXXXXXXXXX 부분을 교체 -->
<script
  async
  src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-XXXXXXXXXXXXXXXX"
  crossorigin="anonymous"
></script>
```

- [ ] **Step 3: .env.production 생성/수정**

`frontend/.env.production` 파일이 없으면 생성, 있으면 아래 내용 추가 (실제 발급받은 값으로 교체):

```bash
# Google AdSense — AdSense 대시보드에서 발급받은 값으로 교체
VITE_ADSENSE_CLIENT=ca-pub-XXXXXXXXXXXXXXXX
VITE_ADSENSE_SLOT_WAITING=1234567890
VITE_ADSENSE_SLOT_RESULT=0987654321
```

- [ ] **Step 4: .env.development 생성/수정**

`frontend/.env.development` 파일이 없으면 생성, 있으면 아래 내용 추가 (빈 값 → dev에서 AdBanner null 반환):

```bash
# 개발 환경에서는 AdBanner를 렌더링하지 않음 (비워둠)
VITE_ADSENSE_CLIENT=
VITE_ADSENSE_SLOT_WAITING=
VITE_ADSENSE_SLOT_RESULT=
```

- [ ] **Step 5: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 6: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/components/AdBanner.tsx frontend/index.html frontend/.env.production frontend/.env.development
git commit -m "feat: add AdBanner component and AdSense environment variables"
```

---

## Task 2: WaitingRoom + ResultOverlay에 AdBanner 삽입

**Files:**
- Modify: `frontend/src/components/WaitingRoom.tsx`
- Modify: `frontend/src/components/ResultOverlay.tsx`

- [ ] **Step 1: WaitingRoom.tsx — AdBanner import 추가**

`frontend/src/components/WaitingRoom.tsx` 상단 import 블록에 추가:

```ts
import AdBanner from './AdBanner'
```

- [ ] **Step 2: WaitingRoom.tsx — 플레이어 목록 아래 AdBanner 삽입**

현재 플레이어 목록 `</div>` 바로 다음, `{/* Start button / waiting text */}` 주석 바로 위에 삽입:

현재 코드 패턴:
```tsx
          {room.players.map((p, i) => (
            // ... 플레이어 항목
          ))}
        </div>

        {/* Start button / waiting text */}
        {isHost ? (
```

다음으로 변경:
```tsx
          {room.players.map((p, i) => (
            // ... 플레이어 항목
          ))}
        </div>

        <AdBanner
          slotId={import.meta.env.VITE_ADSENSE_SLOT_WAITING}
          style={{ marginBottom: '24px' }}
        />

        {/* Start button / waiting text */}
        {isHost ? (
```

- [ ] **Step 3: ResultOverlay.tsx — AdBanner import 추가**

`frontend/src/components/ResultOverlay.tsx` 상단 import 블록에 추가:

```ts
import AdBanner from './AdBanner'
```

- [ ] **Step 4: ResultOverlay.tsx — 결과 목록 아래 AdBanner 삽입**

플레이어 결과 목록 스크롤 div 닫는 태그 다음, 액션 버튼 div 바로 위에 삽입.

현재 코드 패턴:
```tsx
        {/* Player reveal ledger */}
        <div style={{ maxHeight: '320px', overflowY: 'auto' }}>
          {/* Column headers */}
          ...
          {result.players.map((p, i) => (
            ...
          ))}
        </div>

        {/* Action buttons */}
        <div style={{
          display: 'flex', gap: '8px', padding: '20px 24px',
          borderTop: `1px solid ${T.surfaceBorder}`,
        }}>
```

다음으로 변경:
```tsx
        {/* Player reveal ledger */}
        <div style={{ maxHeight: '320px', overflowY: 'auto' }}>
          {/* Column headers */}
          ...
          {result.players.map((p, i) => (
            ...
          ))}
        </div>

        <AdBanner
          slotId={import.meta.env.VITE_ADSENSE_SLOT_RESULT}
          style={{ marginTop: '24px', marginBottom: '16px', marginLeft: '24px', marginRight: '24px' }}
        />

        {/* Action buttons */}
        <div style={{
          display: 'flex', gap: '8px', padding: '20px 24px',
          borderTop: `1px solid ${T.surfaceBorder}`,
        }}>
```

- [ ] **Step 5: 빌드 확인**

```bash
cd /Users/yuhojin/Desktop/ai_side/frontend && npm run build
```

Expected: 에러 없음

- [ ] **Step 6: 커밋**

```bash
cd /Users/yuhojin/Desktop/ai_side
git add frontend/src/components/WaitingRoom.tsx frontend/src/components/ResultOverlay.tsx
git commit -m "feat: insert AdBanner in WaitingRoom and ResultOverlay"
```

---

## 최종 검증

- [ ] `npm run build` 에러 없음
- [ ] 개발 서버(`npm run dev`)에서 대기실과 결과 화면에 광고 영역이 보이지 않음 (`.env.development` 빈 값)
- [ ] `.env.development`에 임시 테스트 값 넣으면 `<ins class="adsbygoogle">` 요소가 DOM에 존재함
- [ ] `VITE_ADSENSE_SLOT_WAITING` 또는 `VITE_ADSENSE_SLOT_RESULT` 중 하나만 비워도 해당 배너만 null 반환
- [ ] AdBanner가 null 반환 시 레이아웃 깨짐 없음 (빈 공간 없이 주변 요소가 붙음)
