# Switchboard

Responding to HTTP requests the old school way with simple commands.

Switchboard converts HTTP requests into environment variables and data on
STDIN, fires up a command matching a specified route, and STDOUT is parsed
along with the exit status and is turned back into an HTTP response.

Here's an example. Let's start with a minimal config to define a route and a
command. You can put this in a file named `config.yaml` if you're following
along locally.

```
routes:
  "/hello":
    command:
      inline: |
        #!/usr/bin/env bash

        echo "hello world"
```

There. We've got a command that returns `hello world` wired up to the route
`/hello`. So if we start Switchboard with this config and make a request to
`/hello`, the response should be `hello world`. Let's try it out.  

In one shell run the following to install and start Switchboard with our
config.

```
go install github.com/vanstee/switchboard
switchboard -c config.yaml
```

And then in another shell, make the request to `/hello`.

```
$ curl -i http://localhost:8080/hello                                                                                                                                                               
HTTP/1.1 200 OK
Date: Sun, 04 Mar 2018 04:01:41 GMT
Content-Length: 12
Content-Type: text/plain; charset=utf-8

hello world
```

And there you have it, responding to HTTP requests using nothing but a command
the writes to STDOUT.

Take a look in the [examples directory](https://github.com/vanstee/switchboard/tree/master/examples)
for details on more advanced features like setting HTTP status codes, reading
data from a POST request, nesting routes, and running commands in Docker
containers.
