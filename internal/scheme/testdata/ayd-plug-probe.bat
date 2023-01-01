@echo off

if "%1" == "plug:change" (
    echo {"time":"2001-02-03T17:05:06+01:00","status":"HEALTHY","latency":123.456,"target":"changed:plug","message":"check changed:plug"}
) else if "%1" == "plug:extra" (
    echo {"time":"2001-02-03T17:05:06+01:00","status":"HEALTHY","latency":123.456,"target":"%1","message":"with extra","hello":"world"}
) else if not "%1" == "plug:empty" (
    echo {"time":"2001-02-03T17:05:06+01:00","status":"HEALTHY","latency":123.456,"target":"%1","message":"check %1"}
)

if "%1" == "plug:invalid-record" (
    echo this is invalid
)

REM test empty line
echo.
