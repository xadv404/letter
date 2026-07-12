package params

import (
	_ "embed"
)

//go:embed data/burp-parameter-names.txt
var burpParamsRaw string
