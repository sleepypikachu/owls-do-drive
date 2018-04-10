# owls-do-drive
Being a comic management system written in Go on top of [Gin](https://github.com/gin-gonic/gin).
With a sample front end written on top of [Bumla](https://bulma.io/).

## Motivation
This is a V2 re-write of my python/django version of the same backend with a few notable improvements.

 * By rendering the page on the back end crawlers will be able to extract the post image without JavaScript execution.
 * The templating engine means that keeping multiple similar pages (menus etc.) in sync no longer requires multiple edits.
 * The CSS is considerably simpler.
 * The application no longer relies on a specific database. Any datasource implementing the interface from db.go can serve.
 * The admin interface receives a significant facelift.

## Playing with it

This isn't even alpha level software. I'm keen to hear about bugs (raise an issue) but you shouldn't expect to be able to run this as a product and I can't commit to fixing anything you find.

If you want to play:
 * `go get` it and then run `pgsql.sql` against a PostgreSQL database.
 * Copy odd.cfg.sample to odd.cfg and fill in your settings as appropriate.
 * `go build`
 * Either make sure your process can bind to port 80 or export PORT with the desired port.
 * `./owls-do-drive`
 * If that's running you'll probably want to rip out most of static and templates and fit your own layout. (This repository currently contains proprietary graphics but those will be removed with the templating re-work)

## To-Do

### Package for deployment
It's possible to deploy this as a web app, [demo](https://beta.oddcartoons.com), but it's very handcrafted. Create service files and installer packages to allow the application to be deployed trivially.

### Improve error handling
At the moment most errors at startup just cause panic. It would be better to log a more useful error and preserve the stack trace in a crash report for the user to provide.
