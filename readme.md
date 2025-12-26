# xDNS

Lightweight DNS proxy with ad-blocking and caching.

## Installation

```bash
go build
```

## Usage

```bash
./xdns -upstream 1.1.1.1
```

Default upstream is `1.1.1.1:53` (Cloudflare).

## Features

- Blocks ads and tracking domains
- Caches DNS responses
- Auto-downloads blocklists on first run ([StevenBlack's hosts list](https://github.com/StevenBlack/hosts) **,** [anudeepND's blacklist](https://github.com/anudeepND/blacklist))
- Custom upstream DNS server

## Blacklist

Edit `blacklists.ini` to add custom domains:

```
example.com
*.ads.example.com
```

## License

MIT