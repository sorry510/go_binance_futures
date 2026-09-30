# Frozen Feasibility Protocol

Input: archived Snapshot proposals from governanceID-safe token mappings.

Candidate event: proposal end timestamp for exactly binary proposals whose choices map mechanically to one positive and one negative governance option.

Signal: negative option wins => SHORT at the next complete market boundary. No LONG mirror is defined.

Feasibility gate before any return inspection: at least 8 eligible events with meaningful cross-symbol coverage. If the gate fails, stop without price replay.

No opposition-share threshold tuning, title/body semantic expansion, or ambiguous multi-choice proposals are allowed after seeing counts.
