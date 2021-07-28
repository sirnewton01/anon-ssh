# Anonymous SSH Server

This is a reference server implementation for Anonymous SSH access. It can
support a variety of protocols on top of SSH, such as git, scp, gemini and
others with a consistent root path structure making it much easier for
users to cross protocol boundaries with the same or similar paths.

Connections to the server may use any username, although most clients following
the Anonymous SSH access framework will have the name "anonymous" to avoid
leaking any extra trackable details. Any public key at all is permitted
to grant the user access to the service.

This server implementation can work with any protocol built on top of SSH
that makes use of OS commands provided that they are installed on the server
and available on the path of the server. Allowing remote command exeuction
to anonymous users can pose some security risks. There is a list of allowed
commands and arguments that are permitted to run with the server configurable
by the administrator. Similarly, only certain allowed vritual paths are
permitted instead of full access to the server's filesystem.

It is not expected that this server will provide any kind of interactive user
session, such as OS shell access. Instead, commands are expected to run
in individual requests, and only from the allowed command list. Timeouts are
in place to help discourage attempts at interactive access and free server
resources in a timely manner.

## Setup and configuration

As an SSH server, this server requires a host key that can be used by clients
to track and monitor suspicious activity. This key can be generated using the
ssh-keygen tool and the server will generate one automatically at the provided
path when it is first launched.

The server can be configured to host one or more capsules. A capsule has its
own content and commands that are allowed to be run on it. If more than one
capsule is served from a server instance on a particular port number then there
is a default capsule where requests will be routed by default. Additional
capsules declare their unique list of hostnames. The client provides the
HOST environment variable to the hostname that they used to connect to the
service and this information is used to select the capsule, or the default if
no match can be found. This helps to enable the use of virtual hosting where
capsules may reside on the same server, but can be split off to others in the
future.

Each capsule has a list of allowed commands. These are used ot limit the 
types of interactions that anonymous users may request from the service.
The command list has a simple structure with one command per line and a special
path token that represents the virtual path provided by the client request
that is relative to the capsule contents directory. If a command doesn't
match one of the templates provided exactly then the request is ended with
"Command not found."

```
# Comments can go on lines that start with the pound
cat <path>
gemini <path>
scp -f <path>
```

Commands that are allowed will run as the user that is running the server along
with all of their privileges. A layered security approach should be taken to
prevent malicious access to the server. Only the commands that are needed for
the service should be allowed with the precise parameters. The service user
should only have the access needed to run the anonymous SSH service and
protocols in case of a command that is exploited for elevated privileges on the
server. If necessary, the service could be put into a container or VM to
further isolate the possible damage.

Internal paths are usually not very interesting to external users of your
service. Virtualizing paths is a way to make the paths shorter and more relevant
to visitors of your site. This is why they map to a capsule's content directory.

## Verifying SSH client settings

This server has a built-in greeting mechanism that you can use to check your
SSH client settings. You can validate what username the server received,
the public key that was used and the environment settings with the ssh client
like this.

```
$ ssh anonymous@example.com
Welcome anonymous
Your public key is SHA256:xpKAmi3+kC0BbDVRh6FdQHYR4TH6FCEu9iIgOUBCEDF
Your environment: [LANG=en_US.UTF-8 HOST=example.com]
```
