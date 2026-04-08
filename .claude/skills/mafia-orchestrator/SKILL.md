---
name: mafia-orchestrator
description: "AI 마피아 게임 플랫폼의 에이전트 팀을 조율하는 오케스트레이터. 기능 구현, 버그 수정, QA 검증, 테스트 작성 등 백엔드·프론트엔드·QA가 함께 필요한 작업 시 반드시 이 스킬을 사용. 다시 실행, 업데이트, 이전 결과 개선, 부분 수정, 보완 요청 시에도 이 스킬을 사용."
---

# Mafia Platform Orchestrator

AI 마피아 게임 플랫폼의 Backend, Frontend, QA 에이전트를 조율하여 작업을 완료하는 통합 스킬.

## 실행 모드: 에이전트 팀

## 에이전트 구성

| 팀원 | 에이전트 타입 | 역할 | 스킬 | 출력 |
|------|-------------|------|------|------|
| backend | `backend` | Go 백엔드 구현·수정·테스트 | `backend-dev` | `_workspace/backend_*.md` |
| frontend | `frontend` | React 프론트엔드 구현·수정 | `frontend-dev` | `_workspace/frontend_*.md` |
| qa | `qa` | 경계면 검증·테스트 작성·버그 탐지 | `qa-test` | `_workspace/qa_report.md` |

## 워크플로우

### Phase 0: 컨텍스트 확인

1. `_workspace/` 존재 여부 확인
   - 미존재 → 초기 실행, Phase 1 진행
   - 존재 + 부분 수정 요청 → 해당 에이전트만 재호출
   - 존재 + 새 작업 → `_workspace/`를 `_workspace_{YYYYMMDD_HHMMSS}/`로 이동 후 Phase 1 진행

2. 요청 유형 분류:
   - **기능 구현**: Backend + Frontend 병렬 → QA 검증
   - **버그 수정**: 영향 레이어 파악 → 해당 에이전트만 투입 → QA 검증
   - **QA/테스트만**: QA 에이전트 단독 투입
   - **백엔드만**: Backend 에이전트 단독 투입
   - **프론트엔드만**: Frontend 에이전트 단독 투입

### Phase 1: 준비

1. 사용자 요청에서 파악:
   - 구현할 기능 또는 수정할 버그
   - 영향받는 레이어 (backend / frontend / both)
   - 완료 기준 (어떻게 확인할 것인가)

2. `_workspace/` 디렉토리 생성 (초기 실행 시)
3. 요청 분석 결과를 `_workspace/00_request.md`에 저장

### Phase 2: 팀 구성

요청 유형에 맞게 팀 구성:

**기능 구현 / 전체 버그 수정 시:**
```
TeamCreate(
  team_name: "mafia-dev-team",
  members: [
    { name: "backend",  agent_type: "backend",  model: "opus",
      prompt: "backend-dev 스킬을 읽고 [작업 내용]을 구현하라. 완료 시 리더와 QA에게 알린다." },
    { name: "frontend", agent_type: "frontend", model: "opus",
      prompt: "frontend-dev 스킬을 읽고 [작업 내용]을 구현하라. 완료 시 리더와 QA에게 알린다." },
    { name: "qa",       agent_type: "qa",       model: "opus",
      prompt: "qa-test 스킬을 읽고, backend/frontend 완료 알림을 받으면 경계면 검증 및 테스트를 실행하라." }
  ]
)
```

**백엔드 또는 프론트엔드만 필요한 경우:** 해당 에이전트 + QA만 구성

작업 등록:
```
TaskCreate(tasks: [
  { title: "백엔드 구현", assignee: "backend", description: "..." },
  { title: "프론트 구현", assignee: "frontend", description: "..." },
  { title: "QA 검증", assignee: "qa", depends_on: ["백엔드 구현", "프론트 구현"],
    description: "backend/frontend 완료 후 경계면 검증 및 go test ./... 실행" }
])
```

### Phase 3: 병렬 구현

팀원들이 자체 조율하며 작업을 수행한다.

**팀원 간 통신 규칙:**
- Backend가 DTO 변경 시 → Frontend에게 `SendMessage`로 변경 내역 즉시 공유
- Frontend가 WS payload 불일치 발견 시 → Backend에게 확인 요청
- Backend/Frontend 완료 시 → QA에게 "작업 완료" `SendMessage`
- QA가 버그 발견 시 → 해당 에이전트에게 버그 리포트 `SendMessage`

**리더 모니터링:**
- 팀원이 막히면 SendMessage로 방향 제시
- DTO 변경으로 Frontend가 영향받는 경우 backend-frontend 간 조율 지원

### Phase 4: QA 검증 및 완료

1. QA 에이전트가 검증 완료 알림 수신
2. `_workspace/qa_report.md` Read
3. 이슈 있으면 해당 에이전트에게 수정 지시 → 재검증
4. 이슈 없으면 Phase 5로 진행

### Phase 5: 정리

1. 팀원들에게 종료 `SendMessage`
2. `TeamDelete`
3. `_workspace/` 보존
4. 사용자에게 완료 보고:
   - 변경된 파일 목록
   - 빌드/테스트 결과
   - QA 이슈 요약

## 데이터 흐름

```
[리더]
  ↓ TeamCreate
[backend] ←SendMessage→ [frontend]
    ↓ 완료 알림             ↓ 완료 알림
              [qa]
                ↓ qa_report.md
[리더] → Read → 완료 보고
```

## 에러 핸들링

| 상황 | 전략 |
|------|------|
| 빌드 실패 | backend 에이전트에게 에러 메시지 전달, 수정 후 재빌드 |
| 테스트 실패 | QA가 분석 → 코드 버그 vs 테스트 버그 구분 → 해당 에이전트 수정 |
| DTO 불일치 | QA가 양쪽 에이전트에게 알림 → 합의 후 양쪽 수정 |
| 에이전트 중단 | SendMessage로 상태 확인 → 재시작 또는 나머지로 진행 |

## 테스트 시나리오

### 정상 흐름 (기능 구현)
1. 사용자: "새 기능 X를 추가해줘"
2. Phase 1: backend + frontend 모두 영향받음으로 분류
3. Phase 2: 3명 팀 구성, 3개 작업 등록
4. Phase 3: backend/frontend 병렬 구현, DTO 변경 시 공유
5. Phase 4: QA 검증, `go test ./...` 통과
6. Phase 5: 변경 파일 목록 + 테스트 결과 보고

### 에러 흐름 (테스트 실패)
1. Phase 4에서 QA가 `go test ./...` 실패 감지
2. 실패 케이스를 backend에게 SendMessage
3. backend가 수정 후 재빌드/재테스트
4. 테스트 통과 시 Phase 5 진행
5. 2회 실패 시 리더가 사용자에게 에러 내용 보고하고 방향 확인
