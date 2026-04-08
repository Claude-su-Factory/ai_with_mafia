## REMOVED Requirements

### Requirement: GameModule 등록
**Reason**: 다중 게임 플랫폼 지원을 포기하고 마피아 전용 플랫폼으로 단순화. GameModule/Registry 추상화가 실질적인 가치 없이 복잡성만 유발.
**Migration**: 해당 없음. 마피아 게임만 존재하며 직접 생성 방식으로 대체.

---

## MODIFIED Requirements

### Requirement: 방 생성
사용자는 방 이름, 최대 사람 인원수(1~6명), 공개/비공개 여부를 지정하여 방을 생성할 수 있어야 한다. 방을 생성한 사용자가 방장이 된다.

#### Scenario: 공개 방 생성
- **WHEN** 방 이름, max_humans(1~6), visibility=public으로 방 생성 요청을 보내면
- **THEN** 고유 ID를 가진 공개 방이 생성되고 방 정보가 반환되어야 한다

#### Scenario: 비공개 방 생성
- **WHEN** visibility=private으로 방 생성 요청을 보내면
- **THEN** 6자리 랜덤 영숫자 참가 코드가 생성되어 방 정보에 포함되어야 한다

#### Scenario: 잘못된 max_humans 값
- **WHEN** max_humans가 1 미만이거나 6 초과이면
- **THEN** 422 오류를 반환해야 한다
