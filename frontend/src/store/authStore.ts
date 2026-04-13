import { create } from 'zustand'
import type { User } from '@supabase/supabase-js'
import { supabase } from '../lib/supabase'

interface AuthStore {
  user: User | null
  playerID: string
  displayName: string
  loading: boolean
  initialize: () => Promise<void>
  signInWithGoogle: () => Promise<void>
  signOut: () => Promise<void>
  getAccessToken: () => Promise<string>
}

let initialized = false

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  playerID: '',
  displayName: '',
  loading: true,

  async initialize() {
    if (initialized) return
    initialized = true

    const { data: { session } } = await supabase.auth.getSession()
    if (session?.user) {
      const res = await fetch('/api/me', {
        headers: { Authorization: `Bearer ${session.access_token}` },
      })
      if (res.ok) {
        const data = await res.json() as { player_id: string; display_name: string }
        set({ user: session.user, playerID: data.player_id, displayName: data.display_name, loading: false })
      } else {
        set({ user: session.user, loading: false })
      }
    } else {
      set({ loading: false })
    }

    const { data: { subscription: _authSubscription } } = supabase.auth.onAuthStateChange(async (event, session) => {
      if (event === 'SIGNED_IN' && session?.user) {
        const res = await fetch('/api/me', {
          headers: { Authorization: `Bearer ${session.access_token}` },
        })
        if (res.ok) {
          const data = await res.json() as { player_id: string; display_name: string }
          set({ user: session.user, playerID: data.player_id, displayName: data.display_name, loading: false })
        } else {
          set({ user: session.user, playerID: '', displayName: '', loading: false })
        }
      } else if (event === 'SIGNED_OUT') {
        set({ user: null, playerID: '', displayName: '' })
      }
    })
    void _authSubscription
  },

  async signInWithGoogle() {
    await supabase.auth.signInWithOAuth({
      provider: 'google',
      options: { redirectTo: `${window.location.origin}/lobby` },
    })
  },

  async signOut() {
    await supabase.auth.signOut()
    set({ user: null, playerID: '', displayName: '' })
  },

  async getAccessToken() {
    const { data: { session } } = await supabase.auth.getSession()
    return session?.access_token ?? ''
  },
}))
