package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"
)

var ErrManagedRuntimeUnavailable = errors.New("managed Python runtime source is unavailable")

const pipOutputTailLineCount = 12

type Status struct {
	State      string `json:"state"`
	RootDir    string `json:"rootDir"`
	PythonPath string `json:"pythonPath"`
	WorkerPath string `json:"workerPath"`
	Detail     string `json:"detail"`
}

type Option func(*serviceConfig)

type serviceConfig struct {
	rootDir              string
	requirementsPath     string
	workerScriptPath     string
	managedRuntimeSource string
}

type bootstrapState struct {
	Version          int    `json:"version"`
	InstalledAt      string `json:"installedAt"`
	RequirementsHash string `json:"requirementsHash"`
	Source           string `json:"source"`
}

type Service struct {
	rootDir              string
	requirementsPath     string
	workerScriptPath     string
	managedRuntimeSource string
}

func WithRootDir(path string) Option {
	return func(cfg *serviceConfig) {
		cfg.rootDir = path
	}
}

func WithRequirementsPath(path string) Option {
	return func(cfg *serviceConfig) {
		cfg.requirementsPath = path
	}
}

func WithWorkerScriptPath(path string) Option {
	return func(cfg *serviceConfig) {
		cfg.workerScriptPath = path
	}
}

func WithManagedRuntimeSource(path string) Option {
	return func(cfg *serviceConfig) {
		cfg.managedRuntimeSource = path
	}
}

func NewService(appName string, options ...Option) (*Service, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	rootDir := filepath.Join(configDir, appName, "runtime")
	return newService(rootDir, options...)
}

func NewServiceAtRoot(rootDir string, options ...Option) *Service {
	service, err := newService(rootDir, options...)
	if err != nil {
		panic(err)
	}

	return service
}

func newService(rootDir string, options ...Option) (*Service, error) {
	cfg := serviceConfig{
		rootDir: rootDir,
	}

	for _, option := range options {
		option(&cfg)
	}

	var err error
	if strings.TrimSpace(cfg.workerScriptPath) == "" {
		cfg.workerScriptPath, err = resolveRuntimeSupportPath("worker.py")
		if err != nil {
			return nil, err
		}
	}

	if strings.TrimSpace(cfg.requirementsPath) == "" {
		cfg.requirementsPath, err = resolveRuntimeSupportPath("requirements.txt")
		if err != nil {
			return nil, err
		}
	}

	return &Service{
		rootDir:              cfg.rootDir,
		requirementsPath:     cfg.requirementsPath,
		workerScriptPath:     cfg.workerScriptPath,
		managedRuntimeSource: cfg.managedRuntimeSource,
	}, nil
}

func (s *Service) EnsureReady(ctx context.Context) (Status, error) {
	if err := os.MkdirAll(s.rootDir, 0o755); err != nil {
		return s.failedStatus(fmt.Sprintf("Managed runtime root could not be created: %v", err)), err
	}

	requirementsHash, err := hashFile(s.requirementsPath)
	if err != nil {
		return s.failedStatus(fmt.Sprintf("Runtime dependencies could not be read: %v", err)), err
	}

	state, err := s.loadBootstrapState()
	if err != nil {
		return s.failedStatus(fmt.Sprintf("Managed runtime state could not be read: %v", err)), err
	}

	if state != nil && state.RequirementsHash == requirementsHash && fileExists(s.pythonPath()) {
		return s.readyStatus("Managed runtime is already prepared."), nil
	}

	source, err := s.resolveManagedRuntimeSource()
	if err != nil {
		return s.failedStatus(err.Error()), err
	}

	if err := os.RemoveAll(s.installDir()); err != nil {
		return s.failedStatus(fmt.Sprintf("Managed runtime could not be refreshed: %v", err)), err
	}

	if err := installManagedRuntime(source, s.installDir()); err != nil {
		return s.failedStatus(fmt.Sprintf("Managed runtime could not be installed: %v", err)), err
	}

	if !fileExists(s.pythonPath()) {
		err := fmt.Errorf("managed Python executable not found at %s", s.pythonPath())
		return s.failedStatus(err.Error()), err
	}

	if err := s.installRequirements(ctx); err != nil {
		return s.failedStatus(err.Error()), err
	}

	state = &bootstrapState{
		Version:          1,
		InstalledAt:      time.Now().UTC().Format(time.RFC3339),
		RequirementsHash: requirementsHash,
		Source:           source,
	}

	if err := s.writeBootstrapState(state); err != nil {
		return s.failedStatus(fmt.Sprintf("Managed runtime state could not be written: %v", err)), err
	}

	return s.readyStatus("Managed runtime prepared and dependencies installed."), nil
}

func (s *Service) RuntimeRoot() string {
	return s.rootDir
}

func (s *Service) PythonPath() string {
	return s.pythonPath()
}

func (s *Service) WorkerScriptPath() string {
	return s.workerScriptPath
}

func (s *Service) Status() Status {
	if fileExists(s.pythonPath()) {
		return s.readyStatus("Managed runtime executable is present.")
	}

	if !fileExists(s.workerScriptPath) {
		return s.failedStatus(fmt.Sprintf("Managed runtime worker script is missing at %s.", s.workerScriptPath))
	}

	if !fileExists(s.requirementsPath) {
		return s.failedStatus(fmt.Sprintf("Managed runtime requirements are missing at %s.", s.requirementsPath))
	}

	if _, err := s.resolveManagedRuntimeSource(); err != nil {
		return s.failedStatus(err.Error())
	}

	return Status{
		State:      "missing",
		RootDir:    s.rootDir,
		PythonPath: s.pythonPath(),
		WorkerPath: s.workerScriptPath,
		Detail:     "Managed runtime has not been prepared yet.",
	}
}

