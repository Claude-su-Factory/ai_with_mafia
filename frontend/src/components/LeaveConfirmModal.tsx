const T = {
  bg: '#0E0C09', surface: '#181410', surfaceBorder: '#2E2820',
  accent: '#C4963A', accentDim: 'rgba(196,150,58,0.12)',
  text: '#ECE7DE', textMuted: '#786F62',
  danger: '#8C1F1F', dangerDim: 'rgba(140,31,31,0.15)',
}
const SERIF = "'Instrument Serif', Georgia, serif"
const SANS  = "'DM Sans', system-ui, sans-serif"
const MONO  = "'JetBrains Mono', monospace"

interface Props {
  isOpen: boolean
  onConfirm: () => void
  onCancel: () => void
}

export default function LeaveConfirmModal({ isOpen, onConfirm, onCancel }: Props) {
  if (!isOpen) return null

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 60,
      background: 'rgba(14,12,9,0.92)', backdropFilter: 'blur(8px)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '16px',
    }}>
      <div style={{
        background: T.surface, border: `1px solid ${T.surfaceBorder}`,
        borderRadius: '4px', width: '100%', maxWidth: '400px',
        overflow: 'hidden',
      }}>
        {/* Header */}
        <div style={{
          padding: '28px 28px 0',
        }}>
          <div style={{
            fontFamily: MONO, fontSize: '10px',
            color: T.danger,
            textTransform: 'uppercase', letterSpacing: '0.12em', marginBottom: '8px',
          }}>
            게임 진행 중
          </div>
          <h2 style={{
            fontFamily: SERIF, fontSize: '24px', color: T.text,
            margin: '0 0 16px', letterSpacing: '-0.02em', lineHeight: 1.3,
          }}>
            게임에서 나가시겠습니까?
          </h2>
        </div>

        {/* Body */}
        <div style={{
          padding: '0 28px 24px',
        }}>
          <p style={{
            fontFamily: SANS, fontSize: '13px', color: T.textMuted,
            margin: 0, lineHeight: 1.6,
          }}>
            나가면 AI가 당신의 역할을 대신하며 게임은 계속 진행됩니다.
          </p>
        </div>

        {/* Actions */}
        <div style={{
          display: 'flex', gap: '8px', padding: '20px 28px',
          borderTop: `1px solid ${T.surfaceBorder}`,
        }}>
          <button
            onClick={onConfirm}
            style={{
              flex: 1, padding: '11px 0', cursor: 'pointer',
              borderRadius: '2px', transition: 'all 150ms ease',
              background: T.dangerDim, color: T.danger,
              border: `1px solid ${T.danger}50`,
              fontFamily: SANS, fontSize: '13px', fontWeight: 500,
            }}
          >
            나가기
          </button>
          <button
            onClick={onCancel}
            style={{
              flex: 1, padding: '11px 0', cursor: 'pointer',
              borderRadius: '2px', transition: 'all 150ms ease',
              background: T.accentDim, color: T.accent,
              border: `1px solid ${T.accent}50`,
              fontFamily: SANS, fontSize: '13px', fontWeight: 600,
            }}
          >
            계속 플레이
          </button>
        </div>
      </div>
    </div>
  )
}
