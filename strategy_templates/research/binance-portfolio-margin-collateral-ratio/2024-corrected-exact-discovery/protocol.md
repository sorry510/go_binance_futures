# 2024 Corrected Exact Discovery Protocol

1. Use exactly the 8 production-eligible events frozen by ../2024-feasibility-correction/. No asset/event deletion after outcomes.
2. Preserve preregistered direction: collateral-ratio increase -> LONG; decrease -> SHORT.
3. Signal time is the official effective time, not publishDate.
4. Entry is the next complete 1m open strictly after effective time.
5. Exact replay: leverage=4; TP=8; SL=6; fee=0.0005/side; adverse slippage=5bps/side; funding included; maximum hold=72h.
6. TP/SL are evaluated on each completed 1m close using the audited Engine-mirror ROI rule `ROI >= 8 || ROI <= -6`; exit is the next minute open.
7. Funding uses Binance Vision fundingRate; LONG pays positive funding, SHORT receives positive funding; if historical MarkPrice is absent use that funding minute's 1m close.
8. Single event trade; no overlap filtering is needed unless two events for the same symbol overlap.
9. Discovery gate: PF >=1.15, aggregate normalized net >0, and >=3/4 independent official article batches have positive mean net.
10. Failure freezes the family and 2025+ remains unread. Pass allows a separately frozen 2025+ OOS only; no threshold/direction/TP-SL retuning.