func (s *Service) installRequirements(ctx context.Context) error {
	cmd := exec.CommandContext(
		ctx,
		s.pythonPath(),
		"-m",
		"pip",
		"install",
		"--disable-pip-version-check",
		"-r",
		s.requirementsPath,
	)
	ConfigureSubprocess(cmd)
	cmd.Env = append(os.Environ(), "PYTHONUTF8=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputTail := summarizeCommandOutput(output)
		if ctxErr := ctx.Err(); ctxErr != nil {
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				message := "managed runtime dependency installation exceeded the 30 minute setup window; check internet connectivity and retry"
				if outputTail != "" {
					message = fmt.Sprintf("%s\nRecent pip output:\n%s", message, outputTail)
				}
				return errors.New(message)
			}
			if errors.Is(ctxErr, context.Canceled) {
				message := "managed runtime dependency installation was canceled"
				if outputTail != "" {
					message = fmt.Sprintf("%s\nRecent pip output:\n%s", message, outputTail)
				}
				return errors.New(message)
			}
		}

		message := outputTail
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("managed runtime dependencies could not be installed: %s", message)
	}

	return nil
}

func summarizeCommandOutput(output []byte) string {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return ""
	}

	lines := strings.Split(message, "\n")
	if len(lines) > pipOutputTailLineCount {
		lines = lines[len(lines)-pipOutputTailLineCount:]
	}

	for index, line := range lines {
		lines[index] = strings.TrimRight(line, "\r")
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (s *Service) resolveManagedRuntimeSource() (string, error) {
	if candidate := strings.TrimSpace(s.managedRuntimeSource); candidate != "" {
		return validateManagedRuntimeSource(candidate)
	}

	candidates := []string{
		os.Getenv("ASRSUBS_PYTHON_STANDALONE"),
		ResolveBundledResourcePath("runtime", "python"),
		filepath.Join("build", "runtime", "python"),
		filepath.Join("resources", "python"),
	}

	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}

		resolved, err := validateManagedRuntimeSource(candidate)
		if err == nil {
			return resolved, nil
		}
	}

	return "", fmt.Errorf(
		"%w: package a standalone runtime under runtime/python in the app resources, build/runtime/python in the repo, or set ASRSUBS_PYTHON_STANDALONE for local development",
		ErrManagedRuntimeUnavailable,
	)
}

func resolveRuntimeSupportPath(name string) (string, error) {
	if bundled := ResolveBundledResourcePath("runtime", name); bundled != "" {
		return bundled, nil
	}

	return filepath.Abs(filepath.Join("internal", "runtime", name))
}

func ResolveBundledResourcePath(parts ...string) string {
	for _, root := range bundledResourceRoots() {
		if strings.TrimSpace(root) == "" {
			continue
		}

		candidate := filepath.Join(append([]string{root}, parts...)...)
		if pathExists(candidate) {
			return candidate
		}
	}

	return ""
}

func bundledResourceRoots() []string {
	roots := []string{}
	added := map[string]struct{}{}

	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}

		resolved, err := filepath.Abs(path)
		if err != nil {
			return
		}
		if _, exists := added[resolved]; exists {
			return
		}

		added[resolved] = struct{}{}
		roots = append(roots, resolved)
	}

	add(os.Getenv("ASRSUBS_RESOURCE_ROOT"))

	executablePath, err := os.Executable()
	if err == nil {
		executableDir := filepath.Dir(executablePath)
		add(executableDir)
		add(filepath.Join(executableDir, "resources"))

		if strings.EqualFold(filepath.Base(executableDir), "MacOS") {
			add(filepath.Join(executableDir, "..", "Resources"))
		}
	}

	return roots
}

func (s *Service) loadBootstrapState() (*bootstrapState, error) {
	data, err := os.ReadFile(s.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var state bootstrapState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (s *Service) writeBootstrapState(state *bootstrapState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.statePath(), data, 0o644)
}

func (s *Service) readyStatus(detail string) Status {
	return Status{
		State:      "ready",
		RootDir:    s.rootDir,
		PythonPath: s.pythonPath(),
		WorkerPath: s.workerScriptPath,
		Detail:     detail,
	}
}

func (s *Service) failedStatus(detail string) Status {
	return Status{
		State:      "failed",
		RootDir:    s.rootDir,
		PythonPath: s.pythonPath(),
		WorkerPath: s.workerScriptPath,
		Detail:     detail,
	}
}

func (s *Service) installDir() string {
	return filepath.Join(s.rootDir, "python")
}

func (s *Service) statePath() string {
	return filepath.Join(s.rootDir, "bootstrap.json")
}

func (s *Service) pythonPath() string {
	if goruntime.GOOS == "windows" {
		return filepath.Join(s.installDir(), "python.exe")
	}

	return filepath.Join(s.installDir(), "bin", "python3")
}

func installManagedRuntime(source string, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return copyTree(source, destination)
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	target := destination
	if goruntime.GOOS != "windows" {
		target = filepath.Join(destination, "bin", "python3")
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
	}

	return copyFile(source, target, info.Mode())
}

func validateManagedRuntimeSource(path string) (string, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		return resolved, nil
	}

	pythonCandidate := filepath.Join(resolved, "bin", "python3")
	if goruntime.GOOS == "windows" {
		pythonCandidate = filepath.Join(resolved, "python.exe")
	}

	if !fileExists(pythonCandidate) {
		return "", fmt.Errorf("managed runtime source missing Python executable at %s", pythonCandidate)
	}

	return resolved, nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyTree(source string, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destination, relativePath)
		info, err := entry.Info()
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		if entry.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			return os.Symlink(linkTarget, targetPath)
		}

		return copyFile(path, targetPath, info.Mode())
	})
}

func copyFile(source string, destination string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return dst.Close()
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}
