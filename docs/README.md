# InvestOne — Interactive Browser Demo

**[▶ PLAY DEMO — ONE CLICK](https://changxuanyu26-collab.github.io/InvestOne_ENT207-/?autoplay=1)**

The browser demo is a standalone educational prototype that uses only **synthetic** prices, charts, portfolio holdings, and scripted AI responses. It does not make real trades, call any AI service, or retrieve live market data.

## GitHub Pages deployment

This repo uses the GitHub Actions workflow `.github/workflows/pages.yml`.

- In **Settings → Pages**, set **Build and deployment → Source → GitHub Actions**.
- Open **Actions → Deploy Interactive Demo**, and inspect the most recent deployment.
- To rerun it manually, choose **Run workflow → main → Run workflow** if the button is available.
- The workflow also starts automatically when a file in `docs/` is pushed to `main`.

Once the workflow has successfully finished deploying, the live link is:

https://changxuanyu26-collab.github.io/InvestOne_ENT207-/?autoplay=1

If the address displays a GitHub 404, check the **Actions** deployment result; having the HTML file in the repository alone does not publish a Pages website.

## Demo features

Explore dashboard, illustrative market research, paper trading, factor strategy illustration, scripted AI assistant, local alert rules, community perspective comparisons and educational investment frameworks. The Play Demo button starts an automated scene walkthrough.

## Offline alternative

Download the repo and open `docs/index.html` in a browser. It does not require installation or an API key, but some browser features may differ when opened from a local file.
