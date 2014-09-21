twit-archive-update
=

twit-archive-update is a script to update local Twitter archives with your newest tweets. With a bit of work, it could probably be run in a cron job to function as an auto updater.

## How to:
1. `go get` the package
2. Sign up for a Twitter application <a href="https://apps.twitter.com">here</a>
3. Request an access token, and obtain an API key, API secret, access token, and access token secret.
4. Put those in the TU_KEY, TU_SECRET, TU_TOKEN, and TU_TOKEN_SECRET environment variables respectively.
5. Run with the path to the archive as the first argument (~'s will be handled appropriately)

## Todo:
* Cut down on the size of the JSON files
* Fix t.co links, to make them actually be clickable
* Update user_details.js with any new user details
* This may be the hackiest script I've ever written, so code cleanup and additional comments are needed.

### Warning: This script has no tests, not enough error checking, and some known and unknown bugs. Please make a backup copy of your archive in case things go haywire.
