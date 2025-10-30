# Admin interface

## Plan

User can inspect all db dbdata and all file listing by hfit tool, but can create a download package only by admin approve.

A simple admin web gui should list the pkg-requests pending and the user requiring approve.

The admin can inspect the package.yaml file to check, then approve the pkg download.

## Old

**Abandoned**

This admin interface provides:
- admin authentication by email/password
- admin page is served in port different from the regular API
- add user and public key pairs
- list users in a table
- remove user
- (planned) edit access rule (array of access a la IAM definition)
