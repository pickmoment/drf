package app

import (
	"os"
	"time"

	"github.com/pickmoment/drf/config"
	"github.com/pickmoment/drf/fs"
	"github.com/pickmoment/drf/state"
)

type AppMode int

const (
	ModeFileList AppMode = iota
	ModeViewer
	ModeOpenWith
	ModeSettings
	ModeCommandPalette
	ModeHelp
	ModeGit
	ModeFileManager
	ModePathClipboard
	ModeOpenChoice
)

type FocusedPanel int

const (
	PanelFileList FocusedPanel = iota
	PanelBookmarks
	PanelPathClipboard
)

type FmOp int

const (
	FmOpCopy FmOp = iota
	FmOpMove
	FmOpRename
	FmOpDelete
	FmOpNewDir
)

type FileType int

const (
	FileTypeText     FileType = iota
	FileTypeMarkdown
	FileTypeCode
	FileTypeImage
	FileTypePdf
	FileTypeCsv
	FileTypeParquet
	FileTypeArchive
	FileTypeUnknown
	FileTypeBinary
)

type CodeLang string

type StatusKind int

const (
	StatusSuccess StatusKind = iota
	StatusError
	StatusInfo
)

type StatusMessage struct {
	Text      string
	Kind      StatusKind
	ExpiresAt time.Time
}

func NewStatusSuccess(text string) *StatusMessage {
	return &StatusMessage{Text: text, Kind: StatusSuccess, ExpiresAt: time.Now().Add(3 * time.Second)}
}

func NewStatusError(text string) *StatusMessage {
	return &StatusMessage{Text: text, Kind: StatusError, ExpiresAt: time.Now().Add(5 * time.Second)}
}

func (s *StatusMessage) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

type PendingTerminalOpener struct {
	Cmd  string
	Args []string
}

type Model struct {
	Width  int
	Height int

	CurrentDir      string
	FileEntries     []fs.FileEntry
	SelectedIndex   int
	FilteredIndices []int

	Mode         AppMode
	FocusedPanel FocusedPanel

	Config config.Config

	ShouldQuit bool

	PreviewScroll       int
	PreviewHScroll      int
	PreviewWrap         bool
	PreviewLineNumbers  bool
	PreviewContentWidth int // inner content width of preview panel (set each frame)

	BookmarkIndex int

	SearchQuery string
	IsSearching bool

	ViewerSearchQuery   string
	ViewerIsSearching   bool
	ViewerSearchMatches []int
	ViewerSearchIdx     int
	ViewerGotoInput     string
	ViewerIsGoto        bool
	ViewerPrevKeyG      bool

	OpenWithIndex int

	OpenChoiceIndex int
	OpenChoiceIsDir bool

	FmMenuIdx         int
	FmInput           string
	FmCursor          int
	FmOperation       *FmOp
	FmError           string
	FmOverwriteTarget string

	PathClipboard    []string
	PathClipboardIdx int

	FileListHeight int
	ViewerHeight   int

	Status *StatusMessage

	Git *state.GitState

	PendingEditorOpen     bool
	PendingTerminalOpener *PendingTerminalOpener

	// Viewer state (fullscreen + preview panel)
	ViewerLines []string
	ViewerPath  string

	// AsyncResult receives the error from the running git async command.
	AsyncResult chan error
}

func New() (*Model, error) {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	entries, err := fs.ListDir(dir)
	if err != nil {
		entries = []fs.FileEntry{}
	}

	filtered := makeRange(len(entries))
	gitState := state.NewGitState(dir)

	m := &Model{
		CurrentDir:      dir,
		FileEntries:     entries,
		FilteredIndices: filtered,
		Mode:            ModeFileList,
		FocusedPanel:    PanelFileList,
		Config:          cfg,
		PreviewWrap:     cfg.Preview.Wrap,
		Git:             gitState,
	}
	return m, nil
}

func makeRange(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i
	}
	return s
}

func (m *Model) SetStatusSuccess(msg string) {
	m.Status = NewStatusSuccess(msg)
}

func (m *Model) SetStatusError(msg string) {
	m.Status = NewStatusError(msg)
}

func (m *Model) SelectedPath() string {
	if m.SelectedIndex >= len(m.FilteredIndices) {
		return ""
	}
	idx := m.FilteredIndices[m.SelectedIndex]
	if idx >= len(m.FileEntries) {
		return ""
	}
	return m.FileEntries[idx].Path
}

func (m *Model) SelectedEntry() *fs.FileEntry {
	if m.SelectedIndex >= len(m.FilteredIndices) {
		return nil
	}
	idx := m.FilteredIndices[m.SelectedIndex]
	if idx >= len(m.FileEntries) {
		return nil
	}
	return &m.FileEntries[idx]
}

func DetectFileType(path string) (FileType, CodeLang) {
	ext := ""
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			ext = lowerStr(path[i+1:])
			break
		}
		if path[i] == '/' {
			break
		}
	}
	switch ext {
	case "md", "markdown":
		return FileTypeMarkdown, ""
	case "png", "jpg", "jpeg", "gif", "webp", "svg", "bmp":
		return FileTypeImage, ""
	case "pdf":
		return FileTypePdf, ""
	case "csv", "tsv":
		return FileTypeCsv, ""
	case "parquet":
		return FileTypeParquet, ""
	case "zip", "tar", "gz", "bz2", "xz", "7z", "rar":
		return FileTypeArchive, ""
	case "exe", "dll", "so", "dylib", "o", "a", "out", "elf", "bin", "class", "wasm":
		return FileTypeBinary, ""
	case "rs":
		return FileTypeCode, "rust"
	case "py":
		return FileTypeCode, "python"
	case "js", "mjs":
		return FileTypeCode, "javascript"
	case "ts", "tsx":
		return FileTypeCode, "typescript"
	case "go":
		return FileTypeCode, "go"
	case "c", "h":
		return FileTypeCode, "c"
	case "cpp", "cc", "cxx":
		return FileTypeCode, "cpp"
	case "java":
		return FileTypeCode, "java"
	case "sh", "bash", "zsh":
		return FileTypeCode, "bash"
	case "toml":
		return FileTypeCode, "toml"
	case "yaml", "yml":
		return FileTypeCode, "yaml"
	case "json":
		return FileTypeCode, "json"
	case "html", "htm":
		return FileTypeCode, "html"
	case "css":
		return FileTypeCode, "css"
	case "txt", "log", "conf", "ini":
		return FileTypeText, ""
	default:
		return FileTypeUnknown, ""
	}
}

func lowerStr(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
