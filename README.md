# Covid Report

Covid report is an app that demonstrates how a covid vaccination report application might work with [CMS Blue Button](http://bluebutton.cms.gov) and [VA Lighthouse](https://developer.va.gov).

[Here](https://adhoc.team/2021/01/28/prototyping-covid-vaccine-verification-app/) is a blog post explaining how it works and what it does.

![Covid Report mockup](https://adhoc.team/ad-hoc-vaccine-verification-app-record.7bf9d82b.jpg)

## Tools

The application serves web pages from a [go](https://golang.org) server.

The front end is built using the [USWDS design system](https://designsystem.digital.gov), which gives it a good base for accessibility and ease of development.

## Oauth clients

This app expects you to have an oauth client registered with the [Blue Button sandbox](https://sandbox.bluebutton.cms.gov) and with [VA Lighthouse](https://developer.va.gov/apply).

- Register an application and save all the keys you are given.
  - For a callback URL, use `https://localhost.dev:6655/callback` for Lighthouse and `https://localhost.dev:6655/bbcallback` for Blue Button
  - For VA Lighthouse, you will only need access to the VA Health API, so you only need to select that one when asked

## Setup

To get the app up and running, you should have go version 1.14 or greater.

- Add localhost.dev as an alias for your local computer
  - Make sure your `/etc/hosts` file has this line: `127.0.0.1 localhost.dev`
  - if you need to edit your hosts file, you will need to be superuser; I use `sudo vim /etc/hosts` but you can use whatever editor you would like
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

### Google Cloud

ad hoc hosts this application and its static assets on google cloud; getting that set up is beyond the scope of this README. If you're interested in this, file an issue on the repository and tag @llimllib
