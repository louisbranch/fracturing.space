import { useEffect, useId, useRef } from "react";
import { X } from "lucide-react";
import { useTransitionPreferences } from "./TransitionPreferencesContext";

type TransitionSettingsModalProps = {
  isOpen: boolean;
  onClose: () => void;
};

const toggleRows = [
  { key: "sceneVisual", label: "Scene transition effect" },
  { key: "sceneSound", label: "Scene transition sound" },
  { key: "interactionVisual", label: "Interaction transition effect" },
  { key: "interactionSound", label: "Interaction transition sound" },
] as const;

export function TransitionSettingsModal({ isOpen, onClose }: TransitionSettingsModalProps) {
  const dialogRef = useRef<HTMLDialogElement | null>(null);
  const titleID = useId();
  const { preferences, setPreference } = useTransitionPreferences();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (isOpen && !dialog.open) {
      if (typeof dialog.showModal === "function") {
        dialog.showModal();
      } else {
        dialog.setAttribute("open", "");
      }
    }

    if (!isOpen && dialog.open) {
      if (typeof dialog.close === "function") {
        dialog.close();
      } else {
        dialog.removeAttribute("open");
      }
    }
  }, [isOpen]);

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleID}
      className="modal"
      onClose={onClose}
    >
      <div className="modal-box max-w-md p-0">
        <header className="flex items-center justify-between border-b border-base-300/70 px-5 py-4">
          <h2 id={titleID} className="text-lg font-semibold text-base-content">
            Play Settings
          </h2>
          <button
            type="button"
            aria-label="Close settings"
            className="btn btn-ghost btn-sm btn-square"
            onClick={onClose}
          >
            <X size={18} aria-hidden="true" />
          </button>
        </header>

        <div className="flex flex-col gap-1 px-5 py-4">
          {toggleRows.map(({ key, label }) => (
            <label key={key} className="flex cursor-pointer items-center justify-between gap-4 rounded-lg px-1 py-2">
              <span className="text-sm text-base-content">{label}</span>
              <input
                type="checkbox"
                className="toggle toggle-primary"
                checked={preferences[key]}
                onChange={(e) => setPreference(key, e.currentTarget.checked)}
              />
            </label>
          ))}
        </div>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button type="submit" onClick={onClose}>
          close
        </button>
      </form>
    </dialog>
  );
}
