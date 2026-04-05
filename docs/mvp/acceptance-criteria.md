# 승인 기준

다음 조건이 모두 충족되면 MVP가 완료된 것으로 봅니다.

1. `go test ./...`가 통과합니다.
2. `go build ./...`가 통과합니다.
3. `ccagent help`가 정상 종료합니다.
4. `ccagent doctor`가 해석된 로컬 진단 정보를 출력합니다.
5. `ccagent login --api-key KEY`가 API-key 자격 증명을 로컬에 저장합니다.
6. `ccagent login --device-auth`가 호환되는 auth 서버에 대해 완료되고 Codex 호환 토큰을 로컬에 저장할 수 있습니다.
7. 자격 증명이 존재할 때 `ccagent chat`이 한 턴을 실행할 수 있습니다.
8. 워크스페이스 도구가 파일 목록, 파일 읽기, 텍스트 검색을 수행할 수 있습니다.
9. shell 명령은 명시적 승인이 필요합니다.
10. 파일 수정은 명시적 승인이 필요합니다.
11. git 저장소 안에서 Git status와 diff가 동작합니다.
12. README, AGENTS, ADR, CI가 실제 동작과 일치한 상태로 존재합니다.
