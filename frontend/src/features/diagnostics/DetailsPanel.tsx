import type { DiagnosticsSnapshot } from "../../lib/backend";

type DetailsPanelProps = {
  onClose: () => void;
  open: boolean;
  snapshot: DiagnosticsSnapshot;
};

export function DetailsPanel({ onClose, open, snapshot }: DetailsPanelProps) {
  if (!open) {
    return null;
  }

  return (
    <div className="overlay-shell" role="presentation">
      <button aria-label="Close details" className="overlay-backdrop" onClick={onClose} type="button" />
      <aside aria-label="details panel" className="details-panel" role="dialog">
        <div className="drawer-header">
          <div>
            <p className="section-label">Diagnostics</p>
            <h2>{snapshot.summary.title}</h2>
          </div>
          <button className="ghost-action" onClick={onClose} type="button">
            Close
          </button>
        </div>

        <div className={`status-summary status-${snapshot.summary.level}`}>
          <p>{snapshot.summary.message}</p>
        </div>

        {snapshot.entries.length === 0 ? (
          <p className="inline-feedback">Logs appear here during app activity.</p>
        ) : (
          <ul className="details-list">
            {snapshot.entries
              .slice()
              .reverse()
              .map((entry) => (
                <li className="details-item" key={entry.id}>
                  <div>
                    <p className="details-item-source">{entry.source}</p>
                    <strong>{entry.message}</strong>
                  </div>
                  <span>{entry.timestamp}</span>
                </li>
              ))}
          </ul>
        )}
      </aside>
    </div>
  );
}
