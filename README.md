justget
=======

# Usage

## Server side:
```shell
go install github.com/zjx20/justget

# Start the server
$GOPATH/bin/justget
```

```
Usage of justget:
  -addr string
    	Server listen ip (default "0.0.0.0")
  -cert string
    	TLS certificate
  -key string
    	TLS certificate private key
  -port int
    	Server listen port (default 8123)
  -tls-port int
    	Server listen port for TLS (default 8124)
```

Thanks to the powerful `http.Client` in Golang, you can easily specify a parent proxy for `justget` via the `HTTP_PROXY` env variable. That makes it possible to extend the functionality of `justget` (e.g. caching) by using other mature HTTP proxy server such as [Polipo](https://www.irif.univ-paris-diderot.fr/~jch/software/polipo/).

## Client side:
```shell
# Note: the "url" parameter should be urlencoded
curl http://your_justget_server:8123/?url=http%3A%2F%2Fwww.google.com%2F
```

or

```shell
URL=http://www.google.com/
BASE64URL=$( echo -n "${URL}" | base64 | python -c "import urllib; import sys; sys.stdout.write(urllib.quote_plus(sys.stdin.read()))" )
curl http://your_justget_server:8123/?base64Url=${BASE64URL}
```

It's recommended to use the "base64Url" parameter instead of the "url" parameter, for better obfuscation.
