package platform

import (
	"testing"

	"go.uber.org/zap"

	"ai-playground/internal/domain/dto"
	"ai-playground/internal/domain/entity"
)

// testRoomService returns a RoomService backed by in-memory state only (no DB).
func testRoomService(t *testing.T) *RoomService {
	t.Helper()
	return NewRoomService(nil, zap.NewNop())
}

// createTestRoom is a convenience helper that creates a public room with one host.
func createTestRoom(t *testing.T, svc *RoomService, name, hostID, hostName string, maxHumans int) *entity.Room {
	t.Helper()
	room, err := svc.Create(dto.CreateRoomRequest{
		Name:       name,
		MaxHumans:  maxHumans,
		Visibility: "public",
	}, hostID, hostName)
	if err != nil {
		t.Fatalf("createTestRoom: %v", err)
	}
	return room
}

// ─── Create ──────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	svc := testRoomService(t)
	room, err := svc.Create(dto.CreateRoomRequest{
		Name:       "테스트방",
		MaxHumans:  4,
		Visibility: "public",
	}, "host-1", "방장")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if room.ID == "" {
		t.Error("room ID should not be empty")
	}
	if room.Name != "테스트방" {
		t.Errorf("expected name '테스트방', got %q", room.Name)
	}
	if room.HostID != "host-1" {
		t.Errorf("expected host_id 'host-1', got %q", room.HostID)
	}
	if room.HumanCount() != 1 {
		t.Errorf("expected 1 human (host), got %d", room.HumanCount())
	}
	if room.GetStatus() != entity.RoomStatusWaiting {
		t.Errorf("expected status 'waiting', got %q", room.GetStatus())
	}
}

func TestCreate_InvalidMaxHumans_TooLow(t *testing.T) {
	svc := testRoomService(t)
	_, err := svc.Create(dto.CreateRoomRequest{Name: "방", MaxHumans: 0, Visibility: "public"}, "h", "호스트")
	if err == nil {
		t.Error("expected error for max_humans=0, got nil")
	}
}

func TestCreate_InvalidMaxHumans_TooHigh(t *testing.T) {
	svc := testRoomService(t)
	_, err := svc.Create(dto.CreateRoomRequest{Name: "방", MaxHumans: 7, Visibility: "public"}, "h", "호스트")
	if err == nil {
		t.Error("expected error for max_humans=7, got nil")
	}
}

func TestCreate_PrivateRoom_HasJoinCode(t *testing.T) {
	svc := testRoomService(t)
	room, err := svc.Create(dto.CreateRoomRequest{
		Name:       "비공개방",
		MaxHumans:  4,
		Visibility: "private",
	}, "host-1", "방장")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.JoinCode == "" {
		t.Error("private room should have a join code")
	}
	if len(room.JoinCode) != 6 {
		t.Errorf("join code should be 6 chars, got %d", len(room.JoinCode))
	}
}

// ─── GetByID ─────────────────────────────────────────────────────────────────

