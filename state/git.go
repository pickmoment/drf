package state

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pickmoment/drf/git"
)

type ConfirmKind int

const (
	ConfirmDeleteBranchSoft ConfirmKind = iota
	ConfirmDeleteBranchForce
	ConfirmCheckoutFile
	ConfirmForcePush
)

type ConfirmData struct {
	Kind ConfirmKind
	Name string
}

type AsyncKind int

const (
	AsyncPush AsyncKind = iota
	AsyncPull
	AsyncFetch
)

type AsyncData struct {
	Kind   AsyncKind
	Force  bool
	Branch string
}

type GitState struct {
	Status *git.GitStatus

	StagedIdx   int
	UnstagedIdx int
	Section     GitSection

	Diff       []string
	DiffScroll int
	DiffHScroll int
	DiffWrap   bool
	DiffFullscreen bool
	DiffPanelHeight int

	IsCommitting bool
	CommitInput  string

	ShowLog        bool
	Log            []string
	LogFocused     bool
	LogIdx         int
	LogFileFocused bool
	CommitFiles    []git.CommitFileEntry
	CommitFileIdx  int
	CommitShow     []string
	CommitShowScroll int

	Branches        []git.BranchInfo
	BranchPanelOpen bool
	BranchIdx       int
	BranchInputActive bool
	BranchInput     string

	Confirm *ConfirmData

	AsyncData    *AsyncData
	AsyncCmd     *exec.Cmd
	AsyncStarted time.Time
	SpinnerTick  int
}

type GitSection int

const (
	GitSectionStaged   GitSection = iota
	GitSectionUnstaged
)

func NewGitState(dir string) *GitState {
	return &GitState{
		Status:  git.GetStatus(dir),
		Section: GitSectionUnstaged,
	}
}

func (g *GitState) Refresh(root string) {
	g.Status = git.GetStatus(root)
	if g.Status != nil {
		if g.StagedIdx >= len(g.Status.Staged) {
			g.StagedIdx = max(0, len(g.Status.Staged)-1)
		}
		if g.UnstagedIdx >= len(g.Status.Unstaged) {
			g.UnstagedIdx = max(0, len(g.Status.Unstaged)-1)
		}
	}
}

func (g *GitState) LoadDiff() {
	g.DiffScroll = 0
	g.DiffHScroll = 0
	if g.Status == nil {
		return
	}
	switch g.Section {
	case GitSectionStaged:
		if f := getFile(g.Status.Staged, g.StagedIdx); f != nil {
			g.Diff = git.GetDiff(g.Status.Root, f.Path, true)
		}
	case GitSectionUnstaged:
		if f := getFile(g.Status.Unstaged, g.UnstagedIdx); f != nil {
			g.Diff = git.GetDiff(g.Status.Root, f.Path, false)
		}
	}
}

func (g *GitState) LoadCommitShow() {
	if g.Status == nil {
		return
	}
	if g.LogIdx >= len(g.Log) {
		return
	}
	entry := g.Log[g.LogIdx]
	hash := strings.Fields(entry)[0]
	g.CommitFiles = git.GetCommitFiles(g.Status.Root, hash)
	g.CommitFileIdx = 0
	g.CommitShow = nil
	g.CommitShowScroll = 0
}

func (g *GitState) LoadCommitFileDiff() {
	if g.Status == nil || g.LogIdx >= len(g.Log) {
		return
	}
	entry := g.Log[g.LogIdx]
	hash := strings.Fields(entry)[0]
	if g.CommitFileIdx >= len(g.CommitFiles) {
		return
	}
	f := g.CommitFiles[g.CommitFileIdx]
	g.CommitShow = git.GetCommitFileDiff(g.Status.Root, hash, f.Path)
	g.CommitShowScroll = 0
	g.DiffHScroll = 0
}

func (g *GitState) StartAsync(data AsyncData, root string) *exec.Cmd {
	g.AsyncData = &data
	g.AsyncStarted = time.Now()
	g.SpinnerTick = 0

	var args []string
	switch data.Kind {
	case AsyncPush:
		args = git.PushArgs(data.Branch, data.Force)
	case AsyncPull:
		args = git.PullArgs()
	case AsyncFetch:
		args = git.FetchArgs()
	}

	all := append([]string{"-C", root}, args...)
	cmd := exec.Command("git", all...)
	cmd.Stdout = nil
	cmd.Stderr = os.NewFile(0, os.DevNull)
	return cmd
}

func getFile(files []git.GitFile, idx int) *git.GitFile {
	if idx < len(files) {
		return &files[idx]
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
