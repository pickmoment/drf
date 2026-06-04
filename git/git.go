package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type GitFile struct {
	Path string
	X    byte
	Y    byte
}

type GitStatus struct {
	Branch    string
	Root      string
	Staged    []GitFile
	Unstaged  []GitFile
	FileMap   map[string][2]byte
	Upstream  string
	Ahead     int
	Behind    int
	RemoteURL string
}

type BranchInfo struct {
	Name      string
	Hash      string
	Subject   string
	IsCurrent bool
	IsRemote  bool
}

type CommitFileEntry struct {
	Status byte
	Path   string
}

type GitError struct{ Msg string }

func (e *GitError) Error() string { return e.Msg }

func run(dir string, args ...string) (string, error) {
	all := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", all...).Output()
	return strings.TrimSpace(string(out)), err
}

func runWithStderr(dir string, args ...string) (string, string, error) {
	all := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", all...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()), err
}

func abbreviateRemoteURL(url string) string {
	if rest, ok := strings.CutPrefix(url, "git@"); ok {
		s := strings.TrimSuffix(rest, ".git")
		return strings.Replace(s, ":", "/", 1)
	}
	for _, p := range []string{"https://", "http://"} {
		if rest, ok := strings.CutPrefix(url, p); ok {
			return strings.TrimSuffix(rest, ".git")
		}
	}
	return url
}

func FindRoot(dir string) string {
	root, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return ""
	}
	return root
}

func GetStatus(dir string) *GitStatus {
	root := FindRoot(dir)
	if root == "" {
		return nil
	}

	branch, _ := run(root, "branch", "--show-current")
	if branch == "" {
		branch = "HEAD"
	}

	porcelain, err := run(root, "-c", "core.quotepath=false", "status", "--porcelain=v1", "-u")
	if err != nil {
		return nil
	}

	st := &GitStatus{
		Branch:  branch,
		Root:    root,
		FileMap: make(map[string][2]byte),
	}

	for _, line := range strings.Split(porcelain, "\n") {
		if len(line) < 3 {
			continue
		}
		x := line[0]
		y := line[1]
		rawPath := line[3:]
		path := rawPath
		if idx := strings.Index(rawPath, " -> "); idx != -1 {
			path = rawPath[idx+4:]
		}
		st.FileMap[path] = [2]byte{x, y}
		if x != ' ' && x != '?' {
			st.Staged = append(st.Staged, GitFile{Path: path, X: x, Y: y})
		}
		if y != ' ' || (x == '?' && y == '?') {
			st.Unstaged = append(st.Unstaged, GitFile{Path: path, X: x, Y: y})
		}
	}

	branchInfo, _ := run(root, "status", "--porcelain=v2", "--branch")
	for _, line := range strings.Split(branchInfo, "\n") {
		if rest, ok := strings.CutPrefix(line, "# branch.upstream "); ok {
			u := strings.TrimSpace(rest)
			if u != "(null)" {
				st.Upstream = u
			}
		} else if rest, ok := strings.CutPrefix(line, "# branch.ab "); ok {
			parts := strings.Fields(rest)
			if len(parts) >= 2 {
				if a, err := strconv.Atoi(parts[0][1:]); err == nil {
					st.Ahead = a
				}
				if b, err := strconv.Atoi(parts[1][1:]); err == nil {
					st.Behind = b
				}
			}
		}
	}

	if st.Upstream != "" {
		remote := strings.SplitN(st.Upstream, "/", 2)[0]
		if url, err := run(root, "remote", "get-url", remote); err == nil {
			st.RemoteURL = abbreviateRemoteURL(url)
		}
	}

	return st
}

