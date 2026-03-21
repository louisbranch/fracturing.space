import { ClipboardList, Drama, Menu, MessagesSquare } from "lucide-react";
import type { HUDNavbarTab } from "../contract";
import type { HUDNavbarProps } from "./contract";

const tabs: { id: HUDNavbarTab; label: string; icon: typeof Drama; tooltip: string }[] = [
  { id: "on-stage", label: "On Stage", icon: Drama, tooltip: "Play as your character." },
  {
    id: "backstage",
    label: "Backstage",
    icon: ClipboardList,
    tooltip: "Authoritative OOC pauses, rulings, and coordination.",
  },
  { id: "side-chat", label: "Side Chat", icon: MessagesSquare, tooltip: "Optional non-authoritative session chat." },
];

export function HUDNavbar({
  activeTab,
  isSidebarOpen,
  onSidebarOpenChange,
  onTabChange,
  tabsWithUpdates,
}: HUDNavbarProps) {
  return (
    <nav aria-label="Player HUD navigation" className="navbar min-h-0 bg-base-100 px-2 py-1 shadow-sm">
      <div className="navbar-start">
        <button
          type="button"
          aria-label={isSidebarOpen ? "Close campaign sidebar" : "Open campaign sidebar"}
          className="btn btn-ghost btn-square btn-sm"
          onClick={() => onSidebarOpenChange(!isSidebarOpen)}
        >
          <Menu size={18} aria-hidden="true" />
        </button>
      </div>
      <div className="navbar-center gap-1.5">
        {tabs.map(({ id, label, icon: Icon, tooltip }) => {
          const active = activeTab === id;
          const updateCount = !active ? tabsWithUpdates?.get(id) : undefined;
          return (
            <div key={id} className={`tooltip tooltip-bottom ${updateCount ? "indicator" : ""}`} data-tip={tooltip}>
              {updateCount ? (
                <span className="indicator-item indicator-center badge badge-primary badge-xs">
                  {updateCount}
                </span>
              ) : null}
              <button
                type="button"
                aria-label={label}
                className={`btn btn-sm gap-1.5 ${active ? "btn-primary btn-soft" : "btn-ghost"}`}
                aria-current={active ? "page" : undefined}
                onClick={() => onTabChange(id)}
              >
                <Icon size={18} aria-hidden="true" />
                <span className="text-[0.68rem]">{label}</span>
              </button>
            </div>
          );
        })}
      </div>
      <div className="navbar-end" />
    </nav>
  );
}
