# HUProxy
HTTP(S)-Upgrade Proxy — Tunnel anything (but primarily SSH) over HTTP
websockets.

## Server setup
This is built to work with [Caddy](https://github.com/mholt/caddy) and [loginsrv](https://github.com/tarent/loginsrv), using Google authentication.

Add the following configuration to caddy:
```
auth.example.com {
    tls example@example.com

    redir 302 {
        if {path} is /
        / /login
    }

    login {
        google client_id={client_id},client_secret={client_secret},scope=https://www.googleapis.com/auth/userinfo.email
        redirect_check_referer false
        redirect_host_file /etc/redirect_hosts.txt
        cookie_domain example.com
    }
}

ssh.example.com {
    tls example@example.com

    jwt {
        path /
        redirect https://auth.example.com/login?backTo=https%3A%2F%2F{host}{rewrite_uri_escaped}
        allow sub example@example.com
    }

    proxy / http://127.0.0.1:8086 {
        transparent
        websocket
    }
}

```

Create a file named `/etc/redirect_hosts.txt` containing the following:
```
ssh.example.com
```

Start proxy:

```
./huproxy
```

## Using the client

Add the following to your SSH config (`~/.ssh/config`):

```
Host shell.example.com
    ProxyCommand /path/to/huproxyclient wss://ssh.example.com/proxy/%h/%p
```

Then you can SSH to `shell.example.com` through the proxy by running:
```
ssh shell.example.com
```

## License
See [LICENSE](LICENSE)

Copyright 2018 David Marby  
Copyright 2017 Google Inc.

This is not a Google product.
