import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

// ErrorBoundary catches render-time exceptions anywhere below it and shows a
// legible, on-brand recovery screen instead of a blank white page — so a single
// thrown error in one component doesn't take the whole panel down silently.
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Surface it in the console for debugging; the UI shows a friendly message.
    console.error("Uncaught render error:", error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="sp-error-boundary">
          <div className="sp-surface sp-card" style={{ maxWidth: 520, margin: "12vh auto", textAlign: "center" }}>
            <h1 className="sp-page-title" style={{ marginBottom: 8 }}>
              Something broke
            </h1>
            <p className="sp-mono" style={{ fontSize: 13, color: "var(--sp-text-muted)", marginBottom: 18 }}>
              An unexpected error crashed this view. Reloading usually fixes it.
            </p>
            <p className="sp-mono" style={{ fontSize: 12, color: "#ff9b9b", marginBottom: 18, wordBreak: "break-word" }}>
              {this.state.error.message}
            </p>
            <button className="sp-btn sp-btn--primary" onClick={() => window.location.reload()}>
              Reload
            </button>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
