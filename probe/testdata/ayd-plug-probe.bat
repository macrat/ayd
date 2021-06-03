@echo off

if not "%1" == "plug:empty" (
    echo 2001-02-03T16:05:06Z	HEALTHY	123.456	%1	check %1
)

if "%1" == "plug:invalid-record" (
    echo this is invalid
)

REM test empty line
echo
