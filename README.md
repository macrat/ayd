![Ayd?](./assets/logo.svg)

[![GitHub Actions CI Status](https://github.com/macrat/ayd/actions/workflows/ci.yml/badge.svg)](https://github.com/macrat/ayd/actions/workflows/ci.yml)
[![Code Climate maintainability](https://img.shields.io/codeclimate/maintainability-percentage/macrat/ayd)](https://codeclimate.com/github/macrat/ayd)
[![Codecov Test Coverage](https://img.shields.io/codecov/c/gh/macrat/ayd)](https://app.codecov.io/gh/macrat/ayd/)
[![Docker Cloud Build Status](https://img.shields.io/docker/cloud/build/macrat/ayd)](https://hub.docker.com/r/macrat/ayd)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fmacrat%2Fayd.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fmacrat%2Fayd?ref=badge_shield)

Easiest status monitoring service to check something service is dead or alive.


## Features

- status checking with:
  * [HTTP/HTTPS](#http--https)
  * [ICMP echo (ping)](#ping)
  * [TCP connect](#tcp)
  * [DNS resolve](#dns)
  * [execute external command (or script file)](#exec)
  * [plugin](#plugin)
- [view status page in browser, console, or program.](#status-page-and-endpoints)
- [kick alert if target failure.](#alerting)

### Good at
- Make a status page for temporary usage.

  You can start it via one command! And, stop via just Ctrl-C!

- Make a status page for a minimal system.

  Single binary server, single log file, there is no database!

### Not good at
- Complex customization, extension.

  There is a few extension way, but extensibility is not the goal of this project.

- Investigate more detail.

  This is just for check dead or alive.


## Quick start

1. Download latest version from [release page](https://github.com/macrat/ayd/releases/).

2. Extract downloaded package and place **ayd** (or **ayd.exe**) to some place.

3. Run the server.

``` shell
$ ayd https://your-service.example.com ping:another-host.example.com
```

4. Check your status page.

- HTML page for browser: [http://localhost:9000/status.html](http://localhost:9000/status.html)
- Plain text page for use in console: [http://localhost:9000/status.txt](http://localhost:9000/status.txt)
- Json format for handling in program: [http://localhost:9000/status.json](http://localhost:9000/status.json)


## Usage detail

### Status page and endpoints

Ayd has these pages/endpoints.

|path                                             |description                                                         |
|-------------------------------------------------|--------------------------------------------------------------------|
|[/status.html](http://localhost:9000/status.html)|Human friendly status page in HTML.                                 |
|[/status.txt](http://localhost:9000/status.txt)  |Human friendly status page in plain text.                           |
|[/status.json](http://localhost:9000/status.json)|Machine readable status page in JSON format.                        |
|[/metrics](http://localhost:9000/metrics)        |Minimal status page for use by [Prometheus](https://prometheus.io/).|
|[/healthz](http://localhost:9000/healthz)        |Health status page for checking status of Ayd itself.               |


### Specify target

Ayd demands URI as targets.
Please see below what you can use as a scheme (protocol).

#### http / https

Fetch HTTP(S) page and check status code is 2xx or not.

You can use GET, HEAD, POST, and OPTIONS method by specifying like `http-post://...` or `https-head://...`.
The default method is GET.

Ayd will Follow redirect maximum 10 times.

examples:
- `http://example.com`
- `https://example.com`
- `http-head://example.com/path/to/somewhere`
- `https-options://example.com/abc?def=ghi`

#### ping

Send ICMP echo request (a.k.a. ping command) and check the server is connected or not.

Ayd sends 4 packets in 2 seconds and expects all packets to return.

examples:
- `ping:example.com`
- `ping:192.168.1.1`

#### tcp

Connect to TCP and check the service listening or not.

`tcp://` will select IPv4 or IPv6 automatically. You can use `tcp4://` or `tcp6://` to choose IP protocol version.

examples:
- `tcp://example.com:3309`
- `tcp4://127.0.0.1:3309`
- `tcp6://[::1]:3309`

#### dns

Resolve hostname via DNS and check the host exists or not.

You can specify record type as a `type` query.
Supported type is `A`, `AAAA`, `CNAME`, `MX`, `NS`, and `TXT`.

examples:
- `dns:example.com`
- `dns:example.com?type=AAAA`

#### exec

Execute external command and check return code is 0 or not.

The command's stdout and stderr will be captured as a message of the status check record.
You should keep output as short as possible because Ayd is not good at record a long message.

You can specify the first argument as the fragment of URI like below.

```
exec:/path/to/command#this-is-argument
```

Above target URI works the same as the below command in the shell.

``` shell
$ /path/to/command this-is-argument
```

And, you can specify environment arguments as the query of URI like below.

```
exec:/path/to/command?something=foobar&hello=world
```

Above target URI works the same as the below command in the shell.

```
$ export something=foobar
$ export hello=world
$ /path/to/command
```

examples:
- `exec:./check.exe`
- `exec:/usr/local/bin/check.sh`

##### Extra report output for exec

In exec, you can set latency of service, and status of service with the output of the command.
Please write output like below.

```
::latency::123.456
::status::failure
hello world
```

This output is reporting latency is `123.456ms`, status is `FAILURE`, and message is `hello world`.

- `::latency::`: Reports the latency of service in milliseconds.
- `::status::`: Reports the status of service in `healthy`, `failure`, `aborted`, or `unknown`.

Ayd uses the last value if found multiple reports in single output.

#### plugin

Plugin for check target is almost the same as [`exec:` target](#exec).
The differences are below.

|                                            |`exec: `    |plugin                    |
|--------------------------------------------|------------|--------------------------|
|scheme of URI                               |`exec:` only|anything                  |
|executable file place                       |anywhere    |only in the PATH directory|
|set argument and environment variable in URI|can         |can not                   |
|receive raw target URI                      |can not     |can                       |

Plugin is the "plugin".
This is a good way to extend Ayd (you can use any URI!), but not good at writing a short script (you have to parse URI yourself).

Plugin is an executable file in the PATH directory.
Ayd looks for `ayd-XXX-probe` if found target with `XXX:` scheme.
The file name to be `ayd-XXX-alert` if using as an [alert](#alerting).
In both cases, you can use your wanted scheme by changing `XXX`.

You can't use URI schemes that `ayd`, `alert`, and the scheme that is supported by Ayd itself.

Plugin receives these values as the environment variable.

|variable name|example                |description               |
|-------------|-----------------------|--------------------------|
|`ayd_url`    |`http://localhost:9000`|The URL of Ayd            |
|`ayd_target` |`foobar:hello-world`   |The target URI to checking|

You can use [the directives the same as exec](#extra-report-output-for-exec) in output of plugin.

#### source

This is a special scheme for load targets from a file.
Load each line in the file as a target URI and check all targets.

The line that starts with `#` will ignore as a comment.

examples:
- `source:./targets.txt`
- `source:/path/to/targets.txt`


### Specify check interval/schedule

In default, Ayd will check targets every 5 minutes.

You can place the timing specification before the target specification like below if you want.

``` shell
$ ayd 10m https://your-service.example.com 1h https://another-service.example.com
```

The above command will check `your-service.example.com` every 10 minutes, and check `another-service.example.com` every 1 hour.

You can also use [the Cron](https://en.wikipedia.org/wiki/Cron) style spec as a timing spec like below.

``` shell
$ ayd '*/5  6-21 * *'     https://your-service.example.com \
      '*/10 *    * * 1-5' https://another-service.example.com
```

The above command will check `your-service.example.com` every 5 minutes from 6 a.m. to 9 p.m, and check `another-service.example.com` every 10 minutes from monday to friday.

```
 ┌─────── minute (0 - 59)
 │ ┌────── hour (0 - 23)
 │ │ ┌───── day of the month (1 - 31)
 │ │ │ ┌──── month (1 - 12)
 │ │ │ │ ┌─── [optional] day of the week (0 - 6 (sunday - saturday))
 │ │ │ │ │
'* * * * *'
```


### Log file

Logfile of Ayd is TSV (Tab Separated Values) format.
The log has these columns.

1. Timestamp in [RFC3339 format](https://tools.ietf.org/html/rfc3339) like `2001-02-30T16:05:06+00:00`.

2. Status of the record that `HEALTHY`, `FAILURE`, `ABORTED`, or `UNKNOWN`.

   * HEALTHY means service seems working well.
   * FAILURE means service seems failure or stopped.
   * ABORTED means Ayd terminated during status checking. For example, this reported when terminated Ayd with Ctrl-C.
   * UNKNOWN means Ayd is failed to status checking. For example, not found test script, failed to resolve service name, etc.

3. Latency of the service in milliseconds.

4. Target URI.

5. The detail of status, the reason for failure, or the output of the executed script.

For example, log lines look like below.

```
2001-02-30T16:00:00+09:00	FAILURE	0.544	http://localhost	Get "http://localhost": dial tcp [::1]:80: connect: connection refused
2001-02-30T16:05:00+09:00	UNKNOWN	0.000	ping:somehost	lookup somehost on 192.168.1.1:53: no such host
2001-02-30T16:10:00+09:00	HEALTHY	0.375	ping:mikage	rtt(min/avg/max)=0.31/0.38/0.47 send/rcv=4/4
```

Ayd will save the log file in the current directory in default.
You can change this with `-o` option like below.

``` shell
$ ayd -o /path/to/ayd.log ping:example.com
```

There is no feature to log rotate.
Please consider using the log rotation tool if you have a plan to use it for a long time.
(Ayd can handle the huge log, but it is not easy to investigate the huge log when trouble)


### Alerting

Ayd can kick a URI when a target status checks failure.
You may want to use [exec](#exec), [HTTP](#http), or [plugin](#plugin) for alerting.
(Even you can use ping, DNS, etc as alerting. but... it's useless in almost all cases)

Ayd will kick alert at only the timing that incident caused, and it won't kick at the timing that continuing or resolved the incident.

You can specify alerting URI like below.

``` shell
$ ayd -a https://alert.example.com/alert https://target.example.com
```

In the above example, Ayd access `https://alert.example/alert` with the below queries when `https://target.example.com` down.

|query name      |example                     |description                  |
|----------------|----------------------------|-----------------------------|
|`ayd_url`       |`http://localhost:9000`     |The external URL of Ayd      |
|`ayd_target`    |`https://target.example.com`|The alerting target URI      |
|`ayd_checked_at`|`2001-02-03T16:05:06+09:00` |The checked timestamp        |
|`ayd_status`    |`FAILURE` or `UNKNOWN`      |The status of target checking|

#### e-mail (SMTP)

If you want to send an email via SMTP as an alert, you can use [ayd-mail-alert](https://github.com/macrat/ayd-mail-alert).

![The screenshot of Ayd alert in email. You can see service status, target URI, and reason to failure. And there is button to open Status Page.](./assets/email-alert.jpg)

Please download from [release page of ayd-mail-alert](https://github.com/macrat/ayd-mail-alert/releases) and use like below.

``` shell
$ export SMTP_SERVER=smtp.example.com:465 SMTP_USERNAME=your-name SMTP_PASSWORD=your-password
$ export AYD_MAIL_TO="your name <your-email@example.com>"

$ ayd -a exec:ayd-mail-alert https://target.example.com
```

Please see more information in [the readme of ayd-mail-alert](https://github.com/macrat/ayd-mail-alert#readme).

#### Slack

You can send an alert to Slack via [ayd-slack-alert](https://github.com/macrat/ayd-slack-alert).

![The screenshot of Ayd alert in the Slack. You can see service status, target URI, and reason to failure. And there is button to open Status Page.](./assets/slack-alert.jpg)

Please download from [release page of ayd-slack-alert](https://github.com/macrat/ayd-slack-alert/releases) and use like below.

``` shell
$ export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/......"

$ ayd -a exec:ayd-slack-alert https://target.example.com
```

Please see more information in [the readme of ayd-slack-alert](https://github.com/macrat/ayd-slack-alert#readme).


### Change listen port

You can change the HTTP server listen port with `-p` option.
In default, Ayd uses port 9000.


### Daemonize

#### Use docker

You can use [docker image](https://hub.docker.com/r/macrat/ayd) for execute Ayd.
This image includes ayd, and alert sender for email and slack.

``` shell
$ docker run --restart=always -v /var/log/ayd:/var/log/ayd macrat/ayd http://your-target.example.com
```

Of course, you can also use docker-compose or Kubernetes, etc.
Please see [ayd-docker](https://github.com/macrat/ayd-docker) repository for more information about this contianer image.

#### Systemd

If you using systemd, it is easy to daemonize Ayd.

Please put `ayd` command to `/usr/local/bin/ayd` (you can use another place if you want), and write a setting like below to `/etc/systemd/system/ayd.service`.

``` ini
[Unit]
Description=Ayd status monitoring server
After=network.target remote-fs.target

[Service]
ExecStart=/usr/local/bin/ayd -o /var/log/ayd.log \
    http://your-target.example.com
#   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ please change target

[Install]
WantedBy=multi-user.target
```

And then, you can enable this service.

``` shell
# reload config
$ sudo systemctl daemon-reload

# start service
$ sudo systemctl start ayd

# enable auto start when boot system
$ sudo systemctl enable ayd
```


### Check status just once

If you want to use Ayd in a script, you may use `-1` option.
Ayd will check status just once and exit when passed `-1` option.

Exit status code is 0 if all targets are healthy.
If some targets are unhealthy, the status code will 1.
And, if your arguments are wrong (or can't resolve hostnames, or exec scripts not found), the status code will 2.


## License

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fmacrat%2Fayd.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fmacrat%2Fayd?ref=badge_large)
