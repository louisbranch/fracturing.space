import { ClipboardList, Drama, MessagesSquare } from "lucide-react";
import type { HUDNavbarTab } from "../shared/contract";
import type { HUDNavbarProps } from "./contract";

const tabs: { id: HUDNavbarTab; label: string; icon: typeof Drama; tooltip: string }[] = [
  { id: "on-stage", label: "On Stage", icon: Drama, tooltip: "Play as your character." },
  { id: "backstage", label: "Backstage", icon: ClipboardList, tooltip: "Resolve rules and issues." },
  { id: "side-chat", label: "Side Chat", icon: MessagesSquare, tooltip: "Out-of-character chat." },
];

export function HUDNavbar({ activeTab, onTabChange, tabsWithUpdates }: HUDNavbarProps) {
  return (
    <nav aria-label="Player HUD navigation" className="navbar bg-base-100 shadow-sm">
      <div className="navbar-start" />
      <div className="navbar-center gap-2">
        {tabs.map(({ id, label, icon: Icon, tooltip }) => {
          const active = activeTab === id;
          const updateCount = !active ? tabsWithUpdates?.get(id) : undefined;
          return (
            <div key={id} className={`tooltip tooltip-bottom ${updateCount ? "indicator" : ""}`} data-tip={tooltip}>
              {updateCount && (
                <span className="indicator-item indicator-center badge badge-primary badge-xs">
                  {updateCount}
                </span>
              )}
              <button
                type="button"
                className={`btn ${active ? "btn-primary btn-soft" : "btn-ghost"}`}
                aria-current={active ? "page" : undefined}
                onClick={() => onTabChange(id)}
              >
                <Icon size={20} />
                <span className="text-xs">{label}</span>
              </button>
            </div>
          );
        })}
      </div>
      <div className="navbar-end" />
    </nav>
  );
}
