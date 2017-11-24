## dedupimport

[![Build Status](https://travis-ci.org/nishanths/dedupimport.svg?branch=master)](https://travis-ci.org/nishanths/dedupimport)

Remove duplicate imports that have been imported using different names.

```
go get -u github.com/nishanths/dedupimport

usage: dedupimport [flags] [path ...]
  dedupimport file1.go file2.go dir
  dedupimport -w file1.go           # overwrite source file
  dedupimport -keep named file1.go  # when resolving duplicate imports, keep the shortest named import
  dedupimport -h                    # help
```

## Docs

See [godoc](https://godoc.org/github.com/nishanths/dedupimport). 

Or run:

```
go doc github.com/nishanths/dedupimport
```

## Example

Given the file

```
package pkg

import (
	"code.org/frontend"
	fe "code.org/frontend"
)

var client frontend.Client
var server fe.Server
```

running dedupimport on it with default options

```
dedupimport file.go
```

will produce

```
package pkg

import (
	"code.org/frontend"
)

var client frontend.Client
var server frontend.Server
```
