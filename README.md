# go2ts

[![Build Status](https://travis-ci.org/alanshaw/go2ts.svg?branch=main)](https://travis-ci.org/alanshaw/go2ts)
[![Coverage](https://codecov.io/gh/alanshaw/go2ts/branch/master/graph/badge.svg)](https://codecov.io/gh/alanshaw/go2ts)
[![Standard README](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg)](https://github.com/RichardLitt/standard-readme)
[![pkg.go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/alanshaw/go2ts)
[![golang version](https://img.shields.io/badge/golang-%3E%3D1.15.0-orange.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/alanshaw/go2ts)](https://goreportcard.com/report/github.com/alanshaw/go2ts)

Convert golang types to Typescript declarations.

Note:
* `chan T` is converted to `AsyncIterable<T>`.
* Assumes functions/methods are async so return values are all `Promise<T>` and errors assumed to be thrown not returned.
* If a function returns multiple values they are returned as an array.
* Context in function params is ignored.
* Recursion is NOT supported.
* Interfaces are converted to `any`.
* `struct` methods are NOT converted, but `Converter.ConvertMethod` can be used to create method declarations.

## Install

```sh
go get github.com/alanshaw/go2ts
```

## Usage

### Example

```go
package main

import (
  "reflect"
  "github.com/alanshaw/go2ts"
)

type User struct {
  Name string
}

func main () {
  c := go2ts.NewConverter()

  c.Convert(reflect.TypeOf("")) // string
  c.Convert(reflect.TypeOf(User{})) // { Name: string }
  c.Convert(reflect.TypeOf(func(string, int, bool) User { return nil })
  // (str: string, int: number, bool: boolean) => Promise<{ Name: string }>

  // Add custom type declarations
  c.AddTypes(map[reflect.Type]string{
    reflect.TypeOf(User{}): "User"
  })

  // Output now includes "User" instead of { Name: string }
  c.Convert(reflect.TypeOf(map[string]User{})) // { [k: string]: User }
}
```

## API

[pkg.go.dev Reference](https://pkg.go.dev/github.com/alanshaw/go2ts)

## Contribute

Feel free to dive in! [Open an issue](https://github.com/alanshaw/go2ts/issues/new) or submit PRs.

## License

[MIT](LICENSE) © Alan Shaw
