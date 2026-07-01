import { describe, expect, it, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ThemeProvider, useTheme } from "./ThemeProvider";
import { PRESET_THEMES } from "./theme";

function Probe() {
  const { theme, themes, setThemeId, saveCustomTheme } = useTheme();
  return (
    <div>
      <span data-testid="active-theme">{theme.id}</span>
      <span data-testid="theme-count">{themes.length}</span>
      <button onClick={() => setThemeId("signal")}>use-signal</button>
      <button onClick={() => saveCustomTheme({ ...PRESET_THEMES[0], id: "my-custom", name: "Mine", builtin: false })}>
        save-custom
      </button>
    </div>
  );
}

describe("ThemeProvider", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("defaults to the monochrome preset", () => {
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );
    expect(screen.getByTestId("active-theme").textContent).toBe("monochrome");
  });

  it("switches the active theme and persists the choice", () => {
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );

    fireEvent.click(screen.getByText("use-signal"));

    expect(screen.getByTestId("active-theme").textContent).toBe("signal");
    expect(localStorage.getItem("sky-panel:active-theme")).toBe("signal");
  });

  it("saving a custom theme adds it to the list and activates it", () => {
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>,
    );

    const initialCount = Number(screen.getByTestId("theme-count").textContent);
    fireEvent.click(screen.getByText("save-custom"));

    expect(screen.getByTestId("active-theme").textContent).toBe("my-custom");
    expect(Number(screen.getByTestId("theme-count").textContent)).toBe(initialCount + 1);
  });
});
