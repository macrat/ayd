![Ayd?](./assets/logo.svg)

[![CI test status](https://img.shields.io/github/workflow/status/macrat/ayd/CI?label=CI%20test)](https://github.com/macrat/ayd/actions/workflows/ci.yml)
[![Code Climate maintainability](https://img.shields.io/codeclimate/maintainability-percentage/macrat/ayd)](https://codeclimate.com/github/macrat/ayd)
[![Codecov Test Coverage](https://img.shields.io/codecov/c/gh/macrat/ayd)](https://app.codecov.io/gh/macrat/ayd/)
[![Docker Build Status](https://img.shields.io/github/workflow/status/macrat/ayd-docker/build?color=blue&label=docker%20build&logoColor=white)](https://hub.docker.com/r/macrat/ayd)

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

2. Extract downloaded package and put to somewhere that registered to PATH.

3. Run the server with [target URLs](#target-url) (and [schedule](#scheduling) if need) as arguments.

``` shell
$ ayd https://your-service.example.com ping:another-host.example.com
```

4. Check your status page.

- HTML page for browser: [http://localhost:9000/status.html](http://localhost:9000/status.html)
- Plain text page for use in console: [http://localhost:9000/status.txt](http://localhost:9000/status.txt)
- JSON format for handling in program: [http://localhost:9000/status.json](http://localhost:9000/status.json)


## Usage detail

- [Status page and endpoints](#status-page-and-endpoints)
- [Target URL](#target-url)
- [Scheduling](#scheduling)
- [Log file](#log-file)
- [Alerting](#alerting)
- [Daemonize](#daemonize)
- [Other options](#other-options)


### Status page and endpoints

Ayd has these pages/endpoints.

|path                                             |description|
|-------------------------------------------------|-----------|
|[/status.html](http://localhost:9000/status.html)|Human friendly status page in HTML.|
|[/status.txt](http://localhost:9000/status.txt)  |Human friendly status page in plain text.<br />You can use `/status.txt?charset=unicode` and `/status.txt?charset=ascii` for specify charset.|
|[/status.json](http://localhost:9000/status.json)|Machine readable status page in JSON format.|
|[/log.tsv](http://localhost:9000/log.tsv)        |Raw log file in TSV format.<br />You can change period via `since` and `until` query in RFC3339 format like `/log.tsv?since=2000-01-01T00:00:00Z&until=2001-01-01T00:00:00Z`.<br />And you can filter records by target URL using `target` query like `/log.tsv?target=http://example.com`|
|[/metrics](http://localhost:9000/metrics)        |Minimal status page for use by [Prometheus](https://prometheus.io/).|
|[/healthz](http://localhost:9000/healthz)        |Health status page for checking status of Ayd itself.|


### Target URL

Ayd demands URL as targets.
Please see below what you can use as a scheme (protocol).

#### http: / https:

Fetch HTTP(S) page and check status code is 2xx or not.

You can use GET, HEAD, POST, and OPTIONS method by specifying like `http-post://...` or `https-head://...`.
The default method is GET.

Ayd will Follow redirect maximum 10 times.

HTTP will timeout in 10 minutes and report as failure.

examples:
- `http://example.com`
- `https://example.com`
- `http-head://example.com/path/to/somewhere`
- `https-options://example.com/abc?def=ghi`

#### ping:

Send ICMP echo request (a.k.a. ping command) and check the server is connected or not.

Ayd sends 3 packets in 1 second and expects all packets to return.

In Linux or MacOS, Ayd use non-privileged ICMP in default. So, you can use ping even if rootless.
But this way is not work on some platforms for instance docker container.
Please set `yes` to `AYD_PRIVILEGED` environment variable to use privileged ICMP.

Ping will timeout in 10 seconds and report as failure.

examples:
- `ping:example.com`
- `ping:192.168.1.1`

#### tcp:

Connect to TCP and check the service listening or not.

`tcp://` will select IPv4 or IPv6 automatically. You can use `tcp4://` or `tcp6://` to choose IP protocol version.

TCP will timeout in 10 seconds and report as failure.

examples:
- `tcp://example.com:3309`
- `tcp4://127.0.0.1:3309`
- `tcp6://[::1]:3309`

#### dns:

Resolve hostname via DNS and check the host exists or not.

You can specify record type as a `type` query.
Supported type is `A`, `AAAA`, `CNAME`, `MX`, `NS`, and `TXT`.

DNS will timeout in 10 seconds and report as failure.

examples:
- `dns:example.com`
- `dns:example.com?type=AAAA`

#### exec:

Execute external command and check return code is 0 or not.

The command's stdout and stderr will be captured as a message of the status check record.
You should keep output as short as possible because Ayd is not good at record a long message.

You can specify the first argument as the fragment of URL like below.

```
exec:/path/to/command#this-is-argument
```

Above target URL works the same as the below command in the shell.

``` shell
$ /path/to/command this-is-argument
```

And, you can specify environment arguments as the query of URL like below.

```
exec:/path/to/command?something=foobar&hello=world
```

Above target URL works the same as the below command in the shell.

```
$ export something=foobar
$ export hello=world
$ /path/to/command
```

Exec will timeout in 1 hour and report as failure.

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

#### source:

This is a special scheme for load targets from a file.
Load each line in the file as a target URL and check all targets.

Source file is looks like below.

```
# servers
ping:somehost.example.com
ping:anotherhost.example.com
ping:yet.anotherhost.example.com

# services
https://service1.example.com
https://service2.example.com

# you can also read another file
source:./another-list.txt
```

The line that starts with `#` will ignore as a comment.

examples:
- `source:./targets.txt`
- `source:/path/to/targets.txt`

#### plugin

Plugin is a executable file named like `ayd-xxx-probe`, and installed to the PATH directory.

You can use plugin via like `xxx:` scheme after installed it if plugin name is `ayd-xxx-probe`.
Of course, you can change executable file name to change scheme name.

If you want to make your own plugin please read [make plugin](#make-plugin) section.

Plugin will timeout in maximum 1 hour and report as failure.

##### plugin list

- [FTP / FTPS](https://github.com/macrat/ayd-ftp-probe#readme)
- [SMB (samba)](https://github.com/macrat/ayd-smb-probe#readme)
- [NTP](https://github.com/macrat/ayd-ntp-probe#readme)
- or, you can [make your plugin](#make-plugin) yourself.


### Scheduling

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

2. Status of the record that `HEALTHY`, `FAILURE`, `UNKNOWN`, or `ABORTED`.

   * `HEALTHY` means service seems working well.

   * `FAILURE` means service seems failure or stopped.
     You should do something to the target system because the target may be broken if received this status.

   * `UNKNOWN` means Ayd is failed to status checking.
     For example, not found test script, failed to resolve service name, etc.
     You should check the target system, other systems like DNS, or Ayd settings because maybe something worse happened if received this status.

   * `ABORTED` means Ayd terminated during status checking.
     For example, Ayd reports this when terminated Ayd with Ctrl-C.
     You do not have to action about this status because it happens by your operation. (might be you have to check Ayd settings if you do not know why caused this)

3. Latency of the service in milliseconds.

   Some probes like [ping:](#ping) reports average latency, and other probes reports total value..

4. Target URL.

   This URL is the same to passed one as argument, but normalized.
   For example, `ping:somehost?hello=world` to be `ping:somehost` because [ping:](#ping) does not use query values.

5. The detail of status, the reason for failure, or the output of the executed script.

For example, log lines look like below.

```
2001-02-30T16:00:00+09:00	FAILURE	0.544	http://localhost	Get "http://localhost": dial tcp [::1]:80: connect: connection refused
2001-02-30T16:05:00+09:00	UNKNOWN	0.000	tcp:somehost:1234	lookup somehost on 192.168.1.1:53: no such host
2001-02-30T16:10:00+09:00	HEALTHY	0.375	ping:anotherhost	rtt(min/avg/max)=0.31/0.38/0.47 send/rcv=4/4
```

Ayd will save the log file in the current directory in default.
You can change this with `-o` option like below.

``` shell
$ ayd -o /path/to/ayd.log ping:example.com
```

There is no feature to log rotate.
Please consider using the log rotation tool if you have a plan to use it for a long time.
(Ayd can handle the huge log, but it is not easy to investigate the huge log when trouble)

Please use `-o -` option for disable writing log file if you don't use log file.
This is not recommended for production use because Ayd can't restore last status when restore. But, this is may useful for [use Ayd as a parts of script file](#one-shot-mode).


### Alerting

Ayd can kick a URL when a target status checks failure.
You may want to use [exec](#exec), [HTTP](#http), or plugin for alerting.
(Even you can use ping, DNS, etc as alerting. but... it's useless in almost all cases)

Ayd will kick alert at the timing that incident caused, resolved, or message changed.

You can specify alert URL like below.

``` shell
$ ayd -a https://alert.example.com/alert https://target.example.com
```

In the above example, Ayd access `https://alert.example/alert` with the below queries when `https://target.example.com` down.

|query name     |example                        |description                            |
|---------------|-------------------------------|---------------------------------------|
|`ayd_caused_at`|`2001-02-03T16:05:06+09:00`    |The time of the first incident detected|
|`ayd_status`   |`FAILURE`, `UNKNOWN`, `HEALTHY`|The status of target checking          |
|`ayd_target`   |`https://target.example.com`   |The alerting target URL                |
|`ayd_message`  |                               |The message of the incident            |

Alert plugin receives these as arguments.
Please see [Alert plugin](#alert-plugin) section if you want make your own plugin.

#### e-mail (SMTP)

If you want to send an email via SMTP as an alert, you can use [ayd-mailto-alert](https://github.com/macrat/ayd-mailto-alert) plugin.

![The screenshot of Ayd alert in email. You can see service status, target URL, and reason to failure. And there is button to open Status Page.](./assets/email-alert.jpg)

This plugin can use like below.

``` shell
$ export SMTP_SERVER=smtp.example.com:465 SMTP_USERNAME=your-name SMTP_PASSWORD=your-password
$ export AYD_URL="http://ayd-external-url.example.com"

$ ayd -a mailto:your-email@example.com https://target.example.com
```

Please see more information in [the readme of ayd-mailto-alert](https://github.com/macrat/ayd-mailto-alert#readme).

#### Slack

You can send an alert to Slack via [ayd-slack-alert](https://github.com/macrat/ayd-slack-alert) plugin.

![The screenshot of Ayd alert in the Slack. You can see service status, target URL, and reason to failure. And there is button to open Status Page.](./assets/slack-alert.jpg)

This plugin can use like below.

``` shell
$ export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/......"
$ export AYD_URL="http://ayd-external-url.example.com"

$ ayd -a slack: https://target.example.com
```

Please see more information in [the readme of ayd-slack-alert](https://github.com/macrat/ayd-slack-alert#readme).


### Daemonize

#### Use docker

You can use [docker image](https://hub.docker.com/r/macrat/ayd) for execute Ayd.
This image includes ayd, and alert plugin for [email](https://github.com/macrat/ayd-mailto-alert) and [slack](https://github.com/macrat/ayd-slack-alert).

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


### Make plugin

Plugins in Ayd is a executable file named like `ayd-xxx-probe` or `ayd-xxx-alert`, and installed to the PATH directory.

Ayd looks for `ayd-xxx-probe` as target URL or `ayd-xxx-alert` as alert URL, if URL have `xxx:`.
You can change scheme via changing `xxx`, but you can't use URL schemes that `ayd`, `alert`, and the scheme that is supported by Ayd itself.

The output of the plugin will parsed the same way to [log file](#log-file).

The differences from plugin to [`exec:`](#exec) are below.

|                                                       |`exec: `    |plugin                    |
|-------------------------------------------------------|------------|--------------------------|
|scheme of URL                                          |`exec:` only|anything                  |
|executable file place                                  |anywhere    |only in the PATH directory|
|set argument and environment variable in URL           |can         |can not                   |
|receive raw target URL                                 |can not     |can                       |
|record about multiple targets like as [source](#source)|can not     |can                       |

#### Probe plugin

Probe plugin receives target URL as the first argument of the command.

For example, target URL `foobar:your-target` is means like below command.

``` bash
ayd-foobar-probe "foobar:your-target"
```

#### Alert plugin

Alert plugin receives the URL of alert, and 2nd or after arguments is the same as [log file](#log-file) order but without latency.

For example, alert URL `foobar:your-alert` is means like below command.

``` bash
ayd-foobar-alert                \
    "foobar:your-alert"         \
    "2001-02-30T16:05:06+09:00" \
    "FAILURE"                   \
    "1.234"                     \
    "ping:your-target"          \
    "this is message of the record"
```

The output of the probe plugin will parsed the same way to [log file](#log-file), but all target URL will add `alert:` prefix and won't not show in status page.

### Other options

#### Change listen port

You can change the HTTP server listen port with `-p` option.
In default, Ayd uses port 9000.

#### Use HTTPS

You can set cert certificate file and key file via `-c` option and `-k` option.

``` shell
$ ayd -c ./your-certificate.crt -k ./your-certificate.key ping:localhost
```

This option is also enable HTTP/2.

#### Enable authentication for status pages

Ayd has very simple authentication mechanism using Basic Authentication.
You can use it like below.

``` shell
$ ayd -u user:p@ssword ping:localhost
```

For above example, you can access status page using `user` as username and `p@ssword` as password.

This is not very secure because you have to write password to argument. (Attacker can peek arguments of other process easily if you have access to server terminal)
But, this is very easy to setup, and work against end user who don't know how to attack at least.
If you want to more secure option, please consider use reverse proxy like Nginx.

#### One-shot mode

If you want to use Ayd in a script, you may use `-1` option.
Ayd will check status just once and exit when passed `-1` option.

Exit status code is 0 if all targets are healthy.
If some targets are unhealthy, the status code will 1.
And, if your arguments are wrong (or can't resolve hostnames, or exec scripts not found), the status code will 2.
