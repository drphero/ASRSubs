package main

import (
	"fmt"

	"ASRSubs/internal/models"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) LoadModelSnapshot() (models.Snapshot, error) {
	service, err := a.requireModelService()
	if err != nil {
		return models.Snapshot{}, err
	}

	return service.Snapshot(), nil
}

func (a *App) GetModelState(modelID string) (models.ModelStatus, error) {
	service, err := a.requireModelService()
	if err != nil {
		return models.ModelStatus{}, err
	}

	return service.GetModelState(modelID)
}

func (a *App) StartModelDownload(modelID string) (models.ModelStatus, error) {
	service, err := a.requireModelService()
	if err != nil {
		return models.ModelStatus{}, err
	}

	status, err := service.StartDownload(modelID)
	if err != nil {
		a.recordDiagnostic("error", "models", err.Error())
		return models.ModelStatus{}, err
	}

	a.recordDiagnostic("info", "models", "Model download started for "+modelID+".")
	return status, nil
}

func (a *App) DeleteModel(modelID string) (models.ModelStatus, error) {
	service, err := a.requireModelService()
	if err != nil {
		return models.ModelStatus{}, err
	}

	status, err := service.Delete(modelID)
	if err != nil {
		a.recordDiagnostic("error", "models", err.Error())
		return models.ModelStatus{}, err
	}

	a.recordDiagnostic("info", "models", "Deleted local files for "+modelID+".")
	return status, nil
}

func (a *App) requireModelService() (*models.Service, error) {
	if a.models == nil {
		return nil, fmt.Errorf("model service is not ready")
	}

	return a.models, nil
}

func (a *App) emitModelSnapshot(snapshot models.Snapshot) {
	if a.ctx == nil {
		return
	}

	wailsruntime.EventsEmit(a.ctx, "models:state", snapshot)
}
