# Dogecoin Identity Protocol Handler

This project is hosted on [radicle.xyz](https://radicle.xyz) at [rad:z4FoA61FxfXyXpfDovtPKQQfiWJWH](https://app.radicle.xyz/nodes/ash.radicle.garden/z4FoA61FxfXyXpfDovtPKQQfiWJWH)

## What this service does

* Connects to DogeNet on channel "Iden".
* Caches all Identities seen on the network.
* Refreshes their TTL when seen again.
* Provides an API to look up identity by pubkey.
* Allows Identities to be pinned ("Contacts")
* Occasionally gossips identities to peers.

## About Identities

* Identities are broadcast on the "Iden" channel.
* An identity stays active for 30 days after signing.
