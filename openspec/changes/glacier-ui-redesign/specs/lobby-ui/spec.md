## MODIFIED Requirements

### Requirement: 공개 방 목록 표시
로비 화면은 서버의 공개 방 목록을 표시해야 한다.

#### Scenario: 공개 방 목록 조회
- **WHEN** 유저가 로비(`/lobby`)에 접속하면
- **THEN** `GET /api/rooms` 응답(배열)의 방 목록이 표시된다
- **THEN** 각 방에 이름, 현재 플레이어 수, 상태(대기중/진행중)가 표시된다

#### Scenario: 빈 방 목록
- **WHEN** 공개 방이 없으면
- **THEN** "아직 방이 없습니다" 메시지가 표시된다

## ADDED Requirements

### Requirement: 랜딩 페이지로 복귀
로비 화면에서 랜딩 페이지로 돌아갈 수 있어야 한다.

#### Scenario: 홈으로 이동
- **WHEN** 유저가 로비의 홈 버튼 또는 로고를 클릭하면
- **THEN** `/` 랜딩 페이지로 이동한다
