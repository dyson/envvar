# Envvar
Go environment variable parsing in the style of flag.

Envvar is a fork and modification to the official Go flag package (https://golang.org/pkg/flag/). It has retained everything from flag that makes sense in the context of parsing environment variables and removed everything else.

General use of the two packages are the same with the notable exception of:
 - Usage information for environment variables is not included.
 - Boolean environment variables must contain a strconv.ParseBool() accepted string.

## Documentation
https://godoc.org/github.com/dyson/envvar

## Updates against flag
With envvar being so closely related to the flag package it makes sense to keep an eye on it's commits to see what bug fixes, improvements and features should be carried over to envvar.

Envvar was last checked against https://github.com/golang/go/tree/master/src/flag commit c65ceff125ded084c6f3b47f830050339e7cc74e.

If the above commit is not the latest commit to the flag package please submit an issue. This README should always reflect that envvar has been checked against the last flag commit.

## License
See LICENSE file.