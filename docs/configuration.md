
## Configuring Server Saver

Server-saver is configured through a JSON formatted configuration file. There are three areas
of configuration:

* The configuration of the proxy-server part of server-saver
* The Launcher configuration
* The Director configuration

### Configuring the listen address

Server-saver needs to know how to listen for new connections. To listen for connections from
any IP on port 25565, place the following in the config:

```json
{
  "bind_address": ":25565"
}
```

### How to manage the target server

Server-saver needs to know how to start & stop the target server, as well as determine its
address so it can forward players to it. The component in server-saver responsible for this
is called the Launcher. There are several Launcher implementations depending on how you want
to manage the target server:

#### Simple Proxy

The simple proxy is the simplest (surprise!), it assumes a target server is running all the time,
therefore cannot start or stop it, and has a fixed address.

To use a web server as an example, you can proxy connections to your computer onto a remote
server such as http://example.com using the following config:

```json
{
  "bind_address": ":80",
  "launcher_configuration": {
    "selected_launcher": "simple_proxy",
    "simple_proxy": {
      "target_address": "example.com:80"
    }
  }
}
```

You would then be able to connect to http://localhost in your web browser, but see the
contents of the remote website http://example.com

Note that on some platforms port 80 will be in use or restricted, so you may need to change
`bind_address` to some other port such as 8080. In that case you would have to connect to
http://localhost:8080 instead.

#### Executable

The executable launcher is the simplest _useful_ launcher, able to start and stop a targetr server
when players connect & disconnect as appropriate. This is appropriate if you have a game server
running on a machine but just want it to shut down when not in use.

This launcher assumes that the executable being launched will immediately start serving on
a static address.

To use a local minecraft installation (on a linux machine) as an example:

```json
{
  "bind_address": ":25565",
  "launcher_configuration": {
    "selected_launcher": "executable",
    "executable": {
      "path": "java",
      "args": [
        "-Xmx1024M",
        "-Xms1024M",
        "-jar",
        "/path/to/minecraft/server.jar",
        "--nogui",
        "--port=25566"
      ],
      "cwd": "/path/to/minecraft/server.jar",
      "address": ":25566"
    }
  }
}
```

This listens on the default minecraft port, and when a player connects, starts the minecraft
executable. Note that you specify:

* The `path` to the executable, the program that runs (in Minecraft's case it's the java runtime)
* The `arg`uments to the executable, parameters that control its behaviour
* The `address` that the target server will be listening on so that the proxy can talk to it
* Optionally, the `cwd` (current working directory), where the program should be run. If left blank
  it will default to the directory the _server-saverd server_ started in.

### Configuring the Director

The Director is the part of server-saver that tracks opening and closing connections and decides
whether to start or stop the target server. To configure it you add a director configuration:

```json
{
  "director_configuration": {
    "idle_period": "5m"
  }
}
```

* `idle_period` configures how long, after all connections are closed, before server-saver will
  shutdown the target server
