## ADDED Requirements

### Requirement: 공개 방 목록 표시
로비 화면은 서버의 공개 방 목록을 표시해야 한다.

#### Scenario: 공개 방 목록 조회
- **WHEN** 유저가 로비(`/`)에 접속하면
- **THEN** `GET /api/rooms` 응답의 방 목록이 표시된다
- **THEN** 각 방에 이름, 현재 플레이어 수, 상태(대기중/진행중)가 표시된다

#### Scenario: 빈 방 목록
- **WHEN** 공개 방이 없으면
- **THEN** "아직 방이 없습니다" 메시지가 표시된다

### Requirement: 방 생성
유저는 새로운 방을 만들 수 있어야 한다.

#### Scenario: 방 생성 성공
- **WHEN** 유저가 방 이름을 입력하고 생성 버튼을 누르면
- **THEN** `POST /api/rooms` 요청이 전송된다
- **THEN** 응답의 `player_id`가 localStorage에 저장된다
- **THEN** `/rooms/:id` 대기실 화면으로 이동한다

### Requirement: 코드로 비공개 방 참가
유저는 6자리 코드로 비공개 방에 참가할 수 있어야 한다.

#### Scenario: 코드 참가 성공
- **WHEN** 유저가 6자리 코드와 닉네임을 입력하고 참가 버튼을 누르면
- **THEN** `POST /api/rooms/join/code` 요청이 전송된다
- **THEN** 응답의 `player_id`가 localStorage에 저장된다
- **THEN** `/rooms/:id` 대기실 화면으로 이동한다

#### Scenario: 잘못된 코드
- **WHEN** 존재하지 않는 코드를 입력하면
- **THEN** "방을 찾을 수 없습니다" 오류 메시지가 표시된다
