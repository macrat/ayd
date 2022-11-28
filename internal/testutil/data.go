package testutil

import (
	_ "embed"
)

//go:embed testdata/test.log
var DummyLog string
