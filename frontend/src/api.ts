import { supabase } from './lib/supabase'

const BASE = '/api'

async function getAuthHeader(): Promise<Record<string, string>> {
  const { data: { session } } = await supabase.auth.getSession()
  if (session?.access_token) {
    return { Authorization: `Bearer ${session.access_token}` }
  }
  return {}
}

async function request<T>(path: string, options?: RequestInit & { headers?: Record<string, string> }): Promise<T> {
  const authHeader = await getAuthHeader()
  const { headers: optHeaders, ...restOptions } = options ?? {}
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...authHeader, ...optHeaders },
    ...restOptions,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({})) as { error?: string; room_id?: string }
    const err = new Error(body.error ?? `HTTP ${res.status}`) as Error & { roomID?: string }
    if (res.status === 409 && body.room_id) {
      err.roomID = body.room_id
    }
    throw err
  }
  return res.json() as Promise<T>
}

export interface CreateRoomParams {
  name: string
  visibility: 'public' | 'private'
  player_name: string
  max_humans?: number
}

export interface JoinRoomParams {
  room_id: string
  player_name: string
}

export interface JoinByCodeParams {
  code: string
  player_name: string
}

export interface JoinResponse {
  player_id: string
  id: string
}

export function listRooms() {
  return request<import('./types').Room[]>('/rooms')
}

export function createRoom(params: CreateRoomParams) {
  return request<JoinResponse>('/rooms', {
    method: 'POST',
    headers: { 'X-Player-Name': params.player_name },
    body: JSON.stringify({
      name: params.name,
      visibility: params.visibility,
      max_humans: params.max_humans ?? 6,
    }),
  })
}

export function joinRoom(params: JoinRoomParams) {
  return request<JoinResponse>(`/rooms/${params.room_id}/join`, {
    method: 'POST',
    body: JSON.stringify({ player_name: params.player_name }),
  })
}

export function joinByCode(params: JoinByCodeParams) {
  return request<JoinResponse>('/rooms/join/code', {
    method: 'POST',
    body: JSON.stringify(params),
  })
}

export function startGame(roomID: string) {
  return request<void>(`/rooms/${roomID}/start`, { method: 'POST' })
}

export function restartGame(roomID: string) {
  return request<void>(`/rooms/${roomID}/restart`, { method: 'POST' })
}
