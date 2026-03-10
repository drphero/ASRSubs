type AppShellProps = {
  selectedModel: string;
  hasSelection: boolean;
};

export function AppShell({ selectedModel, hasSelection }: AppShellProps) {
  return (
    <div className="app-shell">
      <div className="ambient ambient-left" aria-hidden="true" />
      <div className="ambient ambient-right" aria-hidden="true" />

      <header className="topbar">
        <div>
          <p className="eyebrow">ASRSubs</p>
          <h1>Local subtitles with zero ceremony.</h1>
        </div>
        <span className="model-chip" aria-label="selected model">
          {selectedModel}
        </span>
      </header>

      <main className="shell-main">
        {hasSelection ? (
          <section className="workspace-placeholder" aria-label="workspace view">
            <div className="workspace-card">
              <p className="section-label">Workspace</p>
              <h2>Selected media appears here.</h2>
              <p>
                Phase 1 establishes the shell. File metadata, settings, and diagnostics
                will plug into this view in later plans.
              </p>
            </div>
          </section>
        ) : (
          <section className="landing-panel" aria-label="landing view">
            <div className="dropzone" role="button" tabIndex={0}>
              <span className="dropzone-orbit" aria-hidden="true" />
              <div className="dropzone-body">
                <p className="section-label">Drop media</p>
                <h2>Drag a file straight into the center.</h2>
                <p className="dropzone-meta">Browse stays secondary. The canvas stays clean.</p>
                <div className="dropzone-actions">
                  <button className="primary-action" type="button">
                    Browse Media
                  </button>
                  <span className="secondary-note">Selected model: {selectedModel}</span>
                </div>
              </div>
            </div>
          </section>
        )}
      </main>
    </div>
  );
}
