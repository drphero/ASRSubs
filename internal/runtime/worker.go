package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type WorkerRequest struct {
	Command   string `json:"command"`
	AudioPath string `json:"audioPath,omitempty"`
	ModelPath string `json:"modelPath,omitempty"`
	Language  string `json:"language,omitempty"`
}

type WorkerResponse struct {
	OK      bool            `json:"ok"`
	Command string          `json:"command"`
	Message string          `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
	Details json.RawMessage `json:"details,omitempty"`
}

type WorkerError struct {
	Command  string
	ExitCode int
	Stderr   string
	Message  string
	Cause    error
}

func (e *WorkerError) Error() string {
	if e == nil {
		return ""
	}

	return e.Message
}

func (e *WorkerError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

func (s *Service) RunWorker(ctx context.Context, request WorkerRequest) (WorkerResponse, error) {
	if strings.TrimSpace(request.Command) == "" {
		return WorkerResponse{}, &WorkerError{
			Message: "worker command is required",
		}
	}

	if !fileExists(s.pythonPath()) {
		return WorkerResponse{}, &WorkerError{
			Command: request.Command,
			Message: "managed runtime is not ready",
		}
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return WorkerResponse{}, &WorkerError{
			Command: request.Command,
			Message: fmt.Sprintf("worker request could not be encoded: %v", err),
			Cause:   err,
		}
	}

	cmd := exec.CommandContext(ctx, s.pythonPath(), s.workerScriptPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return WorkerResponse{}, &WorkerError{
			Command: request.Command,
			Message: fmt.Sprintf("worker stdout could not be captured: %v", err),
			Cause:   err,
		}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return WorkerResponse{}, &WorkerError{
			Command: request.Command,
			Message: fmt.Sprintf("worker stderr could not be captured: %v", err),
			Cause:   err,
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return WorkerResponse{}, &WorkerError{
			Command: request.Command,
			Message: fmt.Sprintf("worker stdin could not be opened: %v", err),
			Cause:   err,
		}
	}

	if err := cmd.Start(); err != nil {
		return WorkerResponse{}, &WorkerError{
			Command: request.Command,
			Message: fmt.Sprintf("worker could not be started: %v", err),
			Cause:   err,
		}
	}

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	stdoutDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&stdoutBuf, stdout)
		stdoutDone <- copyErr
	}()

	stderrDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&stderrBuf, stderr)
		stderrDone <- copyErr
	}()

	writeErr := make(chan error, 1)
	go func() {
		defer stdin.Close()
		_, copyErr := stdin.Write(payload)
		writeErr <- copyErr
	}()

	waitErr := cmd.Wait()
	stdoutErr := <-stdoutDone
	stderrErr := <-stderrDone
	stdinErr := <-writeErr

	for _, pipeErr := range []error{stdoutErr, stderrErr, stdinErr} {
		if waitErr != nil && isIgnorablePipeError(pipeErr) {
			continue
		}
		if pipeErr != nil {
			return WorkerResponse{}, &WorkerError{
				Command: request.Command,
				Message: fmt.Sprintf("worker I/O failed: %v", pipeErr),
				Cause:   pipeErr,
				Stderr:  strings.TrimSpace(stderrBuf.String()),
			}
		}
	}

	if ctx.Err() != nil {
		return WorkerResponse{}, &WorkerError{
			Command:  request.Command,
			ExitCode: exitCode(waitErr),
			Stderr:   strings.TrimSpace(stderrBuf.String()),
			Message:  "worker canceled",
			Cause:    ctx.Err(),
		}
	}

	response, responseErr := decodeWorkerResponse(stdoutBuf.Bytes())
	if responseErr != nil {
		return WorkerResponse{}, &WorkerError{
			Command:  request.Command,
			Message:  fmt.Sprintf("worker response could not be decoded: %v", responseErr),
			Cause:    responseErr,
			Stderr:   strings.TrimSpace(stderrBuf.String()),
			ExitCode: exitCode(waitErr),
		}
	}

	if waitErr != nil {
		message := response.Error
		if message == "" {
			message = strings.TrimSpace(stderrBuf.String())
		}
		if message == "" {
			message = waitErr.Error()
		}
		if ctx.Err() != nil {
			message = "worker canceled"
		}

		return response, &WorkerError{
			Command:  request.Command,
			ExitCode: exitCode(waitErr),
			Stderr:   strings.TrimSpace(stderrBuf.String()),
			Message:  message,
			Cause:    waitErr,
		}
	}

	if !response.OK {
		message := response.Error
		if message == "" {
			message = "worker reported failure"
		}
		return response, &WorkerError{
			Command: request.Command,
			Stderr:  strings.TrimSpace(stderrBuf.String()),
			Message: message,
		}
	}

	return response, nil
}

func (s *Service) Smoke(ctx context.Context) (WorkerResponse, error) {
	return s.RunWorker(ctx, WorkerRequest{Command: "smoke"})
}

func decodeWorkerResponse(data []byte) (WorkerResponse, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return WorkerResponse{}, fmt.Errorf("worker stdout was empty")
	}

	var response WorkerResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return WorkerResponse{}, err
	}

	return response, nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return -1
}

func isIgnorablePipeError(err error) bool {
	if err == nil {
		return false
	}

	message := err.Error()
	return strings.Contains(message, "file already closed") || strings.Contains(message, "closed pipe")
}
