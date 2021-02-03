# Covid Report

Covid report is an app that demonstrates how a covid vaccination report application might work with [CMS Blue Button](http://bluebutton.cms.gov) and [VA Lighthouse](https://developer.va.gov).

![Covid Report mockup](https://adhoc.team/ad-hoc-vaccine-verification-app-record.7bf9d82b.jpg)

## Tools

The application serves web pages from a [go](https://golang.org) server.

The front end is built using the [USWDS design system](https://designsystem.digital.gov), which gives it a good base for accessibility and ease of development.

## Setup

To get the app up and running, you should have go version 1.14 or greater.

- get your environment variables set up
  - There are some required environment variables; check out `envrc_sample` to see what they should be set to for your app
  - You are not required to use [direnv](https://direnv.net) to set them, but I recommend it.
    - If you do want to use direnv, modify `envrc_sample` with yuor client API keys and secrets, then save the file as `.envrc` and run `direnv allow`
    - Otherwise, make sure the required environment variables are set using whatever mechanism you prefer
  - if you are missing a required environment variable, your app will print something like `panic: Unable to find key <KEY_NAME>`
- install [`mkcert`](https://mkcert.org) if you don't have it already
  - on a mac using homebrew, `brew install mkcert` will do the job
  - in the certs dir, run `mkcert -install && mkcert localhost.dev localhost 127.0.0.1 ::1` to generate certificates suitable for local development and add them to your system's trust store
    - Check out [mkcert.dev](mkcert.dev) for more info on what this does or how it works
- Build the web server: `go build -o covidreport main.go`
- Run the web server: `./covidreport`

### modd

If you want to do development on the app, it can be helpful to have it rebuild itself when you change source files. This repository uses [modd](https://github.com/cortesi/modd) for that purpose. If you want to use it:

- install `modd`
  - on a mac using homebrew, `brew install modd`
- run `modd` in this app's root directory
