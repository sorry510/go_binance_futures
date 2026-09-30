import urllib.request

URLS = [
    "https://bridges.llama.fi/bridgevolume/Ethereum",
    "https://bridges.llama.fi/bridges?includeChains=true",
    "https://api.llama.fi/bridges/bridgevolume/Ethereum",
]
for url in URLS:
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
        with urllib.request.urlopen(req, timeout=12) as response:
            print(url, response.status, response.read(200))
    except Exception as exc:
        print(url, type(exc).__name__, str(exc))
