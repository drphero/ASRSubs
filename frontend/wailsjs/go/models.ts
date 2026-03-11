export namespace intake {
	
	export class MediaMetadata {
	    path: string;
	    name: string;
	    extension: string;
	    directory: string;
	    sizeBytes: number;
	    durationSeconds: number;
	    durationLabel: string;
	    hasKnownDuration: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MediaMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.extension = source["extension"];
	        this.directory = source["directory"];
	        this.sizeBytes = source["sizeBytes"];
	        this.durationSeconds = source["durationSeconds"];
	        this.durationLabel = source["durationLabel"];
	        this.hasKnownDuration = source["hasKnownDuration"];
	    }
	}

}

export namespace main {
	
	export class DiagnosticsEntry {
	    id: string;
	    level: string;
	    source: string;
	    message: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new DiagnosticsEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.level = source["level"];
	        this.source = source["source"];
	        this.message = source["message"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class DiagnosticsSummary {
	    title: string;
	    message: string;
	    level: string;
	
	    static createFrom(source: any = {}) {
	        return new DiagnosticsSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.message = source["message"];
	        this.level = source["level"];
	    }
	}
	export class DiagnosticsSnapshot {
	    summary: DiagnosticsSummary;
	    entries: DiagnosticsEntry[];
	
	    static createFrom(source: any = {}) {
	        return new DiagnosticsSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = this.convertValues(source["summary"], DiagnosticsSummary);
	        this.entries = this.convertValues(source["entries"], DiagnosticsEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class RuntimeReadiness {
	    state: string;
	    rootDir: string;
	    pythonPath: string;
	    workerPath: string;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeReadiness(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.rootDir = source["rootDir"];
	        this.pythonPath = source["pythonPath"];
	        this.workerPath = source["workerPath"];
	        this.detail = source["detail"];
	    }
	}
	export class transcriptionRequest {
	    mediaPath: string;
	    modelID: string;
	
	    static createFrom(source: any = {}) {
	        return new transcriptionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mediaPath = source["mediaPath"];
	        this.modelID = source["modelID"];
	    }
	}

}

export namespace models {
	
	export class ModelStatus {
	    id: string;
	    name: string;
	    repoId: string;
	    description: string;
	    speedDescription: string;
	    qualityDescription: string;
	    systemRequirement: string;
	    default: boolean;
	    state: string;
	    stateLabel: string;
	    path: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.repoId = source["repoId"];
	        this.description = source["description"];
	        this.speedDescription = source["speedDescription"];
	        this.qualityDescription = source["qualityDescription"];
	        this.systemRequirement = source["systemRequirement"];
	        this.default = source["default"];
	        this.state = source["state"];
	        this.stateLabel = source["stateLabel"];
	        this.path = source["path"];
	        this.error = source["error"];
	    }
	}
	export class Snapshot {
	    version: number;
	    models: ModelStatus[];
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.models = this.convertValues(source["models"], ModelStatus);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace settings {
	
	export class DirectoryPreferences {
	    lastOpenDirectory: string;
	    lastSaveDirectory: string;
	
	    static createFrom(source: any = {}) {
	        return new DirectoryPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lastOpenDirectory = source["lastOpenDirectory"];
	        this.lastSaveDirectory = source["lastSaveDirectory"];
	    }
	}
	export class OutputPreferences {
	    maxLineLength: number;
	    linesPerSubtitle: number;
	
	    static createFrom(source: any = {}) {
	        return new OutputPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxLineLength = source["maxLineLength"];
	        this.linesPerSubtitle = source["linesPerSubtitle"];
	    }
	}
	export class ProcessingPreferences {
	    alignmentChunkMinutes: number;
	    oneWordPerSubtitle: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProcessingPreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.alignmentChunkMinutes = source["alignmentChunkMinutes"];
	        this.oneWordPerSubtitle = source["oneWordPerSubtitle"];
	    }
	}
	export class Preferences {
	    version: number;
	    model: string;
	    theme: string;
	    output: OutputPreferences;
	    directories: DirectoryPreferences;
	    processing: ProcessingPreferences;
	
	    static createFrom(source: any = {}) {
	        return new Preferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.model = source["model"];
	        this.theme = source["theme"];
	        this.output = this.convertValues(source["output"], OutputPreferences);
	        this.directories = this.convertValues(source["directories"], DirectoryPreferences);
	        this.processing = this.convertValues(source["processing"], ProcessingPreferences);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace transcription {
	
	export class Snapshot {
	    active: boolean;
	    canRetry: boolean;
	    stage: string;
	    failedStage: string;
	    partIndex: number;
	    partCount: number;
	    filePath: string;
	    fileName: string;
	    modelID: string;
	    failureSummary: string;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.canRetry = source["canRetry"];
	        this.stage = source["stage"];
	        this.failedStage = source["failedStage"];
	        this.partIndex = source["partIndex"];
	        this.partCount = source["partCount"];
	        this.filePath = source["filePath"];
	        this.fileName = source["fileName"];
	        this.modelID = source["modelID"];
	        this.failureSummary = source["failureSummary"];
	    }
	}

}

