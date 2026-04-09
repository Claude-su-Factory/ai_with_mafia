# Ad Revenue Model Design

**Date:** 2026-04-09
**Status:** Approved
**Sub-project:** C (광고 수익 모델) — A(UX/Animation), B(실시간 플레이어 현황)는 별도 스펙

---

## Context

게임 플레이 중 수익 모델이 없다. Google AdSense 배너 광고를 게임 흐름을 방해하지 않는 위치에 삽입하여 수익을 창출한다.

게임 진행 중(토론/투표/밤 페이즈)에는 광고를 노출하지 않는다. Sub-project A UX 스펙과 충돌하며, AdSense 정책상 게임 인터페이스 위 광고도 제한될 수 있다.

---

## Goals / Non-Goals

**Goals:**
- 대기실(Waiting Room)에 AdSense 배너 삽입
- 게임 종료 화면(ResultOverlay)에 AdSense 배너 삽입
- 개발 환경에서는 광고가 렌더링되지 않도록 처리
- Publisher ID / Slot ID를 환경변수로 관리

**Non-Goals:**
- 게임 진행 중 광고 노출
- 로비 페이지 광고 (향후 별도 추가 가능)
- AdSense Auto Ads (위치 제어 불가, 게임 방해 위험)
- 광고 수익 대시보드 연동

---

## Decisions

### 1. AdSense 스크립트 (`index.html`)

`<head>` 안에 AdSense 비동기 스크립트를 추가한다. Publisher ID는 빌드 시점에 환경변수로 주입한다.

```html
<!-- frontend/index.html -->
<script
  async
  src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=%VITE_ADSENSE_CLIENT%"
  crossorigin="anonymous"
></script>
```

단, Vite는 `index.html`에서 `%VITE_*%` 치환을 기본 지원하지 않으므로 `vite-plugin-html` 또는 빌드 후 치환 스크립트를 사용하거나, 스크립트 src를 고정하고 `data-ad-client`만 환경변수로 관리하는 방식을 택한다.

**권장 방식:** `index.html`의 스크립트 src에 Publisher ID를 직접 작성하고, `AdBanner` 컴포넌트에서 `VITE_ADSENSE_CLIENT` 환경변수로 이중 검증한다.

```html
<!-- index.html — src에 Publisher ID 직접 기재 -->
<script
  async
  src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-XXXXXXXXXXXXXXXX"
  crossorigin="anonymous"
></script>
```

---

### 2. 환경변수

```bash
# .env.production
VITE_ADSENSE_CLIENT=ca-pub-XXXXXXXXXXXXXXXX
VITE_ADSENSE_SLOT_WAITING=1234567890
VITE_ADSENSE_SLOT_RESULT=0987654321

# .env.development (비워두면 AdBanner가 렌더링 건너뜀)
VITE_ADSENSE_CLIENT=
VITE_ADSENSE_SLOT_WAITING=
VITE_ADSENSE_SLOT_RESULT=
```

Slot ID는 AdSense 대시보드에서 광고 단위를 생성한 후 발급받는 값이다.

---

### 3. `AdBanner` 컴포넌트 (신규)

```tsx
// frontend/src/components/AdBanner.tsx
import { useEffect } from 'react'

interface Props {
  slotId: string
  style?: React.CSSProperties
}

export default function AdBanner({ slotId, style }: Props) {
  const client = import.meta.env.VITE_ADSENSE_CLIENT

  useEffect(() => {
    try {
      ((window as any).adsbygoogle = (window as any).adsbygoogle ?? []).push({})
    } catch {
      // dev 환경에서 AdSense 스크립트 없을 때 무시
    }
  }, [])

  // VITE_ADSENSE_CLIENT 없으면 렌더링 건너뜀 (dev 환경 대응)
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

**동작 규칙:**
- `VITE_ADSENSE_CLIENT` 또는 `slotId`가 비어있으면 `null` 반환 → dev 환경 자동 비활성화
- `useEffect`에서 `adsbygoogle.push({})` 호출 → 컴포넌트 마운트 시 AdSense가 광고를 채움
- 동일 슬롯 ID로 두 번 push되지 않도록 컴포넌트는 한 번만 마운트되어야 함 (WaitingRoom, ResultOverlay 각각 독립 슬롯 ID 사용)

---

### 4. 배치

#### 대기실 (`WaitingRoom.tsx`)

플레이어 목록 아래, 시작 버튼 위에 삽입한다.

```tsx
// WaitingRoom.tsx — 플레이어 목록 div 닫는 태그 다음
<AdBanner
  slotId={import.meta.env.VITE_ADSENSE_SLOT_WAITING}
  style={{ marginBottom: '24px' }}
/>
```

#### 게임 종료 화면 (`ResultOverlay.tsx`)

플레이어 결과 목록 아래, "다시 하기" 버튼 위에 삽입한다.

```tsx
// ResultOverlay.tsx — 결과 목록 다음
<AdBanner
  slotId={import.meta.env.VITE_ADSENSE_SLOT_RESULT}
  style={{ marginTop: '24px', marginBottom: '16px' }}
/>
```

---

## Files Changed

| 파일 | 변경 유형 |
|---|---|
| `frontend/index.html` | 수정 (AdSense 스크립트 추가) |
| `frontend/.env.production` | 수정 (환경변수 추가) |
| `frontend/.env.development` | 수정 (빈 값으로 추가) |
| `frontend/src/components/AdBanner.tsx` | 신규 |
| `frontend/src/components/WaitingRoom.tsx` | 수정 (AdBanner 삽입) |
| `frontend/src/components/ResultOverlay.tsx` | 수정 (AdBanner 삽입) |

---

## Risks / Trade-offs

- **AdSense 심사**: AdSense는 사이트 심사 후 광고가 노출됨. 초기 배포 시 광고가 보이지 않을 수 있음 — `VITE_ADSENSE_CLIENT`가 없으면 `AdBanner`가 렌더링 안 되므로 빈 공간 문제 없음.
- **광고 블로커**: 광고 블로커 사용자는 광고를 볼 수 없음. `AdBanner`가 `null`을 반환해도 레이아웃이 깨지지 않도록 설계됨.
- **dev/prod 환경 분리**: `.env.development`에 값이 없으면 광고가 뜨지 않아 개발 중 레이아웃이 달라질 수 있음. 필요 시 `AdBanner`에 dev 전용 placeholder를 추가할 수 있으나 현재는 YAGNI.
- **`index.html` Publisher ID 노출**: AdSense Publisher ID는 공개 정보이므로 클라이언트 코드에 포함되어도 무방.
