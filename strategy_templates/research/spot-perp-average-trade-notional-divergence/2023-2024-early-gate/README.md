# Spot-vs-Perp Average Trade Notional Divergence
Feature is the log ratio of futures average trade notional to spot average trade notional, where average notional is QuoteVolume / trade count. A prior-720h causal z-score first crossing |z|>=3 triggers in the completed futures-hour direction; |z|<1 re-arms. BTC/ETH/BNB/XRP 2023-2024: 397 events, 12h signed mean -0.0183%.

Decision: freeze; no reverse-direction or threshold/window search.
