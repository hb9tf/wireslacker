# wireslacker

WireSlacker is a daemon to read Wires-X logs and update a slack channel with news.

```
go build src/github.com/hb9tf/wireslacker/wireslacker.go
```

In order to run, you need two things:

* targets: A list of all the target URLs for the logs of your Wires-X server.

  Currently, only HTTP(S) targets are supported while file reads would also be possible.
  The target should look something like this:

    * For node log: http://IP:port/nodelog.html?wipassword=password
    * For room log: http://IP:port/roomlog.html?wipassword=password

  Where obviously some variables need to be filled in. The default port is 46190 and can be
  set in the Wires-X application together with the password.

* webhook: A valid webhook URL for slack for the bot to post messages to.

  For more information on webhooks, see https://api.slack.com/custom-integrations/outgoing-webhooks

  A valid webhook URL starts like this: https://hooks.slack.com/services/

If the Wires-X server you are polling sits in a different timezone than the server which
runs wireslacker, you will also have to provide the location as a flag (-location). See
https://golang.org/pkg/time/#LoadLocation for more information on how to specify this.

Examples:

1) Run in dry-run (no slack updates):

```
./wireslacker -dry -targets="target1,target2" -webhook="https://hooks.slack.com/services/..."
```

2) Run with actual slack posts:

```
./wireslacker -targets="target1,target2" -webhook="https://hooks.slack.com/services/..."
```
