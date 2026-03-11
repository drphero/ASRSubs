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
	Internal           bool   `json:"-"`
}

const ForcedAlignerID = "Qwen3-ForcedAligner-0.6B"

var selectableCatalog = []ModelDescriptor{
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

var internalCatalog = []ModelDescriptor{
	{
		ID:                 ForcedAlignerID,
		Name:               ForcedAlignerID,
		RepoID:             "Qwen/Qwen3-ForcedAligner-0.6B",
		Description:        "Internal forced aligner used automatically for timestamp generation.",
		SpeedDescription:   "Managed automatically",
		QualityDescription: "Word-level timestamps",
		SystemRequirement:  "Hidden internal dependency",
		Default:            false,
		Internal:           true,
	},
}

func Catalog() []ModelDescriptor {
	cloned := make([]ModelDescriptor, len(selectableCatalog))
	copy(cloned, selectableCatalog)
	return cloned
}

func Lookup(id string) (ModelDescriptor, bool) {
	for _, model := range allCatalog() {
		if model.ID == id {
			return model, true
		}
	}

	return ModelDescriptor{}, false
}

func allCatalog() []ModelDescriptor {
	combined := make([]ModelDescriptor, 0, len(selectableCatalog)+len(internalCatalog))
	combined = append(combined, selectableCatalog...)
	combined = append(combined, internalCatalog...)
	return combined
}
