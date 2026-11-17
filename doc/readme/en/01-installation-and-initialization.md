# Installation and Initialization

[中文](../zh-CN/01-安装与初始化.md) · [Back to English README](../../../README.EN.md)

## Quick Start (First Installation)

1. Download the [Release](https://github.com/sorry510/go_binance_futures/releases) for your operating system and architecture, or build the Go binary using the instructions on this page.
2. Run `cp conf/app.conf.example conf/app.conf` from the project root. Configure Binance account credentials, the database, and Web login. MySQL is recommended.
3. Run `./go_binance_futures sync db` to update the database.
4. Once database synchronization completes successfully, run `./go_binance_futures` to start the application.
5. Open `http://<server-ip>:<web.port>/zmkm/index.html` and sign in with the credentials in `app.conf`. Before enabling automatic trading, verify Binance connectivity, trading permissions, position risk controls, and the database version.

## Usage

> Download the release matching your operating system from https://github.com/sorry510/go_binance_futures/releases, extract and configure it, or compile it yourself using Go.

---

### Editing the Configuration File

For configuration options, see `app.conf.example`. Copy it to `app.conf` and edit the values:

~~~
cp conf/app.conf.example conf/app.conf
~~~

#### Database Configuration

##### SQLite (for evaluation only; not recommended in production)

- In `app.conf`:

~~~
[database]
driver = "sqlite"
path = "./db/coin.db?_journal_mode=WAL&_busy_timeout=5000"
~~~

##### MySQL (install MySQL separately; recommended for better performance)

~~~
[database]
driver = "mysql"
username = ""
password = ""
host= ""
port= ""
dbname = ""
~~~

---

### Running the Application

For an initial deployment or version upgrade, synchronize the database first:

~~~bash
./go_binance_futures sync db
~~~

After synchronization completes and exits, start the service:

~~~bash
./go_binance_futures
~~~

## Operational Precautions

- Binance must be reachable from your network. Binance HTTP and WebSocket share the proxy-pool configuration. Core market and User Data WebSockets start automatically and reconnect; unstable network or proxy connections can cause REST fallbacks and health warnings.
- API key application: [Binance API Management](https://www.binance.com/zh/usercenter/settings/api-management).
- Automated trading uses Ownership isolation and only manages positions/orders created and recorded by their owner. Manually opened or unrecognized positions will not be claimed automatically.
- **After changing `app.conf`, restart the program for configuration changes to take effect.**
- Make sure the account has enough USDT or order placement can fail.
- Avoid sending more than 20 DingTalk push notifications per minute; exceeding limits can cause temporary IP blocking.
- Excessive settings and combinations of features on one IP may exceed Binance API request limits and result in temporary IP restrictions.

---

## Frequently Asked Questions (Web UI)

1. **Where do I enable automated trading after login?** Open **Configuration Center → Futures Trading**, check the master switch and **Allow Long / Allow Short** settings, then check whether the target symbol is enabled under **Futures Trading → Futures Trading**.
2. **Where can I view simulated trades?** Open **Configuration Center → Test Strategy**, and use **View Test Results**, or open **Futures Trading → Test Results**.
3. **Why did my configuration change not take effect?** Configuration Center parameters are updated through its page. System Configuration edits `conf/app.conf`: saving without changes skips disk writes; when content changes, select **Restart to apply** to make the Go program reload it.
4. **Why are Futures prices delayed?** Prices are updated by mandatory WebSocket streams. Check WebSocket freshness/errors on the System Dashboard and your network connection.
5. **Why can't the account open a position?** Check direction switches, maximum positions, maximum losing positions, symbol enable status, available USDT, and whether the Symbol/side already has a Managed position or active controlled order. Binance may restrict some IP regions.
6. **Why is the page slow?** MySQL is recommended for larger datasets; SQLite is more suitable for small environments.
7. **How should I handle API rate-limit errors?** Reduce active symbols, monitoring rules, and high-frequency notifications, then resume after Binance's restriction expires.
8. **Where are AI alerts, Schedulers, and budgets configured?** Open **AI → AI Settings → AI Configuration**, the first/default tab.
9. **Can AI immediately submit real orders?** No. Controlled execution is off by default and requires a Proposal, deterministic Risk checks, human approval, and final verification.
10. **Where can I view service logs?** Open **Logs** in the sidebar. The page reads the application log file, offers severity and keyword filters, and refreshes every five seconds.
11. **Where is Lead Copy Trading, and is live trading available?** Open **Futures Trading** and click **Copy Trading** next to **Funding Rate**. Credentials, risk settings, pause, read-only observation, and local order history are available; **live Lead entries remain locked**.
