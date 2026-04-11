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
