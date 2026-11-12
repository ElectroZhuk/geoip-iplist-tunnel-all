# GeoIP IP list — merged into `all`

This repository downloads the IPv4 CIDR GeoIP database from OpenCCK, merges every source group into one Xray/V2Ray GeoIP category named `all`, and publishes the binary database and SHA-256 checksum to the `gh-pages` branch.

## Source

`https://iplist.opencck.org/?format=geoip&data=cidr4`

The source URL currently requests **IPv4 CIDR only** (`cidr4`).

## Published files

After the first successful workflow run:

- `https://raw.githubusercontent.com/ElectroZhuk/geoip-iplist-tunnel-all/gh-pages/geoip_iplist.dat`
- `https://raw.githubusercontent.com/ElectroZhuk/geoip-iplist-tunnel-all/gh-pages/geoip_iplist.dat.sha256`

## Automation

The workflow runs every six hours (UTC) and can also be started manually from **Actions → Build and publish GeoIP → Run workflow**.

The workflow validates the protobuf structure and verifies that the output contains exactly one GeoIP entry named `all`. A failed build does not publish a replacement, so the previous published files remain available.

## Xray routing example

For Xray's external GeoIP file syntax, use the file name and category:

```json
"ip": ["ext:geoip_iplist.dat:all"]
```

Keep the file in Xray's configured geodata directory and ensure the routing configuration references the same filename.

## 3x-ui integration

In 3x-ui, add this URL as a custom GeoIP source with the alias `iplist`:

`https://raw.githubusercontent.com/ElectroZhuk/geoip-iplist-tunnel-all/gh-pages/geoip_iplist.dat`

The panel stores custom GeoIP files as `geoip_<alias>.dat`; with alias `iplist`, the expected local filename is `geoip_iplist.dat`. This matches the VPS copy script's source path:

`/usr/local/x-ui/bin/geoip_iplist.dat`

For Xray routing, use:

```json
"ip": ["ext:geoip_iplist.dat:all"]
```

After adding the source, use the panel's GeoFiles update action for this custom source. The existing VPS script can then continue copying the downloaded file to `/var/www/vpn/geo/` and calculating the sidecar SHA-256 for INCY.