func TestGetByID_Success(t *testing.T) {
	svc := testRoomService(t)
	created := createTestRoom(t, svc, "방", "host-1", "방장", 4)

	got, err := svc.GetByID(created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("expected room id %q, got %q", created.ID, got.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := testRoomService(t)
	_, err := svc.GetByID("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent room, got nil")
	}
}

// ─── Join ─────────────────────────────────────────────────────────────────────

func TestJoin_Success(t *testing.T) {
	svc := testRoomService(t)
	room := createTestRoom(t, svc, "방", "host-1", "방장", 4)

	joined, err := svc.Join(room.ID, "player-2", "참가자")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if joined.HumanCount() != 2 {
		t.Errorf("expected 2 humans after join, got %d", joined.HumanCount())
	}
	if joined.PlayerByID("player-2") == nil {
		t.Error("player-2 should be in room after join")
	}
}

func TestJoin_RoomFull(t *testing.T) {
	svc := testRoomService(t)
	room := createTestRoom(t, svc, "방", "host-1", "방장", 1) // MaxHumans=1, 방장이 이미 1명

	_, err := svc.Join(room.ID, "player-2", "참가자")
	if err == nil {
		t.Error("expected error when joining full room, got nil")
	}
}

func TestJoin_RoomNotFound(t *testing.T) {
	svc := testRoomService(t)
	_, err := svc.Join("nonexistent", "player-1", "참가자")
	if err == nil {
		t.Error("expected error for nonexistent room, got nil")
	}
}

// ─── ListPublic ──────────────────────────────────────────────────────────────

func TestListPublic_ReturnsPublicRooms(t *testing.T) {
	svc := testRoomService(t)
	createTestRoom(t, svc, "공개방1", "host-1", "방장1", 4)
	createTestRoom(t, svc, "공개방2", "host-2", "방장2", 4)

	rooms := svc.ListPublic()
	if len(rooms) != 2 {
		t.Errorf("expected 2 public rooms, got %d", len(rooms))
	}
}

func TestListPublic_ExcludesPrivateRooms(t *testing.T) {
	svc := testRoomService(t)
	createTestRoom(t, svc, "공개방", "host-1", "방장1", 4)
	// 비공개 방 생성
	svc.Create(dto.CreateRoomRequest{Name: "비공개방", MaxHumans: 4, Visibility: "private"}, "host-2", "방장2") //nolint

	rooms := svc.ListPublic()
	if len(rooms) != 1 {
		t.Errorf("expected 1 public room (private excluded), got %d", len(rooms))
	}
}

func TestListPublic_EmptyWhenNoRooms(t *testing.T) {
	svc := testRoomService(t)
	rooms := svc.ListPublic()
	if len(rooms) != 0 {
		t.Errorf("expected 0 rooms, got %d", len(rooms))
	}
}

// ─── RemovePlayer ─────────────────────────────────────────────────────────────

func TestRemovePlayer_LastHumanDeletesRoom(t *testing.T) {
	svc := testRoomService(t)
	room := createTestRoom(t, svc, "방", "host-1", "방장", 4)
	roomID := room.ID

	result := svc.RemovePlayer(roomID, "host-1")
	if result != nil {
		t.Error("expected nil (room deleted) when last human leaves")
	}

	// 방이 메모리에서 삭제됐는지 확인
	_, err := svc.GetByID(roomID)
	if err == nil {
		t.Error("room should be deleted from memory after last human leaves")
	}
}

func TestRemovePlayer_TransfersHostOnLeave(t *testing.T) {
	svc := testRoomService(t)
	room := createTestRoom(t, svc, "방", "host-1", "방장", 4)
	svc.Join(room.ID, "player-2", "참가자") //nolint

	result := svc.RemovePlayer(room.ID, "host-1")
	if result == nil {
		t.Fatal("expected room to remain after non-last human leaves")
	}
	if result.HostID == "host-1" {
		t.Error("host should have been transferred after original host left")
	}
	if result.HostID != "player-2" {
		t.Errorf("expected new host to be 'player-2', got %q", result.HostID)
	}
}

func TestRemovePlayer_StaysWithRemainingPlayers(t *testing.T) {
	svc := testRoomService(t)
	room := createTestRoom(t, svc, "방", "host-1", "방장", 4)
	svc.Join(room.ID, "player-2", "참가자") //nolint
	svc.Join(room.ID, "player-3", "참가자2") //nolint

	result := svc.RemovePlayer(room.ID, "player-2")
	if result == nil {
		t.Fatal("expected room to remain")
	}
	if result.HumanCount() != 2 {
		t.Errorf("expected 2 remaining humans, got %d", result.HumanCount())
	}
	if result.PlayerByID("player-2") != nil {
		t.Error("player-2 should have been removed")
	}
}

func TestRemovePlayer_AINotCountedForRoomDeletion(t *testing.T) {
	svc := testRoomService(t)
	room := createTestRoom(t, svc, "방", "host-1", "방장", 4)

	// AI 플레이어 직접 추가
	aiPlayer := entity.NewPlayer("ai-0", "AI봇", true)
	room.AddPlayer(aiPlayer)

	// 유일한 인간 플레이어(방장)가 나감 → AI만 남아도 방은 삭제돼야 함
	result := svc.RemovePlayer(room.ID, "host-1")
	if result != nil {
		t.Error("room should be deleted when no humans remain, even if AI players exist")
	}
}

// ─── ToRoomResponse ───────────────────────────────────────────────────────────

func TestToRoomResponse_ExcludesAIPlayers(t *testing.T) {
	svc := testRoomService(t)
	room := createTestRoom(t, svc, "방", "host-1", "방장", 4)

	// AI 플레이어 추가
	room.AddPlayer(entity.NewPlayer("ai-0", "AI봇1", true))
	room.AddPlayer(entity.NewPlayer("ai-1", "AI봇2", true))

	resp := ToRoomResponse(room)
	for _, p := range resp.Players {
		if p.IsAI {
			t.Errorf("AI player %q should be excluded from ToRoomResponse", p.ID)
		}
	}
	if len(resp.Players) != 1 {
		t.Errorf("expected 1 human player in response, got %d", len(resp.Players))
	}
}

func TestToRoomResponse_Fields(t *testing.T) {
	svc := testRoomService(t)
	room := createTestRoom(t, svc, "내 방", "host-1", "방장", 6)

	resp := ToRoomResponse(room)
	if resp.ID == "" {
		t.Error("response ID should not be empty")
	}
	if resp.Name != "내 방" {
		t.Errorf("expected name '내 방', got %q", resp.Name)
	}
	if resp.HostID != "host-1" {
		t.Errorf("expected host_id 'host-1', got %q", resp.HostID)
	}
	if resp.MaxHumans != 6 {
		t.Errorf("expected max_humans 6, got %d", resp.MaxHumans)
	}
	if resp.Status != "waiting" {
		t.Errorf("expected status 'waiting', got %q", resp.Status)
	}
	if resp.Visibility != "public" {
		t.Errorf("expected visibility 'public', got %q", resp.Visibility)
	}
}

func TestToRoomResponse_PrivateRoom_HasJoinCode(t *testing.T) {
	svc := testRoomService(t)
	room, _ := svc.Create(dto.CreateRoomRequest{
		Name: "비공개방", MaxHumans: 4, Visibility: "private",
	}, "host-1", "방장")

	resp := ToRoomResponse(room)
	if resp.JoinCode == "" {
		t.Error("private room response should include join_code")
	}
}

func TestToRoomResponse_PublicRoom_NoJoinCode(t *testing.T) {
	svc := testRoomService(t)
	room := createTestRoom(t, svc, "공개방", "host-1", "방장", 4)

	resp := ToRoomResponse(room)
	if resp.JoinCode != "" {
		t.Errorf("public room response should not include join_code, got %q", resp.JoinCode)
	}
}

// ─── FindOrCreatePublicRoom (Quick Match helper, Phase A §3-C) ──────────────

func TestFindOrCreatePublicRoom_NoRoom_Creates(t *testing.T) {
	svc := testRoomService(t)
	room, created, err := svc.FindOrCreatePublicRoom("player-a", "알파")
	if err != nil {
		t.Fatalf("FindOrCreatePublicRoom: %v", err)
	}
	if !created {
		t.Error("created = false, want true (no existing rooms)")
	}
	if room.Visibility != entity.VisibilityPublic {
		t.Errorf("visibility = %v, want public", room.Visibility)
	}
}

func TestFindOrCreatePublicRoom_RoomFull_Creates(t *testing.T) {
	svc := testRoomService(t)
	// Fill one public room completely
	full := createTestRoom(t, svc, "가득찬방", "host-1", "호스트", 2)
	svc.Join(full.ID, "player-x", "X") // now HumanCount=2 == MaxHumans

	_, created, err := svc.FindOrCreatePublicRoom("player-a", "알파")
	if err != nil {
		t.Fatalf("FindOrCreatePublicRoom: %v", err)
	}
	if !created {
		t.Error("created = false, want true (only full room available)")
	}
}

func TestFindOrCreatePublicRoom_RoomAvailable_Joins(t *testing.T) {
	svc := testRoomService(t)
	target := createTestRoom(t, svc, "들어갈방", "host-1", "호스트", 4)

	room, created, err := svc.FindOrCreatePublicRoom("player-a", "알파")
	if err != nil {
		t.Fatalf("FindOrCreatePublicRoom: %v", err)
	}
	if created {
		t.Error("created = true, want false (available room exists)")
	}
	if room.ID != target.ID {
		t.Errorf("room.ID = %s, want %s", room.ID, target.ID)
	}
	if !roomContainsPlayer(room, "player-a") {
		t.Error("player-a was not added to the target room")
	}
}

func TestFindOrCreatePublicRoom_OnlyPrivate_Creates(t *testing.T) {
	svc := testRoomService(t)
	_, err := svc.Create(dto.CreateRoomRequest{
		Name: "비밀", MaxHumans: 4, Visibility: "private",
	}, "host-1", "호스트")
	if err != nil {
		t.Fatalf("Create private: %v", err)
	}

	_, created, err := svc.FindOrCreatePublicRoom("player-a", "알파")
	if err != nil {
		t.Fatalf("FindOrCreatePublicRoom: %v", err)
	}
	if !created {
		t.Error("created = false, want true (private rooms must be ignored)")
	}
}

// roomContainsPlayer is a small helper used by FindOrCreatePublicRoom tests.
func roomContainsPlayer(room *entity.Room, playerID string) bool {
	for _, p := range room.GetPlayers() {
		if p.ID == playerID {
			return true
		}
	}
	return false
}
