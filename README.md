## dupeimport

[![Build Status](https://travis-ci.org/nishanths/dupeimport.svg?branch=master)](https://travis-ci.org/nishanths/dupeimport)

Remove duplicate imports that have been imported using different names.

```
go get -u github.com/nishanths/dupeimport

usage: dupeimport [flags] [path ...]
  dupeimport file1.go file2.go dir
  dupeimport -w file1.go           # overwrite source file
  dupeimport -keep named file1.go  # when resolving duplicate imports, keep the shortest named import
  dupeimport -h                    # help
```

## Docs

See [godoc](https://godoc.org/github.com/nishanths/dupeimport). 

Or run:

```
go doc github.com/nishanths/dupeimport
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

running dupeimport on it with default options

```
dupeimport file.go
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
