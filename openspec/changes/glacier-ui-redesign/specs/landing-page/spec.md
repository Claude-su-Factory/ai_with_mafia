## ADDED Requirements

### Requirement: 게임 소개 랜딩 페이지
`/` 경로는 AI Mafia 게임을 소개하고 로비로 진입하는 랜딩 페이지를 표시해야 한다.

#### Scenario: 랜딩 페이지 진입
- **WHEN** 유저가 `/`에 접속하면
- **THEN** AI Mafia 히어로 섹션(제목, 설명, CTA 버튼)이 표시된다
- **THEN** 마피아, 시민, 경찰 역할 소개 카드가 표시된다

#### Scenario: 로비 진입
- **WHEN** 유저가 "게임 시작하기" CTA 버튼을 클릭하면
- **THEN** `/lobby` 경로로 이동한다
