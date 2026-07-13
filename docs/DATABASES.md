# Databases (per-node MariaDB)

Sky Panel can hand your users their own MariaDB/MySQL databases, provisioned on
the **node** that hosts their server. Each node runs its own MariaDB; the daemon
connects to it as an admin and creates a scoped database + user on request. The
panel generates and stores the credentials, enforces a per-user **database
quota** (bought from the Store), and shows the connection details on each
server's **Databases** tab.

Databases are **off by default** — a node only offers them once you install
MariaDB there and point the daemon at it. This guide walks through that.

---

## 1. Install MariaDB on the node

On the machine running `sky-daemon` (Debian/Ubuntu example):

```bash
sudo apt update
sudo apt install -y mariadb-server
sudo systemctl enable --now mariadb
sudo mysql_secure_installation   # set a root password, answer the prompts
```

## 2. Let users reach it

Your users connect to MariaDB directly (from their game server, a DB tool, etc.),
so it must listen on the node's public interface and the port must be open.

Edit the bind address (usually `/etc/mysql/mariadb.conf.d/50-server.cnf`):

```ini
[mysqld]
bind-address = 0.0.0.0
```

Then restart and open the firewall:

```bash
sudo systemctl restart mariadb
sudo ufw allow 3306/tcp        # or your cloud provider's security group
```

> **Security:** exposing 3306 to the internet means anyone can *attempt* to log
> in. Every user Sky Panel creates is scoped to a single database with a strong
> random password, but you should still keep the node patched and consider
> restricting 3306 to known IP ranges if your users have static addresses.

## 3. Create the admin user the daemon will use

The daemon needs an account that can create databases and users **and grant**
privileges. Connect over loopback and create one:

```sql
sudo mysql
CREATE USER 'skyadmin'@'localhost' IDENTIFIED BY 'a-long-random-password';
GRANT ALL PRIVILEGES ON *.* TO 'skyadmin'@'localhost' WITH GRANT OPTION;
FLUSH PRIVILEGES;
```

The daemon connects to this admin over `127.0.0.1`, so `'skyadmin'@'localhost'`
is enough — it never needs to be reachable remotely.

## 4. Point the daemon at MariaDB

Add these to the daemon's environment. If you used Sky Panel's installer, the
daemon runs as a systemd unit and reads its env from
`/opt/sky-panel/sky-daemon.env` (the `EnvironmentFile=` in
`/etc/systemd/system/sky-daemon.service`) — edit that file:

```bash
SKY_DB_ADMIN_USER=skyadmin
SKY_DB_ADMIN_PASSWORD=a-long-random-password
SKY_DB_ADMIN_HOST=127.0.0.1        # default — the daemon dials this to run DDL
SKY_DB_ADMIN_PORT=3306             # default
SKY_DB_PUBLIC_HOST=node1.example.com   # what users connect to (this node's public IP/hostname)
```

Setting **`SKY_DB_ADMIN_USER`** is what turns the feature on. `SKY_DB_PUBLIC_HOST`
is the address baked into the connection details shown to users — set it to the
node's public IP or DNS name (it falls back to `SKY_DB_ADMIN_HOST` if unset,
which is only correct for a single-machine dev setup).

> **Behind NAT / a published Docker port?** If users must connect to MariaDB on
> a *different* port than the daemon uses internally — for example the daemon
> talks to `127.0.0.1:3306` but the port is published/forwarded to the outside
> world as some other port — set **`SKY_DB_PUBLIC_PORT`** to that external port.
> It's the public counterpart to `SKY_DB_PUBLIC_HOST` and only affects the port
> shown in the user-facing connection details; it falls back to
> `SKY_DB_ADMIN_PORT` when unset (the common case where the port is the same
> inside and out).

Reload and restart:

```bash
sudo systemctl daemon-reload
sudo systemctl restart sky-daemon
```

On startup the daemon logs `database provisioning enabled (...)` and advertises
the `databases` capability to the panel. Servers on this node now show a working
**Databases** tab.

## 5. Give users database slots

Databases count against a per-user **quota** dimension, so they can't create
unlimited databases:

- **Users** buy database slots in the **Store** (the "+1 / +3 Databases" items).
- **Admins** can hand everyone a baseline for free by setting
  `quota.default_databases` in **Admin → Settings** (e.g. `2`), and can top up an
  individual user via the store on their behalf.

With at least one slot, a user opens their server's **Databases** tab, clicks
**Create**, names it, and gets the host, port, database name, username, password,
and a ready-to-paste JDBC URL.

---

## How it works / what to expect

- **One MariaDB per node.** A user's database lives on the node hosting the
  server they created it from. Database names are globally unique per node
  (`sky_<rand>_<label>`), so two users never collide.
- **Credentials are stored by the panel** so the owner can view them anytime.
  They're kept in the panel's database alongside node secrets — protect that host
  accordingly.
- **Deleting a server** drops its databases on the node automatically (and the
  panel rows go with it).
- **Older daemons / unconfigured nodes** simply don't advertise the capability;
  creating a database there returns a clear "this node can't provision
  databases" error instead of failing silently.
- **The daemon connects over loopback without TLS.** User-facing connections use
  MariaDB's own TLS configuration — enable `require_secure_transport` in MariaDB
  if you want to force encrypted client connections.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| "this node can't provision databases" | Daemon is older than v0.5.0, or `SKY_DB_ADMIN_USER` isn't set. Update the daemon (`sudo sky-panel-update`) and check the env. |
| Create fails with an access/denied error | The admin user lacks `GRANT OPTION` or `ALL PRIVILEGES` — recreate it as in step 3. |
| Users can't connect but the panel shows credentials | MariaDB isn't listening publicly (`bind-address`) or 3306 is firewalled. See step 2. Check `SKY_DB_PUBLIC_HOST` is the node's reachable address. |
| "databases quota exceeded" | The user has no free slots — buy more in the Store, or raise `quota.default_databases`. |
