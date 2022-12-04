@echo off

if "%2" == "" (
    echo {"time":"2001-02-03T16:05:06Z","status":"HEALTHY","latency":123.456,"target":"%1","message":"%1"}
) else (
    echo {"time":"2001-02-03T16:05:06Z","status":"HEALTHY","latency":123.456,"target":"%1","message":"%1","record":%2}
)
