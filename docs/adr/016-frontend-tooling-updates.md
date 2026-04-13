# ADR-016: Frontend Tooling Updates (React 19, Tailwind 4, Recharts, CodeMirror 6)

## Status

Accepted

## Context

The original blueprint specified React 18, Tailwind CSS 3, visx for charts, Monaco Editor for code editing, and Framer Motion for animations. Since the blueprint was written, significant updates have landed. For a new project starting in 2026, we should use current stable versions and make pragmatic library choices that minimize bundle size and time-to-ship for a B2B dashboard.

## Decision

Update to current stable versions and swap libraries where better alternatives exist:

| Change | From | To | Why |
|--------|------|----|-----|
| React | 18 | **19.2.4** | Stable since Dec 2024. `use()`, `useActionState`, `useOptimistic` useful for SPA. |
| Tailwind CSS | 3.x | **4.2.1** | Rust-based Oxide engine (5x faster). CSS-first config. No `tailwind.config.js`. |
| Charts | visx (Airbnb) | **Recharts 3.7.0** (+@nivo/heatmap) | React-native API, faster to ship. visx requires D3 knowledge. Nivo only for compliance heatmaps. |
| Code Editor | Monaco Editor (~5MB) | **CodeMirror 6** (~300KB) | Users write bash/PowerShell scripts, not TypeScript. Don't need VS Code-level IntelliSense. |
| Animation | Framer Motion (~34KB) | **Tailwind CSS transitions** (0KB) | B2B dashboard doesn't need spring physics. CSS transitions cover 90% of needs. Add `motion` later if needed. |
| Terminal | xterm (deprecated) | **@xterm/xterm 6.0.0** | Package renamed to scoped `@xterm/*`. Old package unmaintained. |
| Timeline | react-chrono | **Recharts custom** or SVAR React Gantt | react-chrono is display-only. Recharts bar chart gives more control for deployment timelines. |
| Workflow builder | reactflow | **@xyflow/react 12.10.1** | Library renamed. Named exports, stricter TypeScript, React 19 + TW4 support. |
| Forms | react-hook-form + zod 3 | **RHF 7.71.2 + Zod 4.3.6** | Zod 4 is faster, smaller, better errors. Standard Schema support. |

## Consequences

- **Positive**: Smaller bundle sizes (CodeMirror 6 saves ~4.7MB, dropping Framer Motion saves ~34KB); current stable versions avoid future migrations; Recharts is significantly faster to ship than visx for dashboard charts
- **Negative**: React 19 Server Components don't apply to SPA (no loss); CodeMirror 6 requires more configuration than Monaco for advanced features; Tailwind 4 CSS-first config is a new paradigm (team learning curve)

## Alternatives Considered

- **Stay on React 18**: Works fine — rejected because starting a new project on an older major version creates unnecessary migration debt
- **Keep visx**: Maximum chart customization — rejected because the D3 learning curve slows dashboard development; Recharts covers 95% of our chart needs
- **Keep Monaco**: Full VS Code experience — rejected because 5MB bundle for editing bash scripts is excessive; CodeMirror 6 handles syntax highlighting and basic editing well
- **Keep Framer Motion**: Beautiful animations — rejected because B2B dashboard users prioritize speed and data density over animation polish
