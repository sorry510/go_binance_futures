# Protocol

1. Freeze canonical ID121 and strict v52 snapshots before replay.
2. Use the same six symbols and 2023-2024 window for both.
3. Keep exact TP8/SL6, fee and slippage identical.
4. Report aggregate, symbol and year results.
5. Use symbol+entry_time+side as an exact overlap key for descriptive attribution.
6. Because single-position sequencing can alter later entries, overlap is descriptive rather than proof of logical set inclusion.
7. Do not alter the 2-3 ATR filter after seeing results.
8. Do not read 2025/2026 OOS from this audit.
