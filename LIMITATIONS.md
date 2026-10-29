# Current limitations

- **Public demo is offline.** Prices, simulated positions, charts, rules, announcements and responses are illustrative and synthetic.
- **Full desktop functionality is separate.** GitHub Pages cannot run the Go backend, persistence, continuous monitoring, or Highway AI calls on its own.
- **Mac M-series executable has not been tested on the user's physical machine in this build environment.** The code was cross-compiled for `darwin/arm64`.
- **Live external financial endpoints are not end-to-end verified** in this offline testing environment; feeds may fail, be delayed, or restrict access.
- **API cost and authorization:** the local app requires the user to supply a valid key to the Highway gateway. A key must never be embedded in the public demo.
- **Financial research is educational.** Historical backtests are not predictions and may have look-back and universe-selection biases; market prices may be stale.
- **Availability:** continuous monitoring works only while the local server is running and the computer is awake.
- **Security:** this is not a production-grade authentication or multi-user investment service.
