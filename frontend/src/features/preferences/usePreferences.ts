import { useEffect, useRef, useState } from "react";
import {
  defaultPreferences,
  loadPreferences,
  updatePreferences,
  type Preferences,
} from "../../lib/backend";

export function usePreferences() {
  const [preferences, setPreferences] = useState<Preferences>(defaultPreferences);
  const [error, setError] = useState<string | null>(null);
  const latestPreferences = useRef(preferences);

  useEffect(() => {
    latestPreferences.current = preferences;
  }, [preferences]);

  useEffect(() => {
    let active = true;

    void loadPreferences()
      .then((loaded) => {
        if (!active) {
          return;
        }

        setPreferences(loaded);
        setError(null);
      })
      .catch(() => {
        if (!active) {
          return;
        }

        setError("Saved settings could not be loaded.");
      });

    return () => {
      active = false;
    };
  }, []);

  async function replacePreferences(next: Preferences) {
    setPreferences(next);

    try {
      const saved = await updatePreferences(next);
      setPreferences(saved);
      setError(null);
      return saved;
    } catch {
      setError("Settings could not be saved.");
      return next;
    }
  }

  async function patchPreferences(updater: (current: Preferences) => Preferences) {
    const next = updater(latestPreferences.current);
    return replacePreferences(next);
  }

  return {
    error,
    patchPreferences,
    preferences,
    replacePreferences,
  };
}
