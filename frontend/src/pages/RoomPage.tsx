import { useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useGameStore } from '../store/gameStore'
import WaitingRoom from '../components/WaitingRoom'
import GameRoom from '../components/GameRoom'
import ResultOverlay from '../components/ResultOverlay'

export default function RoomPage() {
  const { id: roomID } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { connect, disconnect, room, result } = useGameStore()

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

  if (!room) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: '100vh', background: '#0E0C09', color: '#786F62', fontFamily: "'JetBrains Mono', monospace", fontSize: '11px', textTransform: 'uppercase', letterSpacing: '0.1em' }}>
        CONNECTING...
      </div>
    )
  }

  return (
    <div style={{ minHeight: '100vh', background: '#0E0C09', position: 'relative' }}>
      {room.status === 'waiting' || room.status === 'finished' ? (
        <WaitingRoom />
      ) : (
        <GameRoom />
      )}
      {result && <ResultOverlay />}
    </div>
  )
}
