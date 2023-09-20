[<img src="https://github.com/insolar/doc-pics/raw/master/st/github-readme-banner.png">](http://insolar.io/?utm_source=Github)

# Insolar Assured Ledger

Insolar Assured Ledger is the most secure, scalable, and comprehensive business-ready blockchain platform.

It provides an environment for running native Insolar and 3rd party applications.

To learn what distinguishes Insolar from other blockchain projects, go through the [list of our features](https://insolar.io/platform?utm_source=Github). 

To get a grip on how Insolar works, take a look at its [architecture overview](https://docs.insolar.io/en/latest/architecture.html#architecture).

## Quick start

To join the Insolar network, download the [latest release](https://github.com/insolar/assured-ledger/ledger-core/releases) and follow the [integration instructions](https://docs.insolar.io/en/latest/integration.html).

You can test Assured Ledger locally:

1. Install everything from the **Prerequisites** section.
2. Install the platform.
3. Deploy it locally.
4. Run tests.

### Prerequisites

Install v1.14 version of the [Golang programming tools](https://golang.org/doc/install#install).

Set the [$GOPATH environment variable](https://github.com/golang/go/wiki/SettingGOPATH).

### Install

1. Download the Assured Ledger package:

   ```
   go get github.com/insolar/assured-ledger/ledger-core
   ```

2. Go to the package directory:

   ```
   cd $GOPATH/src/github.com/insolar/assured-ledger/ledger-core
   ```

3. Install dependencies and build binaries using [the makefile](https://github.com/insolar/assured-ledger/blob/PLAT-182-all-insolard-cli-commands/ledger-core/Makefile) that automates this process:

   ```
   make vendor 
   make
   ```

### Deploy locally

1. In the directory where you downloaded the Assured Ledger package to, run the launcher:

   ```
   scripts/insolard/launchnet.sh -g
   ```

   The launcher script generates the necessary bootstrap data, starts a pulse watcher, and launches a number of nodes.<br> 
   In local setup, the "nodes" are simply services listening on different ports.<br>
   The default number of nodes is 5.  You can vary this number by commenting/uncommenting nodes in the `discovery_nodes` section in `scripts/insolard/bootstrap_template.yaml`.
   
<sub>
You can set -dg for extra debug for conveyor and text log output
Also you can set additional build tags with BUILD_TAGS environment variable
</sub>
   
### Test

#### Benchmark test

When the pulse watcher says `INSOLAR STATE: READY`, you can run a benchmark in another terminal tab/window:

   ```
    bin/benchmark -c=4 -r=25 -k=.artifacts/launchnet/configs/
   ```

     Options:
     * `-k`: Path to the root user's key pair.
     * `-c`: Number of concurrent threads in which requests are sent.
     * `-r`: Number of transfer requests to be sent in each thread.

#### Functional tests

Functional tests run via a requester that runs a complex scenario: creates a number of users with wallets and transfers some money between them.<br>
Upon the first run, the script does it consecutively, upon subsequent runs — concurrently.

When the pulse watcher says `INSOLAR STATE: READY`, run the following command in another terminal tab/window:

   ```
    bin/apirequester -k=.artifacts/launchnet/configs/ -p=http://127.0.0.1:19101/api/rpc
   ```

     Options:
     * `-k`: Path to the root user's key pair.
             All requests for a new user creation must be signed by the root one.
     * `-p`: Node's public API URL. By default, the first node listens on the `127.0.0.1:19101` port. 
             It can be changed in configuration.


Functional test can be executed in kubernetes cluster. You need to have `kubectl` installed.
_This will install nginx ingress in your cluster and uses 80/443 port_

   ```
    make docker-build           # builds new images
    make test_func_kubernetes   # starts insolar network and run tests
    make kube_stop_net          # stops network
   ```
To run from IDE set the environment to
```
INSOLAR_FUNC_RPC_URL_PUBLIC=http://localhost/api/rpc;INSOLAR_FUNC_RPC_URL=http://localhost/admin-api/rpc;INSOLAR_FUNC_KEYS_PATH=/tmp/insolar/;INSOLAR_FUNC_TESTWALLET_HOST=localhost
```
To check your network you can do
```
kubectl -n insolar logs --tail=20 services/pulsewatcher
```
There is a debug pulsewatcher info after failed tests, it's not working yet. **Don't look at it**.

## Run single Insolar node

We use a two-process approach for running a node in production.<br>
"Two-process" means we insulate the network layer into a separate process and give a higher priority to time-critical consensus and network algorithms:

* The first insolard process contains the network layer and runs with a higher priority and resource quota (via the `nice` command or Linux namespaces). You can identify this process by the `passive` flag.

* The second insolard process is responsible for other node components.

You have two options to run Insolar node in production:

 * First process manages the second.
 
 * A watchdog manages each process separately.
 
In case of the first option, you can use a systemd service script:

```
insolard config generate --role=virtual -c ./node.conf
insolard node --role=virtual -c ./node.conf
```

In case of the second option, you can use Kubernetes.<br>
While starting the process, you must add the `--passive` flag and `--pipeline-port` for connection between processes.

```
insolard config generate --role=virtual -c ./node.conf
insolard node --passive --role=virtual -c ./node.conf --pipeline-port=8989
insolard node app-process --role=virtual -c ./node.conf --pipeline-port=8989
```

## Contribute!

Feel free to submit issues, fork the repository and send pull requests! 

To make the process smooth for both reviewers and contributors, familiarize yourself with the list of guidelines:

1. [Open source contributor guide](https://github.com/freeCodeCamp/how-to-contribute-to-open-source).
2. [Style guide: Effective Go](https://golang.org/doc/effective_go.html).
3. [List of shorthands for Go code review comments](https://github.com/golang/go/wiki/CodeReviewComments).

When submitting an issue, **include a complete test function** that demonstrates it.

Thank you for your intention to contribute to the Insolar project. As a company developing open-source code, we highly appreciate external contributions to our project.

## FAQ

For more information, check out our [FAQ](https://github.com/insolar/assured-ledger/ledger-core/wiki/FAQ).

## Contacts

If you have any additional questions, join our [developers chat on Telegram](https://t.me/InsolarTech).

Our social media:

[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-facebook.png" width="36" height="36">](https://facebook.com/insolario)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-twitter.png" width="36" height="36">](https://twitter.com/insolario)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-medium.png" width="36" height="36">](https://medium.com/insolar)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-youtube.png" width="36" height="36">](https://youtube.com/insolar)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-reddit.png" width="36" height="36">](https://www.reddit.com/r/insolar/)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-linkedin.png" width="36" height="36">](https://www.linkedin.com/company/insolario/)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-instagram.png" width="36" height="36">](https://instagram.com/insolario)
[<img src="https://github.com/insolar/doc-pics/raw/master/st/ico-social-telegram.png" width="36" height="36">](https://t.me/InsolarAnnouncements) 

## License

This project is licensed under the terms of the [MIT License](LICENSE.md).
