# Hot Fixture Tool (hfit)

Hot Fixture Tool is built as a client-server architecture:

- server (hfitd) can be deployed as docker image and can access hot db and fs data
- client (hfit) is a CLI interface to the server

## Definition

Service API:
- list DBS
- list tables
- export table definition
- export table data + where filter
- list accessible folder
- list files with filter (stream LDJSON)

Client:
- interactively run all API provided by service 
- define JSON for export package with parameters
- modify export JSON 
- execute export package.

Features:
- service ACL per user
- user auth list
- user authentication by key-pair, challenge, and jet
- storage by redis
- admin user- public key
- simple admin interface based on vanillaJs

