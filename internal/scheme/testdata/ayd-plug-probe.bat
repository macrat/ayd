@echo off

if "%1" == "plug:change" (
    echo {"time":"2001-02-03T16:05:06Z","status":"HEALTHY","latency":123.456,"target":"changed:plug","message":"check changed:plug"}
) else if not "%1" == "plug:empty" (
    echo {"time":"2001-02-03T16:05:06Z","status":"HEALTHY","latency":123.456,"target":"%1","message":"check %1"}
)

if "%1" == "plug:invalid-record" (
    echo this is invalid
)

REM test empty line
echo.
