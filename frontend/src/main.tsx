import { StrictMode, useEffect, type ReactNode } from 'react'
import { createRoot } from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'
import './index.css'
import LandingPage from './pages/LandingPage'
import LobbyPage from './pages/LobbyPage'
import RoomPage from './pages/RoomPage'
import ProfilePage from './pages/ProfilePage'
import { useAuthStore } from './store/authStore'

function AppInit({ children }: { children: ReactNode }) {
  useEffect(() => {
    void useAuthStore.getState().initialize()
  }, [])
  return <>{children}</>
}

const router = createBrowserRouter([
  { path: '/', element: <LandingPage /> },
  { path: '/lobby', element: <LobbyPage /> },
  { path: '/rooms/:id', element: <RoomPage /> },
  { path: '/profile', element: <ProfilePage /> },
])

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <AppInit>
      <RouterProvider router={router} />
    </AppInit>
  </StrictMode>,
)
