twit-archive-update
=

twit-archive-update is a script to update local Twitter archives with your newest tweets. With a bit of work, it could probably be run in a cron job to function as an auto updater. Just `go get` the package, and run with your archive path as the first argument ("~"'s will automatically be replaced with your home directory).

## Todo:
* Cut down on the size of the JSON files
* Fix t.co links, to make them actually be clickable
* Update user_details.js with any new user details
* This may be the hackiest script I've ever written, so code cleanup and additional comments are needed.

### Warning: This script has no tests, not enough error checking, and some known and unknown bugs. Please make a backup copy of your archive in case things go haywire.
