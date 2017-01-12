This repository contains a package for interacting with the
[Cloudflare](https://www.cloudflare.com) API.

This package only supports a small subset of the API:

  * Listing zones
  * Listing DNS records
  * Updating DNS records
  * Purging all cached files


# Programs
I have some small programs using the API:

  * cfiupdate allows you to update a specific A record. I wrote it specifically
    to be able to keep a DNS record updated for a host with a dynamic IP, so it
    has the capability to determine the local IP as well, and use that for the
    IP to set.
  * cfpurge provides a way to purge the cache for a domain.
