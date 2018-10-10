# wireslacker

WireSlacker is a daemon to read Wires-X logs and update a slack channel with news.

```
go build src/github.com/finfinack/wireslacker/wireslacker.go
```

In order to run, you need two things:

* targets: A list of all the target URLs for the logs of your Wires-X server.

  Currently, only HTTP(S) targets are supported while file reads would also be possible.
  The target should look something like this:

    * For node log: http://IP:port/roomlog.html?wipassword=password
    * For room log: http://IP:port/roomlog.html?wipassword=password

  Where obviously some variables need to be filled in. The default port is 46190 and can be
  set in the Wires-X application together with the password.

* webhook: A valid webhook URL for slack for the bot to post messages to.

  For more information on webhooks, see https://api.slack.com/custom-integrations/outgoing-webhooks

  A valid webhook URL starts like this: https://hooks.slack.com/services/

Examples:

1) Run in dry-run (no slack updates):

```
./wireslacker -dry -targets="target1,target2" -webhook="https://hooks.slack.com/services/..."
```

2) Run with actual slack posts:

```
./wireslacker -targets="target1,target2" -webhook="https://hooks.slack.com/services/..."
```

TODOs:

* Improve the whole data model so it streams from the sources rather than reading
  the whole log at source every time. Once the log gets bigger, this may pose a problem.

* Move some of the flags and vars into a config file to make it more flexible.

* Migrate the Slacker processor into something more generic (allowing other use).

* Extend Slacker to format messages a bit more nicely or use at least templating.

* Write unit tests.

