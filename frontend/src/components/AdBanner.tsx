import { useEffect, useRef, useState } from 'react'
import { logAdImpression } from '../api'

// 세션 단위 쿨다운 — 같은 slot 이 30초 내 중복 impression 전송 안 되도록.
// 모듈 스코프로 유지 → 같은 브라우저 탭에서 컴포넌트가 다시 mount 돼도 쿨다운 유지.
const impressionCooldownMs = 30_000
const lastLogged: Record<string, number> = {}

type Slot = 'lobby' | 'waiting' | 'result'

interface Props {
  slot: Slot
  /**
   * Required when `slot === 'waiting' | 'result'` (a specific game/room context).
   * Must be undefined when `slot === 'lobby'` (backend uses daily sentinel row).
   */
  gameID?: string
}

const slotEnvKey: Record<Slot, string | undefined> = {
  lobby: undefined, // Phase A 에서는 lobby 슬롯 ID 를 env 로 구분하지 않음
  waiting: import.meta.env.VITE_ADSENSE_SLOT_WAITING as string | undefined,
  result: import.meta.env.VITE_ADSENSE_SLOT_RESULT as string | undefined,
}

export default function AdBanner({ slot, gameID }: Props) {
  const ref = useRef<HTMLDivElement | null>(null)
  const [seen, setSeen] = useState(false)

  const client = import.meta.env.VITE_ADSENSE_CLIENT as string | undefined
  const slotID = slotEnvKey[slot]

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const io = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.intersectionRatio >= 0.5 && !seen) {
            setSeen(true)
            const now = Date.now()
            const last = lastLogged[slot] ?? 0
            if (now - last >= impressionCooldownMs) {
              lastLogged[slot] = now
              void logAdImpression(slot, gameID)
            }
          }
        }
      },
      { threshold: 0.5 },
    )
    io.observe(el)
    return () => io.disconnect()
  }, [slot, gameID, seen])

  // Reserved space prevents layout shift even when the ad slot is unconfigured.
  const wrapperStyle: React.CSSProperties = {
    minHeight: 90,
    width: '100%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  }

  // No AdSense env configured → dev placeholder, prod silent no-op.
  if (!client || !slotID) {
    if (import.meta.env.DEV) {
      return (
        <div
          ref={ref}
          style={{
            ...wrapperStyle,
            border: '1px dashed #4A4438',
            color: '#786F62',
            fontFamily: 'monospace',
            fontSize: 11,
          }}
        >
          [AD:{slot}]
        </div>
      )
    }
    return <div ref={ref} style={wrapperStyle} />
  }

  return (
    <div ref={ref} style={wrapperStyle}>
      <ins
        className="adsbygoogle"
        style={{ display: 'block', width: '100%' }}
        data-ad-client={client}
        data-ad-slot={slotID}
        data-ad-format="auto"
        data-full-width-responsive="true"
      />
    </div>
  )
}
