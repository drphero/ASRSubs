package models

type ModelDescriptor struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	RepoID             string `json:"repoId"`
	Description        string `json:"description"`
	SpeedDescription   string `json:"speedDescription"`
	QualityDescription string `json:"qualityDescription"`
	SystemRequirement  string `json:"systemRequirement"`
	Default            bool   `json:"default"`
}

var catalog = []ModelDescriptor{
	{
		ID:                 "Qwen3-ASR-1.7B",
		Name:               "Qwen3-ASR-1.7B",
		RepoID:             "Qwen/Qwen3-ASR-1.7B",
		Description:        "Highest accuracy for mixed speech at the cost of a heavier local runtime footprint.",
		SpeedDescription:   "Slower, best quality",
		QualityDescription: "Best accuracy on harder audio",
		SystemRequirement:  "More RAM and longer first download",
		Default:            true,
	},
	{
		ID:                 "Qwen3-ASR-0.6B",
		Name:               "Qwen3-ASR-0.6B",
		RepoID:             "Qwen/Qwen3-ASR-0.6B",
		Description:        "Faster startup with a lighter model footprint for smaller machines and quick checks.",
		SpeedDescription:   "Faster, lighter runtime",
		QualityDescription: "Good quality for quicker runs",
		SystemRequirement:  "Lower memory use",
		Default:            false,
	},
}

func Catalog() []ModelDescriptor {
	cloned := make([]ModelDescriptor, len(catalog))
	copy(cloned, catalog)
	return cloned
}

func Lookup(id string) (ModelDescriptor, bool) {
	for _, model := range catalog {
		if model.ID == id {
			return model, true
		}
	}

	return ModelDescriptor{}, false
}
