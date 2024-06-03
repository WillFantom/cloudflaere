# **`cloudflære`**

> ⚠️ This has been made with a personal use-case in mind... It probably isn't
> all that portable to other ways of addressing this issue.

A service that manages DNS records with Cloudflare based on current HTTP routers
found on a given Træfik instance.

For example, if new routers are added to the Træfik instance with the domains
`alice.example.com` and `bob.example.com`, new A or AAAA records will be created
pointing the subdomains to configured IP.

## Usage

Simply run the following compose:
```yaml
name: cloudflaere

services:
  container_name: cloudflaere
  image: ghcr.io/willfantom/cloudflaere:latest
  restart: unless-stopped
  hostname: ${HOSTNAME}
  networks:
    - traefik-network
  environment:
    - CLOUDFLARE_ZONE=
    - CLOUDFLARE_DNS=
    - TRAEFIK_URL=
    - DDNS_IPV4=
    - DDNS_IPV6=

networks:
  traefik-network:
    external: true
```

> Env vars should be populated as described [here](#cloudflære-1). Config file
> or commands are also available.

## Configuration

### DNS

This works best when all cloudflare records (A/AAAA) for a domain are removed.
If any are there, they will not be touched by this application. Also, if any
tweaks are made to a record that has been made by a cloudlfære instance, these
will also be kept. However, tweaks made to a managed record will be removed if
the record is removed (as a result of a træfik router being removed).

### Træfik

For this, the API must be enabled and insecure access allowed (if running both
træfik and cloudflære locally together).

```yaml
 - "--api=true"
 - "--api.insecure=true"
```

### Cloudflære

|      Key       |                                                                   Description                                                                   |  Default   |
| :------------: | :---------------------------------------------------------------------------------------------------------------------------------------------: | :--------: |
|   `verbose`    |                                                       **(bool)** Output debug level logs                                                        |  `false`   |
|   `interval`   |                         **(dur)** Time between each interval, checking both cloudflare dns records and treafik domains                          |    `1m`    |
|   `instance`   | The name used in the magic comment. Should be different for each instance of cloudflaere being run where they the same access to a set of zones | *hostname* |
| **cloudflare** |                                                                                                                                                 |            |
|     `zone`     |                                                A cloudflare API key with `zone:read` permissions                                                |            |
|     `dns`      |                                                A cloudflare API key with `dns:edit` permissions                                                 |            |
|   `proxied`    |                        **(bool)** When createing new records, cloudflaere will set the proxied flag to match this option                        |  `false`   |
|  **traefik**   |                                                                                                                                                 |            |
|     `url`      |                                The full URL of the target traefik instance (including scheme such as `https://`)                                |            |
|    **ddns**    |                                                                                                                                                 |            |
|     `ipv4`     |                                 **(bool)** Manage `A` records and associate them with the IPv4 address reported                                 |  `false`   |
|     `ipv6`     |                               **(bool)** Manage `AAAA` records and associate them with the IPv6 address reported                                |  `false`   |

See the example config file [here](./cloudflaere.yaml).

### Env config

These values can be configured by env vars. To do so, use `_` to express
nesting. For example `cloudflare.zone` would be `CLOUDFLARE_ZONE`

## Manual Control

To allows DNS records to be managed automatically yet still accept manual tweaks
on the cloudflare dashboard, all DNS records created by this tool get a comment.
This comment is visible on the dashboard as `cloudflære:XXX`, where `XXX` is the
hostname of the machine this program runs on. If this comment is not on the
record, this program can not overwrite or modify the record. This also allows
multiple instances to run in parallel on different machines.

## Issues

 - Should there be a domain that has Path rules given to it on 2 different
   systems, this will cause problems! Since this tool dismisses all Path rule
   information, and a domain can only really be pointing to 1 address at a
   time...
