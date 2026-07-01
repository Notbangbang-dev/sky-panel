import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("renders the status text", () => {
    render(<StatusBadge status="running" />);
    expect(screen.getByText("running")).toBeInTheDocument();
  });

  it("applies the running variant class for a running server", () => {
    render(<StatusBadge status="running" />);
    expect(screen.getByText("running").className).toContain("sp-badge--running");
  });

  it("applies the errored variant class for an errored server", () => {
    render(<StatusBadge status="errored" />);
    expect(screen.getByText("errored").className).toContain("sp-badge--errored");
  });
});