func GetDiff(root, path string, staged bool) []string {
	var args []string
	if staged {
		args = []string{"diff", "--cached", "--", path}
	} else {
		args = []string{"diff", "--", path}
	}
	out, err := run(root, args...)
	if err == nil && out != "" {
		return strings.Split(out, "\n")
	}

	// untracked: show first 50 lines
	full := filepath.Join(root, path)
	data, err2 := os.ReadFile(full)
	if err2 == nil {
		lines := []string{
			fmt.Sprintf("# 새 파일 (untracked): %s", path),
			"",
		}
		for i, l := range strings.Split(string(data), "\n") {
			if i >= 50 {
				break
			}
			lines = append(lines, "+"+l)
		}
		return lines
	}
	return []string{"diff를 가져올 수 없습니다."}
}

func StageFile(root, path string) error {
	_, stderr, err := runWithStderr(root, "add", "--", path)
	if err != nil {
		return &GitError{Msg: stderr}
	}
	return nil
}

func UnstageFile(root, path string) error {
	_, stderr, err := runWithStderr(root, "restore", "--staged", "--", path)
	if err != nil {
		return &GitError{Msg: stderr}
	}
	return nil
}

func CommitChanges(root, message string) error {
	_, stderr, err := runWithStderr(root, "commit", "-m", message)
	if err != nil {
		return &GitError{Msg: stderr}
	}
	return nil
}

func GetCommitFiles(root, hash string) []CommitFileEntry {
	out, err := run(root, "diff-tree", "--root", "--no-commit-id", "-r", "--name-status", hash)
	if err != nil {
		return nil
	}
	var result []CommitFileEntry
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 {
			continue
		}
		status := line[0]
		path := strings.TrimSpace(line[1:])
		if path != "" {
			result = append(result, CommitFileEntry{Status: status, Path: path})
		}
	}
	return result
}

func GetCommitFileDiff(root, hash, path string) []string {
	out, err := run(root, "show", hash, "--", path)
	if err != nil {
		return []string{"diff를 가져올 수 없습니다."}
	}
	return strings.Split(out, "\n")
}

func GetLog(root string) []string {
	out, err := run(root, "log", "--oneline", "--decorate", "-20")
	if err != nil {
		return nil
	}
	lines := strings.Split(out, "\n")
	var result []string
	for _, l := range lines {
		if l != "" {
			result = append(result, l)
		}
	}
	return result
}

func ListBranches(root string) []BranchInfo {
	out, err := run(root, "for-each-ref",
		"--format=%(HEAD)|%(refname)|%(refname:short)|%(objectname:short)|%(contents:subject)",
		"refs/heads", "refs/remotes")
	if err != nil {
		return nil
	}
	var branches []BranchInfo
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 5)
		if len(parts) < 5 {
			continue
		}
		subject := parts[4]
		if strings.HasPrefix(subject, "-> ") {
			continue
		}
		isRemote := strings.HasPrefix(parts[1], "refs/remotes/")
		branches = append(branches, BranchInfo{
			Name:      parts[2],
			Hash:      parts[3],
			Subject:   subject,
			IsCurrent: strings.TrimSpace(parts[0]) == "*",
			IsRemote:  isRemote,
		})
	}
	return branches
}

func SwitchBranch(root, name string) error {
	_, stderr, err := runWithStderr(root, "switch", name)
	if err != nil {
		return &GitError{Msg: stderr}
	}
	return nil
}

func CreateBranch(root, name string) error {
	_, stderr, err := runWithStderr(root, "switch", "-c", name)
	if err != nil {
		return &GitError{Msg: stderr}
	}
	return nil
}

func DeleteBranch(root, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, stderr, err := runWithStderr(root, "branch", flag, name)
	if err != nil {
		return &GitError{Msg: stderr}
	}
	return nil
}

func RestoreFile(root, path string) error {
	_, stderr, err := runWithStderr(root, "restore", "--", path)
	if err != nil {
		return &GitError{Msg: stderr}
	}
	return nil
}

func PushArgs(branch string, force bool) []string {
	args := []string{"push", "--set-upstream", "origin", branch}
	if force {
		args = append(args, "--force-with-lease")
	}
	return args
}

func PullArgs() []string { return []string{"pull"} }
func FetchArgs() []string { return []string{"fetch", "--all", "--prune"} }
