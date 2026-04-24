export type Phase = 'day_discussion' | 'day_vote' | 'night' | 'result'

export type Role = 'mafia' | 'police' | 'citizen' | ''

export interface Player {
  id: string
  name: string
  is_alive: boolean
  is_ai: boolean
}

export interface Room {
  id: string
  name: string
  status: 'waiting' | 'playing' | 'finished'
  host_id: string
  visibility: 'public' | 'private'
  // Private rooms carry a 6-letter join code. For public rooms the HTTP
  // endpoint omits the key entirely (omitempty) and the WS path sends an
  // empty string — treat absence and empty string equivalently.
  join_code?: string
  max_humans: number
  players: Player[]
}

export interface GameSnapshot {
  phase: Phase
  round: number
  timer_remaining_sec: number
  alive_player_ids: string[]
  votes: Record<string, string>
}

export interface GameOverResultPlayer {
  id: string
  name: string
  role: Role
  is_ai: boolean
  survived: boolean
}

export interface GameOverResult {
  winner: 'mafia' | 'citizen' | 'aborted'
  round: number
  duration_sec: number
  players: GameOverResultPlayer[]
  // Set only when winner === 'aborted' (e.g. 'all_humans_left').
  // Frontend should route to a "game aborted" view instead of the normal result screen.
  reason?: string
}

export interface ChatMessage {
  id: string
  player_id: string
  player_name?: string
  message: string
  mafia_only: boolean
  is_system?: boolean
}

export type WsStatus = 'connecting' | 'connected' | 'reconnecting' | 'disconnected'

export type WsEvent =
  | { type: 'initial_state'; payload: { room: Room; game: GameSnapshot | null; my_role: Role } }
  | { type: 'role_assigned'; payload: { role: Role } }
  | { type: 'phase_change'; payload: { phase: Phase; round?: number; duration?: number; alive_players?: string[] } }
  | { type: 'chat'; payload: { sender_id: string; sender_name: string; message: string; mafia_only?: boolean } }
  | { type: 'mafia_chat'; payload: { sender_id: string; sender_name: string; message: string } }
  | { type: 'vote'; payload: { voter_id?: string; target_id?: string; result?: string; votes?: Record<string, string> } }
  | { type: 'kill'; payload: { player_id: string; role?: string; reason?: string } }
  | { type: 'night_action'; payload: { type: string; target_id: string; is_mafia: boolean } }
  | { type: 'game_over'; payload: GameOverResult }
  | { type: 'player_replaced'; payload: { player_id: string; message: string } }
  | { type: 'player_joined'; payload: { player_id: string; player_name: string } }
  | { type: 'player_left'; payload: { player_id: string; player_name: string } }
