import { create } from 'zustand'
import type { Room, Phase, Role, GameOverResult, ChatMessage, WsStatus, WsEvent } from '../types'

let ws: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let timerInterval: ReturnType<typeof setInterval> | null = null
let reconnectDelay = 1000
let currentRoomID = ''

function clearReconnectTimer() {
  if (reconnectTimer !== null) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

function clearTimerInterval() {
  if (timerInterval !== null) {
    clearInterval(timerInterval)
    timerInterval = null
  }
}

interface GameStore {
  // Session
  playerID: string
  myRole: Role

  // Room
  room: Room | null

  // Game (playing only)
  phase: Phase | null
  round: number
  timerRemainingSec: number
  alivePlayerIDs: string[]
  votes: Record<string, string>

  // Result (after game_over)
  result: GameOverResult | null

  // Chat
  messages: ChatMessage[]

  // WS
  wsStatus: WsStatus

  // Actions
  connect: (roomID: string) => void
  disconnect: () => void
  sendAction: (type: string, payload: Record<string, unknown>) => void
}

function startTimer(get: () => GameStore, set: (partial: Partial<GameStore>) => void, initialSec: number) {
  clearTimerInterval()
  set({ timerRemainingSec: initialSec })
  timerInterval = setInterval(() => {
    const current = get().timerRemainingSec
    if (current <= 0) {
      clearTimerInterval()
      return
    }
    set({ timerRemainingSec: current - 1 })
  }, 1000)
}

export const useGameStore = create<GameStore>((set, get) => ({
  playerID: '',
  myRole: '',
  room: null,
  phase: null,
  round: 0,
  timerRemainingSec: 0,
  alivePlayerIDs: [],
  votes: {},
  result: null,
  messages: [],
  wsStatus: 'disconnected',

  connect(roomID: string) {
    currentRoomID = roomID
    const playerID = localStorage.getItem(`player_id_${roomID}`) ?? ''
    set({ playerID, wsStatus: 'connecting' })

    clearReconnectTimer()
    if (ws) {
      ws.onclose = null
      ws.close()
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${window.location.host}/ws/rooms/${roomID}?player_id=${playerID}`
    ws = new WebSocket(url)

    ws.onopen = () => {
      reconnectDelay = 1000
      set({ wsStatus: 'connected' })
    }

    ws.onmessage = (e) => {
      let event: WsEvent
      try {
        event = JSON.parse(e.data)
      } catch {
        return
      }

      switch (event.type) {
        case 'initial_state': {
          const { room, game, my_role } = event.payload
          clearTimerInterval()
          const newState: Partial<GameStore> = {
            room,
            myRole: my_role,
            phase: game?.phase ?? null,
            round: game?.round ?? 0,
            alivePlayerIDs: game?.alive_player_ids ?? [],
            votes: game?.votes ?? {},
          }
          if (game) {
            startTimer(get, set, game.timer_remaining_sec)
          } else {
            newState.timerRemainingSec = 0
          }
          set(newState)
          break
        }

        case 'role_assigned': {
          set({ myRole: event.payload.role })
          break
        }

        case 'phase_change': {
          const { phase, round, duration, alive_players } = event.payload
          const updates: Partial<GameStore> = { phase }
          if (round !== undefined) updates.round = round
          if (alive_players !== undefined) updates.alivePlayerIDs = alive_players
          updates.votes = {}
          set((s) => ({ ...updates, room: s.room ? { ...s.room, status: 'playing' } : s.room }))
          if (duration !== undefined) {
            startTimer(get, set, duration)
          }
          break
        }

        case 'chat': {
          const msg: ChatMessage = {
            id: `${Date.now()}-${Math.random()}`,
            player_id: event.payload.sender_id,
            player_name: event.payload.sender_name,
            message: event.payload.message,
            mafia_only: event.payload.mafia_only ?? false,
          }
          set((s) => ({ messages: [...s.messages, msg] }))
          break
        }

        case 'mafia_chat': {
          const msg: ChatMessage = {
            id: `${Date.now()}-${Math.random()}`,
            player_id: event.payload.sender_id,
            player_name: event.payload.sender_name,
            message: event.payload.message,
            mafia_only: true,
          }
          set((s) => ({ messages: [...s.messages, msg] }))
          break
        }

        case 'vote': {
          const { voter_id, target_id, votes } = event.payload
          if (votes) {
            set({ votes })
          } else if (voter_id && target_id) {
            set((s) => ({ votes: { ...s.votes, [voter_id]: target_id } }))
          }
          break
        }

        case 'kill': {
          const { player_id, role } = event.payload
          set((s) => ({
            alivePlayerIDs: s.alivePlayerIDs.filter((id) => id !== player_id),
            room: s.room
              ? {
                  ...s.room,
                  players: s.room.players.map((p) =>
                    p.id === player_id ? { ...p, is_alive: false } : p
                  ),
                }
              : null,
            messages: [
              ...s.messages,
              {
                id: `${Date.now()}-${Math.random()}`,
                player_id: 'system',
                message: role
                  ? `플레이어가 사망했습니다. (역할: ${role})`
                  : '플레이어가 사망했습니다.',
                mafia_only: false,
                is_system: true,
              },
            ],
          }))
          break
        }

        case 'game_over': {
          clearTimerInterval()
          set({ result: event.payload, phase: null })
          break
        }

        case 'night_action': {
          if (event.payload.type === 'investigation_result') {
            const { target_id, is_mafia } = event.payload
            const room = get().room
            const target = room?.players.find((p) => p.id === target_id)
            const name = target?.name ?? target_id
            set((s) => ({
              messages: [
                ...s.messages,
                {
                  id: `${Date.now()}-${Math.random()}`,
                  player_id: 'system',
                  message: `${name}은(는) ${is_mafia ? '마피아입니다' : '마피아가 아닙니다'}.`,
                  mafia_only: false,
                  is_system: true,
                },
              ],
            }))
          }
          break
        }

        case 'player_replaced': {
          set((s) => ({
            messages: [
              ...s.messages,
              {
                id: `${Date.now()}-${Math.random()}`,
                player_id: 'system',
                message: event.payload.message,
                mafia_only: false,
                is_system: true,
              },
            ],
          }))
          break
        }
      }
    }

    ws.onclose = () => {
      set({ wsStatus: 'reconnecting' })
      reconnectTimer = setTimeout(() => {
        reconnectDelay = Math.min(reconnectDelay * 2, 10000)
        get().connect(currentRoomID)
      }, reconnectDelay)
    }
  },

  disconnect() {
    clearReconnectTimer()
    clearTimerInterval()
    if (ws) {
      ws.onclose = null
      ws.close()
      ws = null
    }
    set({ wsStatus: 'disconnected' })
  },

  sendAction(type: string, payload: Record<string, unknown>) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({ type, ...payload }))
  },
}))
