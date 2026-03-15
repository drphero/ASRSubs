package main

import (
	"context"
	"fmt"
	"time"

	asrruntime "ASRSubs/internal/runtime"
)

type RuntimeReadiness struct {
	State      string `json:"state"`
	RootDir    string `json:"rootDir"`
	PythonPath string `json:"pythonPath"`
	WorkerPath string `json:"workerPath"`
	Detail     string `json:"detail"`
}

func (a *App) EnsureRuntimeReady() (RuntimeReadiness, error) {
	service, err := a.requireRuntimeService()
	if err != nil {
		return RuntimeReadiness{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	status, err := service.EnsureReady(ctx)
	if err != nil {
		a.recordDiagnostic("error", "runtime", status.Detail)
		return runtimeReadinessFromStatus(status), err
	}

	_, smokeErr := service.Smoke(ctx)
	if smokeErr != nil {
		detail := smokeErr.Error()
		a.recordDiagnostic("error", "runtime", detail)
		status.Detail = detail
		status.State = "failed"
		return runtimeReadinessFromStatus(status), smokeErr
	}

	a.recordDiagnostic("info", "runtime", "Managed runtime is ready.")
	return runtimeReadinessFromStatus(status), nil
}

func (a *App) GetRuntimeReadiness() (RuntimeReadiness, error) {
	service, err := a.requireRuntimeService()
	if err != nil {
		return RuntimeReadiness{}, err
	}

	return runtimeReadinessFromStatus(service.Status()), nil
}

func (a *App) requireRuntimeService() (*asrruntime.Service, error) {
	if a.runtime == nil {
		return nil, fmt.Errorf("runtime service is not ready")
	}

	return a.runtime, nil
}

func runtimeReadinessFromStatus(status asrruntime.Status) RuntimeReadiness {
	return RuntimeReadiness{
		State:      status.State,
		RootDir:    status.RootDir,
		PythonPath: status.PythonPath,
		WorkerPath: status.WorkerPath,
		Detail:     status.Detail,
	}
}
