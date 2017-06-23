# Envvar

[![Build Status](https://travis-ci.org/dyson/envvar.svg?branch=master)](https://travis-ci.org/dyson/envvar)
[![Coverage Status](https://coveralls.io/repos/github/dyson/envvar/badge.svg?branch=master)](https://coveralls.io/github/dyson/envvar?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/dyson/envvar)](https://goreportcard.com/report/github.com/dyson/envvar)

[![GoDoc](https://godoc.org/github.com/dyson/envvar?status.svg)](http://godoc.org/github.com/dyson/envvar)
[![license](https://img.shields.io/github/license/dyson/envvar.svg)](https://github.com/dyson/envvar/blob/master/LICENSE)

Go environment variable parsing in the style of flag.

Envvar is a fork and modification to the official Go flag package (https://golang.org/pkg/flag/). It has retained everything from flag that makes sense in the context of parsing environment variables and removed everything else.

General use of the two packages are the same with the notable exception of:
 - Usage information for environment variables is not included.
 - Boolean environment variables must contain a strconv.ParseBool() accepted string.

## Documentation
https://godoc.org/github.com/dyson/envvar

## Installation
Using dep for dependency management (https://github.com/golang/dep):
```
dep ensure github.com/dyson/envvar
```

Using go get:
```
$ go get github.com/dyson/envvar
```
## Usage
Usage is essentially the same as the flag package. Here is an example program demonstrating envvar and the flag package being used together.

```
// example.go
package main

import (
	"flag"
	"fmt"

	"github.com/dyson/envvar"
)

type conf struct {
	a int
	b int
	c int
}

func main() {
	conf := &conf{
		a: 1,
		b: 2,
		c: 3,
	}

	// Define flags and envvars.
	flag.IntVar(&conf.a, "a", conf.a, "Value of a")
	envvar.IntVar(&conf.a, "A", conf.a)

	flag.IntVar(&conf.b, "b", conf.b, "Value of b")
	envvar.IntVar(&conf.b, "B", conf.a)
	
	flag.IntVar(&conf.c, "c", conf.c, "Value of c")
	envvar.IntVar(&conf.c, "C", conf.a)

	// Parse in reverse precedence order.
	// Flags overwrite environment variables in this example.
	envvar.Parse()
	flag.Parse()
	
	// Print results
	fmt.Println("a set by flag precedence:", conf.a)
	fmt.Println("b set by env var as no flag set:", conf.b) 
	fmt.Println("c set to default value as neither flag or env var set it:", conf.c)

}
```

Running example:
```
$ A=100 B=2 go run example.go -a 3
a set by flag precedence: 3
b set by env var as no flag set: 2
c set to default value as neither flag or env var set it: 1
```

## Updates against flag
With envvar being so closely related to the flag package it makes sense to keep an eye on it's commits to see what bug fixes, improvements and features should be carried over to envvar.

Envvar was last checked against https://github.com/golang/go/tree/master/src/flag commit c65ceff125ded084c6f3b47f830050339e7cc74e.

If the above commit is not the latest commit to the flag package please submit an issue. This README should always reflect that envvar has been checked against the last flag commit.

## License
See LICENSE file.