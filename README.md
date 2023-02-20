Digital Rain
============

Digital Rain is an HTML5 + Canvas demo app written entirely in [Go](http://golang.org/).
It's intended to mimic the look of the [falling text](https://www.youtube.com/watch?v=rpWrtXyEAN0) in the movie [The Matrix](http://www.imdb.com/title/tt0133093/).

[Live Demo](http://tidwall.com/digitalrain/)

Build
-----

Install [Go](http://golang.org/) and [GopherJS](http://github.com/gopherjs/gopherjs)

```bash
# Prepare GOPATH
export PATH=$PATH:$(go env GOPATH)/bin
export GOPATH=$(go env GOPATH)

# Install gopherjs
go install github.com/gopherjs/gopherjs@v1.18.0-beta2

# Install specific go version for gopherjs
go install golang.org/dl/go1.18.10@latest
go1.18.10 download
export GOPHERJS_GOROOT="$(go1.18.10 env GOROOT)"

# Build and Serve
gopherjs build digitalrain.go --minify
gopherjs serve
```

License
-------

Digital Rain is available under the [MIT License](http://github.com/tidwall/digitalrain/LICENSE).
