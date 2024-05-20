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
  container_name: cloudflære
  image: ghcr.io/willfantom/cloudflaere:latest
  restart: unless-stopped
  hostname: ${HOSTNAME}
  networks:
    - traefik-network
  environment:
    - ZONEID=${CF_ZONE_ID}
    - APIKEY=${CF_API_KEY}
    - EMAIL=${CF_EMAIL}
    - TRAEFIKURL=http://traefik:8080
    - ADDRESS=III.JJJ.KKK.LLL
    - PROXIED=true
    - INTERVAL=30s

networks:
  traefik-network:
    external: true
```

Alongside there must be a `.env` file adding the given vars.

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

### App

The following must be provided (flags, env, or config file):
 - `ZONEID`: the ID for the cloudflare zone to be managed.
 - `APIKEY`: your global cloudflare API key -- assuming your account has
   permissions for the given zone.
 - `EMAIL`: the email address for the associated cloudfare account.
 - `TRAEFIKURL`: the full URL to the traefik instance to be monitored
 - `ADDRESS`: the IPv4 or IPv6 address to be provided as the content to any
   managed records.
 - `PROXIED`: use (if `true`) cloudflare proxy on any created records by default.
 - `INTERVAL`: frequency to poll the træfik API for new/removed http routers.

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

## How it works

The following loop runs at the configured interval (default: `30s`):

 - Gets the list of http routers from træfik
   - In turn, gets the set of rules associated with the routers
   - Removes paths from the rules and dismisses duplicates
   - Dismisses any rules where the root domain is not in the given cloudflare zone
 - Get the list of DNS records in the given cloudflare zone
 - For each router rule:
   - (branch 1) If the record already exists:
     - Check if it has the magic comment, if not ignore/continue (warn)
     - If so, check if the record content is correct, if so ignore/continue (debug)
     - Otherwise, update the record with the expected content (info)
   - (branch 2) If the rule does not exist
     - Create the record (info)
 - For each record that has magic comment:
   - If no router rule matches record name, delete the record (info)
