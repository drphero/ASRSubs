import { useEffect, useState } from "react";

export const OVERLAY_ENTER_DELAY_MS = 16;
export const OVERLAY_EXIT_DURATION_MS = 180;

export type OverlayPresenceState = "entering" | "open" | "closing";

export function useOverlayPresence(open: boolean) {
  const [present, setPresent] = useState(open);
  const [state, setState] = useState<OverlayPresenceState>(open ? "open" : "closing");

  useEffect(() => {
    let timer = 0;

    if (open) {
      setPresent(true);
      setState("entering");
      timer = window.setTimeout(() => {
        setState("open");
      }, OVERLAY_ENTER_DELAY_MS);

      return () => {
        window.clearTimeout(timer);
      };
    }

    if (!present) {
      return;
    }

    setState("closing");
    timer = window.setTimeout(() => {
      setPresent(false);
    }, OVERLAY_EXIT_DURATION_MS);

    return () => {
      window.clearTimeout(timer);
    };
  }, [open, present]);

  return { present, state };
}
