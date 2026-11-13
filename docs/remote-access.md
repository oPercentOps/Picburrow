# Seeing your photos away from home

Picburrow runs on your Raspberry Pi at home. At home, you open it at `http://<your-pi>:8080`. This page explains how to reach it from anywhere else, safely, without being a network expert.

**The short answer:** install **Tailscale** on the Pi and on your phone. It takes about ten minutes, costs nothing for personal use, and your photos stay private: only your own devices can reach them. Step-by-step instructions are below.

## Pick an option

| | Who can open Picburrow | Setup | Router changes | Recommended for |
| --- | --- | --- | --- | --- |
| **1. Tailscale** | only your devices (and people you invite) | 10 min, two apps | none | almost everyone |
| **2. Your own VPN (PiVPN / WireGuard)** | only your devices | 30–60 min | one port forward | people who don't want an outside account |
| **3. A public web address (Caddy + domain)** | anyone with the address (they still need a password) | 1 h+, a domain name | two port forwards | sharing albums with people outside your devices |
| ~~Opening port 8080 on your router~~ | the whole internet, **without encryption** | | | **never**: see [why not](#why-not-just-open-port-8080) |

Options 1 and 2 keep Picburrow off the public internet entirely; that's why they're the safe default. Option 3 puts your sign-in page on the internet; it's protected, but think twice before doing it.

---

## Option 1: Tailscale (recommended)

Tailscale creates a private network between your own devices, wherever they are. Your phone talks to your Pi as if it were at home; nobody else can even see it.

**You need:** the Pi with Picburrow installed, your phone, and a free Tailscale account (you can sign in with Google, Microsoft, Apple or GitHub).

### 1. Install Tailscale on the Pi

On the Pi, run:

```bash
curl -fsSL https://tailscale.com/install.sh | sh
```

```bash
sudo tailscale up
```

The second command prints a link. Open it on any computer or phone and sign in. The Pi now appears in your Tailscale account.

### 2. Install Tailscale on your phone

Install the **Tailscale** app from the App Store or Google Play, sign in with the **same account**, and turn it on.

### 3. Open Picburrow

On your phone, away from home Wi-Fi (switch Wi-Fi off to test), open:

```
http://<the Pi's name>:8080
```

The Pi's name is shown in the Tailscale app and at https://login.tailscale.com/admin/machines (usually `raspberrypi`). That's it.

### 4. Recommended: give it a proper HTTPS address

One more command gives Picburrow a private `https://` address. With HTTPS:

- the slideshow can **keep the screen on** (browsers only allow that over HTTPS, see [below](#the-slideshow-and-https));
- phones offer **"Install app" / "Add to Home Screen"** with the full app experience;
- the connection is encrypted end to end by the browser too.

First, in the Tailscale admin console, open **DNS** and click **Enable HTTPS** (once per account). Then on the Pi:

```bash
sudo tailscale serve --bg 8080
```

It prints your address, something like `https://raspberrypi.tail1234.ts.net`. Use that address from now on, at home and away. It only works on your own Tailscale devices.

### Family members

Each person installs the Tailscale app. Either sign them in to your account, or invite them from the admin console (**Users → Invite**). Then give them their own Picburrow account (Picburrow → **People**).

---

## Option 2: Your own VPN with PiVPN

Same result as Tailscale (only your devices), but you run everything yourself. It needs one change on your internet box (router).

**You need:** access to your router's settings, and either a fixed public IP address or a free dynamic DNS name (PiVPN's installer explains this).

1. On the Pi:

   ```bash
   curl -L https://install.pivpn.io | bash
   ```

   Choose **WireGuard** and accept the defaults.

2. On your router, forward **UDP port 51820** to the Pi. Every router is different; search "port forwarding" plus your router's name.

3. Create a profile for your phone and show it as a QR code:

   ```bash
   pivpn add
   ```

   ```bash
   pivpn -qr
   ```

4. Install the **WireGuard** app on your phone, add a tunnel from the QR code, turn it on, and open `http://<the Pi's home address>:8080`.

The slideshow can't keep the screen on over this plain `http://` address; set the device's screen timeout instead.

---

## Option 3: A public web address (advanced)

Use this only if you need people **outside your own devices** to open share links. Picburrow will be reachable by anyone on the internet, protected by its sign-in, by share-link passwords, and by its limit on wrong-password attempts (10 per minute per visitor).

**You need:** a domain name (for example from your registrar), and access to your router.

1. Point your domain (for example `photos.example.org`) to your home's public IP address (an A record; use dynamic DNS if your IP changes).
2. On the router, forward **TCP ports 80 and 443** to the Pi.
3. Install Caddy on the Pi, which gets and renews the HTTPS certificate by itself:

   ```bash
   sudo apt install caddy
   ```

4. Replace the contents of `/etc/caddy/Caddyfile` with:

   ```
   photos.example.org {
       reverse_proxy localhost:8080
   }
   ```

5. Apply it:

   ```bash
   sudo systemctl reload caddy
   ```

6. Make Picburrow listen only on the Pi itself, so nobody can skip Caddy: in `/etc/default/picburrow`, set `PICBURROW_LISTEN=127.0.0.1:8080`, then restart Picburrow:

   ```bash
   sudo systemctl restart picburrow
   ```

Open `https://photos.example.org`. Share links you create there use that address, so anyone you send them to can open them.

> **Tailscale Funnel** does the same without a domain or router changes (`sudo tailscale funnel --bg 8080`), at a `*.ts.net` address. The same caution applies: your sign-in page becomes public.

---

## Keep it safe

- **Use a long password** for every Picburrow account (Picburrow → **People** → New password). With options 1 and 2 nobody outside can try, but it's still your last line of defence.
- **Install updates.** New Picburrow versions are on the [releases page](https://github.com/oPercentOps/Picburrow/releases); `sudo apt upgrade` keeps Raspberry Pi OS, Tailscale and Caddy current.
- **Share links** show one album to anyone who has the link. Add a password and an end date for anything personal, and turn links off when you're done (album → **Share**).
- **Share links follow the address you're using.** A link made through Tailscale (`raspberrypi…ts.net`) only works for people on your Tailscale network. To send an album to someone outside, create the link from a public address (option 3), or invite them to your Tailscale network.
- **Wrong passwords are limited** to 10 attempts per minute per visitor. Behind a proxy on the same Pi, Picburrow uses the visitor address the proxy passes along (`X-Forwarded-For`, which Caddy sends), so a stranger can't lock you out. If a proxy doesn't pass it along, or runs on another machine, all visitors count as one: still safe, but someone guessing passwords could make everyone wait a minute.

## Why not just open port 8080?

It's tempting, because it's one router setting. But:

- **no encryption:** Picburrow has no HTTPS of its own, so your password and photos would cross the internet readable by anyone in between (public Wi-Fi, for example);
- **the whole internet would see your sign-in page**, with nothing in front of it.

Options 1 and 2 avoid both. Option 3 fixes the first with HTTPS and is a deliberate choice about the second.

## The slideshow and HTTPS

Browsers only let a web page keep the screen on over **HTTPS** (or on the device that runs it). Over a plain `http://` address, including `http://raspberrypi:8080` at home, the slideshow still runs, but the device's own screen timeout can turn the screen off. For a tablet used as a photo frame, either use the Tailscale HTTPS address (option 1, step 4) or set the tablet's screen timeout to "never".

## Could Picburrow set this up by itself?

We looked at a guided setup inside Picburrow and decided against it, for now:

- every option needs an **account or a router change** that only you can make (signing in to Tailscale, a domain name, a port forward);
- installing and configuring Tailscale, a VPN or Caddy needs **administrator rights on the Pi**, and Picburrow deliberately runs as a restricted user that can't change the system. Giving it those rights would make every other part of Picburrow riskier.

So the recommended path is option 1 above: two apps and two commands, with nothing exposed to the internet. If you get stuck, [open an issue](https://github.com/oPercentOps/Picburrow/issues/new?template=bug.md) and say which step.
