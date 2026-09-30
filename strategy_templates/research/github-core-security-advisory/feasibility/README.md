# GitHub Core Repository Security Advisory — Feasibility

Hypothesis: a public vulnerability disclosure in a token's canonical core repository is an externally timestamped negative security event distinct from an actual exploit.

No returns were inspected. Repository identity was inherited unchanged from the frozen GitHub Core Release Catalyst source map.

The official GitHub Repository Security Advisories API was queried for the ten fixed core repositories. In 2023–2024 only ethereum/go-ethereum exposed any public advisories: two high-severity GHSA records. The other nine repositories returned zero public repository advisories.

This is not evidence that the other projects had no vulnerabilities; projects differ materially in whether they publish GitHub Repository Security Advisories. Therefore the source is not a comparable cross-token event universe.

Decision: coverage/data-definition blocked before returns. Do not broaden to arbitrary CVE searches or project-specific disclosure channels after seeing coverage.
