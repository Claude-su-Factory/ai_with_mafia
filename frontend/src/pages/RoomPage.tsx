import { useEffect, useCallback } from 'react'
import { useParams, useNavigate, useBlocker } from 'react-router-dom'
import { useGameStore } from '../store/gameStore'
import WaitingRoom from '../components/WaitingRoom'
import GameRoom from '../components/GameRoom'
import ResultOverlay from '../components/ResultOverlay'
import LeaveConfirmModal from '../components/LeaveConfirmModal'
import CinematicOverlay from '../components/CinematicOverlay'

export default function RoomPage() {
  const { id: roomID } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { connect, disconnect, room, result } = useGameStore()

  // Block navigation only when game is actively playing and no result yet
  const shouldBlock = room?.status === 'playing' && !result

  const blocker = useBlocker(shouldBlock)

  // Handle browser tab close / refresh
  useEffect(() => {
    if (!shouldBlock) return

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [shouldBlock])

  useEffect(() => {
    if (!roomID) return
    const playerID = localStorage.getItem(`player_id_${roomID}`)
    if (!playerID) {
      navigate('/lobby')
      return
    }
    connect(roomID)
    return () => {
      disconnect()
    }
  }, [roomID])

  const handleLeaveConfirm = useCallback(() => {
    disconnect()
    if (blocker.state === 'blocked') {
      blocker.proceed()
    }
  }, [blocker, disconnect])

  const handleLeaveCancel = useCallback(() => {
    if (blocker.state === 'blocked') {
      blocker.reset()
    }
  }, [blocker])

  if (!room) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100dvh', background: '#0E0C09', color: '#786F62', fontFamily: "'JetBrains Mono', monospace", fontSize: '11px', textTransform: 'uppercase', letterSpacing: '0.1em' }}>
        CONNECTING...
      </div>
    )
  }

  return (
    <div style={{ minHeight: '100dvh', background: '#0E0C09', position: 'relative' }}>
      {room.status === 'waiting' || room.status === 'finished' ? (
        <WaitingRoom />
      ) : (
        <GameRoom />
      )}
      {result && <ResultOverlay />}
      <CinematicOverlay />
      <LeaveConfirmModal
        isOpen={blocker.state === 'blocked'}
        onConfirm={handleLeaveConfirm}
        onCancel={handleLeaveCancel}
      />
    </div>
  )
}
