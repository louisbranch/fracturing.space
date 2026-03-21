import { Component } from "react";
import type { ErrorInfo, ReactNode } from "react";

type Props = {
  panelName: string;
  children: ReactNode;
};

type State = {
  hasError: boolean;
};

/** Catches render errors in a single HUD panel so other panels remain usable. */
export class PanelErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    if (import.meta.env.DEV) {
      console.error(`[${this.props.panelName}] render error`, error, info.componentStack);
    }
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-1 flex-col items-center justify-center gap-2 p-4 text-center">
          <p className="text-sm font-semibold">Something went wrong in {this.props.panelName}</p>
          <button
            className="btn btn-ghost btn-sm"
            onClick={() => this.setState({ hasError: false })}
          >
            Try again
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
