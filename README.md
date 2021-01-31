[Show me the docs!](./docs)

## What is this?

* Do you want to run a game server?
* But **don't want it running unless you're playing on it?**
  * Do you want to save CPU/RAM on your server?
  * Do you want to save money and turn a cloud VM off?
  * Do you want to stop your Raspberry Pi 4 from cooking its way through your floor?
  * Do you want to see if scale-to-zero can be needlessly applied to Minecraft servers?
  
This project is for you!

## Quick Start

### Install the server-saver server 'server-saverd'

Install the server from this repository:

`go install github.com/omaskery/server-saver/cmd/server-saverd`

### Create a configuration file

See the example configuration files in `examples/example-configs` and
[the configuration documentation](./docs/configuration.md).

### Run server-saver

Run the installed server-saver server (server-saverd) using your newly created configuration:

`server-saverd path/to/your/config.json`

## To do:

- [ ] Make it send a signal to the target server before it kills it, if possible
- [ ] Add a 'managed' executable launcher that supports simple communication over standard in/out
      in order to discover the target server's address, and possibly communicate shutdown requests
      without signals (for platforms that don't support them).
- [ ] Add some utility executables for use with the managed executable launcher for starting/stopping
      a VM on a cloud provider (e.g. Google Cloud).
- [ ] Add an IP whitelist to prevent the server waking up every time a web-crawler pings it
- [ ] Make the idle period more configurable
  