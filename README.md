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
```

Thanks to the powerful `http.Client` in Golang, you can easily specify a parent proxy for `justget` via the `HTTP_PROXY` env variable. That makes it possible to extend the functionality of `justget` (e.g. caching) by using other mature HTTP proxy server such as [Polipo](https://www.irif.univ-paris-diderot.fr/~jch/software/polipo/).

## Client side:
```shell
curl http://your_justget_server_ip:8123/?url=http://www.google.com/
```
