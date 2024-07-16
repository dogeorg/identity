# Dogecoin Identity Protocol Handler

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
