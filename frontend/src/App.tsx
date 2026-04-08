import { Routes, Route } from 'react-router-dom'
import LandingPage from './pages/LandingPage'
import LobbyPage from './pages/LobbyPage'
import RoomPage from './pages/RoomPage'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<LandingPage />} />
      <Route path="/lobby" element={<LobbyPage />} />
      <Route path="/rooms/:id" element={<RoomPage />} />
    </Routes>
  )
}
