![Ayd](./assets/logo.svg)

[![CI Test Status](https://github.com/macrat/ayd/actions/workflows/ci.yml/badge.svg)](https://github.com/macrat/ayd/actions/workflows/ci.yml)
[![Code Climate maintainability](https://img.shields.io/codeclimate/maintainability-percentage/macrat/ayd)](https://codeclimate.com/github/macrat/ayd)
[![Codecov Test Coverage](https://img.shields.io/codecov/c/gh/macrat/ayd)](https://app.codecov.io/gh/macrat/ayd/)
[![Docker Build Status](https://img.shields.io/docker/pulls/macrat/ayd?logo=docker&logoColor=white)](https://hub.docker.com/r/macrat/ayd)

**Ayd** - The easiest alive monitoring tool.

With this only one easy command, you can monitor whether your system is alive or not:

``` shell
$ ayd ping:192.168.1.1 https://example.com
```


## Features

- Check service status using:
  * [ICMP echo (ping)](#ping)
  * [HTTP/HTTPS](#http--https)
  * [FTP/FTPS](#ftp--ftps)
  * [SSH](#ssh)
  * [SFTP](#sftp)
  * [TCP connect](#tcp)
  * [DNS resolve](#dns)
  * [file/directory existence](#file)
  * [execute external command (or script file)](#exec)
  * [plugin](#plugin)

- The [status page](#status-pages-and-endpoints) for using by browsers, consoles, or other programs.

- Send an alert if an incident occurs or is resolved.

### Good at

- Monitoring small systems
- Creating temporary status pages

This tool is designed for small systems, as easy to start using it and easy to quit using it.

It is a single binary tool that uses only plain text log files.
So there is no need to maintain databases or setting files.

### Not good at

- Customization
- Visualization and investigation

This tool is designed for simple checks to see if a service is functioning.
It does not offer complex or advanced features.


## Quick start

1. Download the latest version from [release page](https://github.com/macrat/ayd/releases/).

2. Extract downloaded package and put the binary to somewhere that registered to the PATH.

3. Run the server with [target URLs](#url-scheme) (and [schedule](#scheduling) if need) as arguments.

``` shell
$ ayd https://your-service.example.com ping:another-host.example.com
```

4. Check the status of your services.

You can see the status page on <http://localhost:9000/>, and you can use [HTTP APIs](#status-pages-and-endpoints).


## Reference

Ayd checks whether the targets are alive or not, and reports to the alert targets if the target's status has changed.
The targets and alert targets are specified as URLs, in the command to start Ayd.

A command to start Ayd looks like:

``` plain text
$ ayd  -a exec:/path/to/alert.sh  10m  ping:192.168.1.1 http://example.com
      └────────────┬────────────┘└─┬─┘└───────────────┬───────────────────┘
                   │               │              Target URLs
                   │               │   Check if 192.168.1.1 is responding to ping,
                   │               │   and if http://example.com is serving.
                   │               │
                   │            Schedule
                   │        Check targets every 10 minutes.
                   │
               Alert URL
       Execute /path/to/alert.sh if the target status changed.
```

The [common schemes](#url-scheme) for targets and alerts are supported by Ayd itself.
You can also add other schemes using [plugin](#plugin).

Ayd checks the targets every 5 minutes in default, but you can change it by placing [schedule specification](#sucheduling) before target URL.

While Ayd running, it provides [a simple dashboard and some APIs](#status-pages-and-endpoints).
The log files of Ayd is formatted as JSON, so you can read it or aggregate it easily using common tools like [jq](https://stedolan.github.io/jq/).

- [URL scheme](#url-scheme)
- [Scheduling](#scheduling)
- [Status pages and endpoints](#status-pages-and-endpoints)
- [Log file](#log-file)
- [Tips](#tips)
  * [Daemonize](#daemonize)
  * [Change listen port](#change-listen-port)
  * [Use HTTPS status page on status page](#use-https-on-status-page)
  * [Use Basic Authentication on status page](#use-basic-authentication-on-status-page)
  * [One-shot mode](#one-shot-mode)
  * [Text encoding](#text-encoding)


### URL scheme

Ayd supports below URL schemes in default.

| scheme                             |      as Target     |      as Alert      |
|------------------------------------|:------------------:|:------------------:|
| [`ping:`](#ping)                   | :heavy_check_mark: | :heavy_minus_sign: |
| [`http:` / `https:`](#http--https) | :heavy_check_mark: | :heavy_check_mark: |
| [`ftp:` / `ftps:`](#ftp--ftps)     | :heavy_check_mark: | :heavy_check_mark: |
| [`ssh:`](#ssh)                     | :heavy_check_mark: | :heavy_minus_sign: |
| [`sftp:`](#sftp)                   | :heavy_check_mark: | :heavy_check_mark: |
| [`tcp:`](#tcp)                     | :heavy_check_mark: | :heavy_minus_sign: |
| [`dns:`](#dns)                     | :heavy_check_mark: | :heavy_minus_sign: |
| [`file:`](#file)                   | :heavy_check_mark: | :heavy_check_mark: |
| [`exec:`](#exec)                   | :heavy_check_mark: | :heavy_check_mark: |
| [`source:`](#source)               | :heavy_check_mark: | :heavy_check_mark: |

You can use extra schemes with [plugin](#plugin) if you need.

#### ping:

Send ICMP echo request (a.k.a. ping command) and check if the target is connected or not.

Ayd sends 3 packets in 1 second and expects all packets to return.
These parameter can changed by `AYD_PING_PACKETS` and `AYD_PING_PERIOD` environment variable.

You can specify IPv4 or IPv6 with `ping4:` or `ping6:` scheme.

Ping will timeout in 30 seconds after sent all packets and report as failure.

examples:
- `ping:example.com`
- `ping:192.168.1.1`
- `ping:192.168.1.10#my-server`

##### as Alert

ping does not support to used as an alert URL.

#### http: / https:

Fetch HTTP/HTTPS page and check if the status code is 2xx or not.

You can use GET, HEAD, POST, OPTIONS, and CONNECT method by specifying like `http-post://...` or `https-head://...`.
The default method is GET.

Ayd will Follow redirect maximum 10 times.

HTTP will timeout in 10 minutes and report as failure.

examples:
- `http://example.com/`
- `https://example.com/`
- `http-head://example.com/path/to/somewhere`
- `https-options://example.com/abc?def=ghi`

##### as Alert

If you use HTTP/HTTP as an alert URL, Ayd adds some queries to send information about the incident.

| query name       | example                                    | description                        |
|------------------|--------------------------------------------|------------------------------------|
| `ayd_time`       | `2001-02-03T16:05:06+09:00`                | The timestamp when status changed  |
| `ayd_status`     | `FAILURE`, `DEGRADE`, `UNKNOWN`, `HEALTHY` | The current status of the target   |
| `ayd_latency`    | `123.456`                                  | The latency of the latest checking |
| `ayd_target`     | `https://target.example.com`               | The target URL                     |
| `ayd_message`    | `hello world`                              | The latest message of the target   |
| `ayd_extra`      | `{"hello":"world"}`                        | The Extra values in JSON format    |

#### ftp: / ftps:

Send LIST or MLSD command of FTP and check the result.

It uses `anonymous` as username and password if absent those in the URL.

FTP will timeout in 10 minutes and report as failure.

examples:
- `ftp://example.com/`
- `ftp://example.com/path/to/directory`
- `ftps://foo:bar@example.com/path/to/file.txt`

##### as Alert

Writes the same format logs as the [normal log file](#log-file), over FTP or FTPS, when the service status changed.
It is pretty same as [file:](#file) scheme for alert but uses FTP/FTPS.

#### ssh:

Establish a SSH connection and check if the service is listening or not.

This scheme supports these queries:

- __identityfile__: Path to private key file like id\_rsa.
- __fingerprint__: The fingerprint of the target service, starts with `SHA256:` or `MD5:`.

SSH will timeout in 10 minutes and report as failure.

examples:
- `ssh://username:plain-password@example.com`
- `ssh://username@example.com?identityfile=/path/to/id_rsa`
- `ssh://username:passphrase-for-key@example.com?identityfile=/path/to/id_rsa`
- `ssh://username@example.com?identityfile=/path/to/id_rsa&fingerprint=SHA256:AAAAAA...`

##### as Alert

`ssh:` does not support to be used as an alert URL.

#### sftp:

Connect to an SFTP server and check whether if the file or directory exists or not. 

This scheme supports the same queries as [ssh:](#ssh) scheme.

SFTP will timeout in 10 minutes and report as failure.
examples:
- `sftp://foo:bar@example.com/`
- `sftp://foo@example.com/path/to/directory?identityfile=/path/to/id_rsa`

##### as Alert

Writes the same format logs as the [normal log file](#log-file), over SFTP, when the service status changed.
It is pretty same as [file:](#file) scheme for alert but uses SFTP.

#### tcp:

Connect to TCP and check if the service is listening or not.

`tcp://` will select IPv4 or IPv6 automatically.
But you can also use `tcp4://` or `tcp6://` to choose IP protocol version.

TCP will timeout in 10 seconds and report as failure.

examples:
- `tcp://example.com:3309`
- `tcp4://127.0.0.1:3309`
- `tcp6://[::1]:3309`
- `tcp://192.168.1.10:80#my-server`

##### as Alert

TCP does not support to used as an alert URL.

#### dns:

Resolve hostname via DNS and check if the host exists or not.

You can specify record type as a `type` query like `dns:example.com?type=A`, or as a scheme like `dns-cname:example.com`.
Supported type is `A`, `AAAA`, `CNAME`, `MX`, `NS`, and `TXT`.

There are `dns4:` and `dns6:` scheme for shorthand of `A` and `AAAA` type.

You also can specify the DNS server as a host name of URL like `dns://8.8.8.8/example.com`.

DNS will timeout in 10 seconds and report as failure.

examples:
- `dns:example.com`
- `dns4:example.com`
- `dns:example.com?type=AAAA`
- `dns-cname:example.com`
- `dns://8.8.8.8/example.com`

##### as Alert

DNS does not support to used as an alert URL.

#### file:

Check the file or the directory existence.
It only checks existence, so it does not report error even if it has no enough permission to read the target.

examples:
- `file:./path/to/something`
- `file:/path/to/somewhere`
- `file:/dev/sdc1#does-storage-connected?`

##### as Alert

Writes the same format logs as the [normal log file](#log-file) to the target path, when the service status changed.
It is pretty same as [ftp: / ftps:](ftp--ftps) for alert but writes to a local file.

#### exec:

Execute external command and check if the exit code is 0 or not.
The exit code 0 means HEALTHY, otherwise mean FAILURE. If couldn't execute command, Ayd reports as UNKNOWN.

The command's stdout and stderr will be captured as a message of the status check record.
It is recommended to keep output as short as possible for log readability reason.

You can specify the first argument as the fragment of URL like below.

``` plain text
exec:/path/to/command#this-is-argument
```

Above target URL works like below command in the shell.
(In accurate, Ayd does not use shell to execute command so you can not use some features for example shell variable.)

``` shell
$ /path/to/command this-is-argument
```

And, you can specify environment arguments as the query of URL like below.

``` plain text
exec:/path/to/command?something=foobar&hello=world
```

Above target URL works like below command in the shell.

``` shell
$ export something=foobar
$ export hello=world
$ /path/to/command
```

Exec will timeout in 1 hour and report as failure.

examples:
- `exec:./check.exe`
- `exec:/usr/local/bin/check.sh`

##### Extra report format

Programs executed by `exec:` can report extra values, such as latency or service status, using the syntax `::key::value`.

For example, the output look like this:

``` plain
::latency::123.456
::status::failure
::extra_count::123
::extra_info::this is a test
hello world
```

This output will be parsed by Ayd like this:

``` json
{
  "latency": 123.456,
  "status": "failure",
  "extra_count": 123,
  "extra_info": "this is a test",
  "message": "hello world"
}
```

The `latency` is a latency of the service, in milliseconds.
The `status` is one of `HEALTHY`, `DEGRADE`, `FAILURE`, `ABORTED`, `UNKNOWN`.
The other values are numbers, texts, or JSON values.

You can not set `time`, `target`, and `exit_code`.
If you need to change them, please make a [plugin](#plugin).

If there are multiple keys with the same name in the output, only the last one will be used.

##### as Alert

If you use `exec:` as an alert URL, Ayd sets some environment variables about the incident.
The name of variable and meaning is the same as the queries of [HTTP scheme as alert](#http--https).

##### exec+ssh:

The `exec+ssh:` scheme is a variant of `exec:`.
This scheme executes a command through a SSH connection, and check the exit code.
Please see also [`exec:`](#exec) and [`ssh:`](#ssh).

__IMPORTANT:__ Please allow to use environment variables by SSH server's setting. If you are using OpenSSH, please write like `AllowEnv ayd_*` in to `/etc/ssh/sshd_config`.

examples:
- `exec+ssh://user:password@example.com/usr/bin/test.sh?env_value=variable`
- `exec+ssh://user@example.com/usr/bin/test.sh?identityfile=/path/to/id_ed25519#first-argument`

#### source:

This is a special scheme for loading targets from a file, a remote host, or a command.
Load each line in the file as a target URL and check all targets.

Source file looks like below.

``` plain text
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

The line that starts with `#` will ignored as a comment.

Source file should encoded by UTF-8 with/without BOM or UTF-16 with BOM, but in Windows, you can use legacy encoding.
Please see also [text encoding chapter](#text-encoding).

examples:
- `source:./targets.txt`
- `source:/path/to/targets.txt`

##### source+http: / source+https:

`source+http:` and `source+https:` is variants of `source:`.
These fetch HTTP/HTTPS and load the response as a source file.

__WARNING:__
Please don't use it if you can't completely trust the HTTP server because this scheme can be a backdoor of your server.
For example, Ayd will execute everything even if HTTP server responses `exec:rm#/your/important/directory`

examples:
- `source+https://example.com/targets.txt`

##### source+ftp: / source+ftps:

`source+ftp:` and `source+ftps:` is variants of `source:` that very similar to `source+http:`.
These download source file via FTP/FTPS and load it.

__WARNING:__
Please don't use it if you can't completely trust the source file in the FTP server because this scheme can be a backdoor of your server.
For example, Ayd will execute everything even if the FTP server responses `exec:rm#/your/important/directory`

examples:
- `source+ftps://example.com/targets.txt`

##### source+exec:

`source+exec:` is another variant of `source:`.
It is execute script as the same way as [`exec:`](#exec) and load the output as a source file.

examples:
- `source+exec:./make-targets-list.exe`
- `source+exec:/usr/local/bin/targets.sh`

##### source+ssh:

`source+ssh:` is a variant of `source:` like `source+exec:` but to execute command on remote host.
It is execute script as the same way as [`exec+ssh:`](#exec) and load the output as a source file.

__NOTE:__
It's highly recommended to specify server's fingerprint for security reason.

examples:
- `source+ssh://name:pass@target/usr/bin/make-targets-list.exe?fingerprint=SHA256:AAAA...`
- `source+ssh://name@target/usr/bin/make-targets-list.exe?identityfile=/path/to/key&fingerprint=SHA256:AAAA...`

##### as Alert

Even if use it as an alert URL, the behavior is almost the same, but send alert to the all URLs loaded.

#### Plugin

A plugin is an executable file installed in the PATH directory.

The name of plugin depends on the URL scheme its supports and its purpose, like `ayd-xxx-probe`, `ayd-xxx-alert`, or `ayd-xxx-scheme`.
A plugin named `-probe` is for probing target, named `-alert` is for sending alerts, and named `-scheme` supports both purposes.

For example, if the target URL has the scheme `xxx-yyy:`, Ayd will search these executable files in order of priority:

1. `ayd-xxx-yyy-probe`
2. `ayd-xxx-yyy-scheme`
3. `ayd-xxx-probe`
4. `ayd-xxx-scheme`

The scheme names that supported by Ayd, `ayd`, and `alert`, are reserved and cannot be used by plugins.

The plugin prints result to stdout, in the same format as [log file](#log-file).
Plugins should not report future results, or old results more than 1 hour.

Ayd expects the output of the plugin to be in UTF-8.
However, in Windows, the system's default character encoding can be used.
Please see also [text encoding chapter](#text-encoding).

If a plugin takes longer than 1 hour to execute, it will be timed out and reported as a failure.

The differences from plugin to [`exec:`](#exec) are:

|                                                         | `exec: `     | plugin                     |
|---------------------------------------------------------|--------------|----------------------------|
| URL Scheme                                              | `exec:` only | anything                   |
| executable file directory                               | anywhere     | only in the PATH directory |
| set argument and environment variable in URL            | can          | can not                    |
| receive raw target URL                                  | can not      | can                        |
| record about multiple targets like as [source](#source) | can not      | can                        |

There is [a library for creating plugin](https://pkg.go.dev/github.com/macrat/ayd/lib-ayd).

##### Probe plugin

The probe plugin is for checking the target.
It receives the target URL as the only one argument.

For example, the target URL `foobar:your-target` will be called like:

``` shell
$ ayd-foobar-probe "foobar:your-target"
```

##### Alert plugin

The alert plugin is for sending an alert.
It receives two arguments.
The first argument is an alert URL.
The second one is the latest record that fired the alert in JSON format.

For example, the alert URL `foobar:your-alert` for plugin `ayd-foobar-alert` will be called like:

``` shell
$ ayd-foobar-alert       \
    "foobar:your-alert"  \
    '{"time":"2001-02-30T16:05:06+09:00", "status":"FAILURE", "latency":"1.234", "target":"ping:your-target", "message":"this is message of the record"}'
```

The `target` URLs in the alert plugin's output, will be added `alert:` prefix before store in Ayd, and hide in the status pages.

##### plugin list

###### Probe plugin

- [SMB (samba)](https://github.com/macrat/ayd-smb-probe#readme)
- [NTP](https://github.com/macrat/ayd-ntp-probe#readme)

###### Alert plugin

- [e-mail (SMTP)](https://github.com/macrat/ayd-mailto-alert#readme)

  ![The screenshot of Ayd alert in email. You can see service status, target URL, and reason to failure. And there is button to open Status Page.](./assets/email-alert.jpg)

- [Slack](https://github.com/macrat/ayd-slack-alert#readme)

  ![The screenshot of Ayd alert in the Slack. You can see service status, target URL, and reason to failure. And there is button to open Status Page.](./assets/slack-alert.jpg)


### Scheduling

Ayd will check targets every 5 minutes in default.
You can place the schedule specifications before the target URLs like below if you want to change scheduling.

``` shell
$ ayd 10m https://your-service.example.com \
      1h  https://another-service.example.com https://yet-another-service.example.com
```

The above command means to check `your-service.example.com` every 10 minutes, and to check `another-service.example.com` and `yet-another-service.example.com` every 1 hour.

You can also use [the Cron](https://en.wikipedia.org/wiki/Cron) style spec as a schedule spec like below.

``` shell
$ ayd '*/5  6-21 * *'     https://your-service.example.com \
      '*/10 *    * * 1-5' https://another-service.example.com
```

The above command means checking `your-service.example.com` every 5 minutes from 6 a.m. to 9 p.m, and checking `another-service.example.com` every 10 minutes from monday to friday.

``` plain text
 ┌─────── minute (0 - 59)
 │ ┌────── hour (0 - 23)
 │ │ ┌───── day of the month (1 - 31)
 │ │ │ ┌──── month (1 - 12)
 │ │ │ │ ┌─── [optional] day of the week (0 - 6 (sunday - saturday))
 │ │ │ │ │
'* * * * *'
```


### Status pages and endpoints

Ayd has these pages/endpoints.

| path                                                 | description                                                          |
|------------------------------------------------------|----------------------------------------------------------------------|
| [/status.html](http://localhost:9000/status.html)    | Human friendly status page in HTML.                                  |
| [/status.txt](http://localhost:9000/status.txt)      | Human friendly status page in plain text.                            |
| [/status.json](http://localhost:9000/status.json)    | Machine readable status page in JSON format.                         |
| [/incidents.html](http://localhost:9000/status.html) | Human friendly incident history page in HTML.                        |
| [/incidents.rss](http://localhost:9000/status.rss)   | Incident history feed in RSS 2.0 format.                             |
| [/incidents.csv](http://localhost:9000/status.csv)   | Incident history in CSV format.                                      |
| [/incidents.json](http://localhost:9000/status.json) | Incident history in JSON format.                                     |
| [/log.html](http://localhost:9000/log.html)          | Raw log data in HTML page.                                           |
| [/log.csv](http://localhost:9000/log.csv)            | Raw log file in CSV format.                                          |
| [/log.xlsx](http://localhost:9000/log.xlsx)          | Raw log file in Microsoft Excel (OpenXML Spreadsheet) format.        |
| [/log.ltsv](http://localhost:9000/log.ltsv)          | Raw log file in LTSV (Labeled Tab-Separated Values) format.          |
| [/log.json](http://localhost:9000/log.json)          | Raw log file in JSON format.                                         |
| [/targets.txt](http://localhost:9000/targets.txt)    | The list of target URLs, separated by \\n.                           |
| [/targets.json](http://localhost:9000/targets.json)  | The list of target URLs in JSON format.                              |
| [/metrics](http://localhost:9000/metrics)            | Minimal status page for use by [Prometheus](https://prometheus.io/). |
| [/healthz](http://localhost:9000/healthz)            | Health status page for checking status of Ayd itself.                |


#### Filter log entries

The log endpoints accept the following queries for filtering log entries.

- `since` and `until`: filter logs by datetime in either RFC3339 format (e.g. `2001-02-03T16:05:06+09:00`) or UNIX time (e.g. `981183906`).
  By default, Ayd replies logs from 7 days ago to the current time.

- `limit`: set maximum number of entries in the response.
  You can use `offset` query to fetch more.

  By default, Ayd replies all logs.

- `offset`: set the offset number of the first entry in the response.
  This is usually used in conjunction with the `limit` query for paging.

- `target`: filter entries by target URLs.
  You can use multiple `target` queries as "OR" filtering.

- `query`: filter by a space-delimited query.
  This works as a perfect matching for status, a partial match for target URL and message text.
  You can also use a syntax for filtering latency like `<10ms` or `>=1s`.

examples:
- <http://localhost:9000/log.csv?since=2000-01-01T00:00:00Z&until=2001-01-01T00:00:00Z>: The logs from 2000-01-01 to 2000-12-31.
- <http://localhost:9000/log.csv?since=2021-01-01T00:00:00Z&target=ping:localhost>: The logs about `ping:localhost` since 2021-01-01.
- <http://localhost:9000/log.json?query=-healthy%20ping:>: The logs within recent 7 days that only about unhealthy(`-healthy`) ping(`ping:`) targets.


### Log file

The log file of Ayd is stored in [JSON Lines](https://jsonlines.org/) format, encoded UTF-8.
Each record has at least 4 fields.

- `time` when status check started, in [RFC3339 format](https://tools.ietf.org/html/rfc3339) like `2001-02-30T16:05:06+00:00`.
  Ayd can parse some variant formats like `2001-02-03 16:05:06+0000` or `20010203_160506Z`, and the UNIX time seconds.

- `status` of the record that `HEALTHY`, `DEGRADE`, `FAILURE`, `UNKNOWN`, or `ABORTED`.

   * `HEALTHY` means service seems working well.

   * `DEGRADE` means service seems working but partially degraded.
     You should do something to the target system because the target is not completely healthy.

   * `FAILURE` means service seems failure or stopped.
     You should do something to the target system because the target may be broken if received this status.

   * `UNKNOWN` means Ayd is failed to status checking.
     For example, not found test script, failed to resolve service name, etc.
     You should check the target system, other systems like DNS, or Ayd settings because maybe something worse happened if received this status.

   * `ABORTED` means Ayd terminated during status checking.
     For example, Ayd reports this when terminated Ayd with Ctrl-C.
     You do not have to action about this status because it happens by your operation. (might be you have to check Ayd settings if you do not know why caused this)

- `latency` of the service in milliseconds.

   Some probes like [ping:](#ping) reports average latency, and other probes reports total value..

- `target` URL.

   This URL is the same to passed one as argument, but normalized.
   For example, `ping:somehost?hello=world` to be `ping:somehost` because [ping:](#ping) does not use query values.

- (optional) `message`, the detail of status, the reason for failure, or the output of the executed script.

Log records can have other extra fields.

For example, log lines look like below.

``` jsonl
{"time":"2001-02-30T16:00:00+09:00", "status":"FAILURE", "latency":0.544, "target":"http://localhost", "message":"Get \"http://localhost\": dial tcp [::1]:80: connect: connection refused"}
{"time":"2001-02-30T16:05:00+09:00", "status":"UNKNOWN", "latency":0.000, "target":"tcp:somehost:1234", "message":"lookup somehost on 192.168.1.1:53: no such host"}
{"time":"2001-02-30T16:10:00+09:00", "status":"HEALTHY", "latency":0.375, "target":"ping:anotherhost", "message":"All packets came back", "packets_recv":3, "packets_sent:3, "rtt_avg":0.38, "rtt_max":0.47, "rtt_min":0.31}
```

Ayd will save log files named `ayd_%Y%m%d.log` into the current directory by default.
The `%Y`, `%m`, and `%d` will be replaced with the year, month, and day of month, respectively, of the record.
You can also use `%y` for the year in two characters, `%H` for the hour, `%M` for the minute, and `%%` for the `%` character.

It can change where the logs are saved using the `-f` option like this:

``` shell
$ ayd -f /path/to/%Y/log.json ping:example.com
```

If you want, you can set file name without time specifications to store all logs into a single file.
However, this is not recommended if you plan to run Ayd for a long time.
A large log file is difficult to handle, and can slow down Ayd's log APIs.

If you use `-f -` option, Ayd will not write any log file.
This is not recommended for production use, because Ayd can not restore its last status when it is restarted.
But, this is may useful for [using Ayd as part of a script file](#one-shot-mode).

If you want use log file in other format like CSV, you can download via [HTTP endpoint](#status-pages-and-endpoints), or you can use `ayd conv` subcommand like below.

``` shell
$ cat ayd.log | ayd conv > ayd_log.csv

$ ayd conv ./ayd.log -o ayd_log.csv

$ ayd conv -l ./ayd.log -o ayd_log.ltsv
```


### Tips

#### Daemonize

##### Docker

There is a [docker image](https://hub.docker.com/r/macrat/ayd) for executing Ayd.

``` shell
$ docker run --restart=always -v /var/log/ayd:/var/log/ayd macrat/ayd http://your-target.example.com
```

The container image includes these plugins:

- [SMB probe](https://github.com/macrat/ayd-smb-probe)
- [email alert](https://github.com/macrat/ayd-mailto-alert)
- [slack alert](https://github.com/macrat/ayd-slack-alert)

There are 3 variants of the base images:

- `latest`, `alpine`: Balanced variant. This is tiny but you can use shell.
- `scratch`: Minimal variant. You can use this if you won't use shell.
- `ubuntu`: Large variant. You can use `apt` command for adding command that you want.

##### Systemd

If you using systemd, it is easy to daemonize Ayd.

Please put `ayd` command to `/usr/local/bin/ayd` (you can use another place if you want), and write a setting like below to `/etc/systemd/system/ayd.service`.

``` ini
[Unit]
Description=Ayd status monitoring service
After=network.target remote-fs.target

[Service]
ExecStart=/usr/local/bin/ayd -f /var/log/ayd.log.%Y%m \
    http://your-target.example.com
#   ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ please change target

[Install]
WantedBy=multi-user.target
```

And then, you can enable this service.

``` shell
$ sudo systemctl daemon-reload
$ sudo systemctl start ayd
$ sudo systemctl enable ayd
```

#### Change listen port

You can change the HTTP server listen port with `-p` option.
In default, Ayd uses port 9000.

#### Use HTTPS on status page

You can set certificate file and key file via `-c` option and `-k` option.

``` shell
$ ayd -c ./your-certificate.crt -k ./your-certificate.key ping:localhost
```

The HTTP/2 will be enabled if set these options.

#### Use Basic Authentication on status page

Ayd has very simple authentication mechanism using Basic Authentication.
You can use it like below.

``` shell
$ ayd -u user:p@ssword ping:localhost
```

For above example, you can access status page using `user` as username and `p@ssword` as password.

This is not very secure because you have to write a plain password in the command argument. (Attacker can peek arguments of other process easily if you have access to the server terminal)
But, this is very easy to setup, and at least, it works well against end user who doesn't have access to the server.
If you need more secure option, please consider use reverse proxy like Nginx.

#### One-shot mode

If you want to use Ayd in a script, you can use `-1` option.
Ayd will check status just once and exit when passed `-1` option.

Exit status code will be 0 if all targets are healthy.
If some targets are unhealthy, the status code will be 1.
And, if your arguments are wrong (or can't resolve host names, or exec scripts not found), the status code will be 2.

#### Text encoding

Ayd expects UTF-8 with/without BOM or UTF-8 with BOM as input character encoding.

But in Windows, you can use the system's default character encoding too, for example CP1252 or CP932.
Ayd tries to decode as UTF-8 first, and then tries to use the system's default encoding.
If the text has the BOM, Ayd always follow it.

The characters couldn't decode will replaced by U+FFFD that means unrecognized character before save to the log file.
That means;
- The log file is always valid UTF-8 even if your external command or plugin writes invalid characters.
- You can lose information if external commands or plugins write invalid characters as current encoding.
