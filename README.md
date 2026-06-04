# drf

터미널 파일 매니저. Go + [Bubbletea](https://github.com/charmbracelet/bubbletea) 기반.

## 설치

```bash
go install github.com/pickmoment/drf@latest
```

또는 소스에서 빌드:

```bash
git clone https://github.com/pickmoment/drf
cd drf
go build -o drf .
```

## 실행

```bash
drf
```

## 화면 구성

```
 drf  │  /home/user/projects                              ⎇ main ●  12 파일
┌ 즐겨찾기 ┐┌ 파일 목록 (12) ───────────────┐┌ 미리보기 ────────────────────┐
│ ~/projects││▶  src                         ││ package main                 │
│           ││   README.md             2.1K  ││                              │
│           ││   main.go               1.4K  ││ import "fmt"                 │
└───────────┘└───────────────────────────────┘└──────────────────────────────┘
[j/k] 이동  [Enter] 열기  [/] 검색  [b] 즐겨찾기  [g] Git  [?] 도움말  [Q] 종료
```

- **탭 바** — 현재 경로, Git 브랜치, 변경사항 존재 시 `●` 표시
- **즐겨찾기 패널** — 자주 가는 디렉토리 북마크
- **파일 목록** — Nerd Font 아이콘, 파일 크기
- **미리보기 패널** — 선택한 파일을 실시간으로 미리보기 (코드 하이라이팅, Markdown 렌더링 등)

## 키 바인딩

### 파일 목록

| 키 | 동작 |
|---|---|
| `j` / `k` / `↑` / `↓` | 이동 |
| `Home` / `End` | 목록 처음 / 끝 |
| `PgUp` / `PgDn` | 페이지 이동 |
| `Enter` | 열기 (기본 앱 / opener 선택) |
| `→` / `l` | 디렉토리 진입 / 파일 전체화면 뷰어로 열기 |
| `←` / `h` / `Backspace` | 상위 디렉토리 |
| `Space` | 전체화면 뷰어로 열기 |
| `Tab` | 패널 포커스 전환 (파일 목록 → 즐겨찾기 → 경로 클립보드) |
| `Esc` | 파일 목록 패널로 복귀 |
| `/` | 이름 검색 |
| `.` | 숨김 파일 토글 |
| `b` | 현재 디렉토리 북마크 토글 |
| `B` | 즐겨찾기 패널 포커스 |
| `p` | 경로 클립보드 토글 |
| `P` | 경로 클립보드 패널 |
| `y` | 선택 경로 클립보드 복사 |
| `o` | 프로그램으로 열기 |
| `e` | 편집기로 열기 |
| `n` | 새 폴더 |
| `r` | 이름 변경 |
| `c` | 복사 |
| `m` | 이동 |
| `d` | 삭제 |
| `w` / `W` | 미리보기 줄바꿈 토글 |
| `L` | 미리보기 줄번호 토글 |
| `g` | Git 패널 |
| `?` | 도움말 |
| `q` / `Q` | 종료 |

### 뷰어 (전체화면)

| 키 | 동작 |
|---|---|
| `j` / `k` / `↑` / `↓` | 한 줄 스크롤 |
| `d` / `u` | 반 페이지 스크롤 |
| `f` / `b` / `Space` / `PgDn` / `PgUp` | 한 페이지 스크롤 |
| `h` / `l` / `←` / `→` | 가로 스크롤 |
| `gg` | 맨 위 |
| `G` | 맨 아래 |
| `/` | 검색 |
| `n` / `N` | 다음 / 이전 검색 결과 |
| `:` | 줄 번호로 이동 |
| `y` | 현재 줄 클립보드 복사 |
| `w` / `W` | 줄바꿈 토글 |
| `L` | 줄번호 토글 |
| `q` / `Esc` | 닫기 |

### Git 패널

| 키 | 동작 |
|---|---|
| `j` / `k` / `↑` / `↓` | 파일 이동 |
| `Tab` | Staged ↔ Unstaged 전환 |
| `s` | 선택 파일 스테이지 (Staged 섹션에서는 전체 스테이지) |
| `u` | 선택 파일 언스테이지 |
| `a` | 전체 스테이지 |
| `A` | 전체 언스테이지 |
| `r` | 파일 되돌리기 (확인 후 실행) |
| `c` | 커밋 |
| `p` | Push |
| `P` | Pull |
| `F` | Force push (확인 후 실행) |
| `f` | Fetch |
| `R` | Git 상태 새로고침 |
| `b` | 브랜치 패널 |
| `L` | 커밋 로그 |
| `w` / `W` | diff 줄바꿈 토글 |
| `Enter` | diff 전체화면 |
| `q` / `Esc` | 닫기 |

#### 브랜치 패널

| 키 | 동작 |
|---|---|
| `j` / `k` / `↑` / `↓` | 브랜치 이동 |
| `Enter` | 브랜치 전환 |
| `n` | 새 브랜치 생성 및 전환 |
| `d` | 브랜치 삭제 |
| `D` | 브랜치 강제 삭제 |
| `q` / `Esc` | 닫기 |

#### 커밋 로그

| 키 | 동작 |
|---|---|
| `j` / `k` | 커밋 이동 |
| `Tab` | 로그 ↔ 변경 파일 ↔ diff 패널 전환 |
| `h` / `l` | 패널 간 이동 |
| `q` / `Esc` | 닫기 |

## 설정

설정 파일 위치: `~/.config/drf/config.toml`

```toml
[general]
start_dir = ""          # 시작 디렉토리 (비어 있으면 홈 디렉토리)
show_hidden = false
sort_by = "name"        # name | size | modified | extension
sort_descending = false

[ui]
theme = "default"
show_bookmarks_panel = true
show_preview_panel = true
show_hint_bar = true
show_icons = true       # Nerd Font 필요

[keymap]
vim_keys = true

[preview]
markdown_render = true
syntax_theme = "monokai"
max_file_size = 10485760  # 10MB
wrap = false
image_protocol = "auto"   # auto | kitty | iterm2 | sixel | braille

[[openers]]
name = "VS Code"
command = "code"
args = []
terminal = false

[bookmarks]
# 북마크 경로 목록 (앱에서 b 키로 관리)
```

## 미리보기 지원 형식

| 형식 | 지원 내용 |
|---|---|
| 코드 | 문법 하이라이팅 (Go, Python, JS, TS, Rust, Java 등) |
| Markdown | 렌더링 또는 원문 |
| CSV / TSV | 테이블 형식 |
| 이미지 | Kitty / iTerm2 / Sixel / 브라유 프로토콜 |
| PDF / 아카이브 / 바이너리 | 파일 정보 표시 |
| INI / conf / txt / log | 텍스트 (UTF-16 BOM 자동 변환) |

## 요구 사항

- Go 1.21+
- [Nerd Fonts](https://www.nerdfonts.com/) (아이콘 표시 시)
- Git (Git 기능 사용 시)
