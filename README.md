SSH Capsules
============

**What is this?**

TL;DR It's a way to use SSH that's as simple as browsing the web, but supports
interoperability of protocols and tools. A space capsule is a combination
of systems that work together.

This is a convention built on top of the common Secure SHell (SSH) Protocol.
SSH Capsules have these capabilities.

* Provide higher order services than simple infrastructure access
* Client authentication on first use (anonymous access) using public keys
* Offering services based on multiple protocols (eg. scp, gemini, git, rsync)

Clients have a single method to manage their identity that works with multiple
tools and with the flexibility to support different levels of anonymomity.

**What's so special about SSH?**

SSH is a protocol that provides more than remote interactive shells over an
encrypted channel. It is also has a distributed trust mechanism that helps to
prevent "man in the middle" attacks by caching public keys of hosts at the time
of first use and producing very loud errors if the key changes in the future.
The client's identity is configurable, flexible and can be shared between
different protocols, such as remote shell, git or scp. This makes it largely
transparent to the user once it has been configured and allows them to tailor
their identity for use with every and all hosts. Plus, SSH is mature and used
very widely.

**What's the current state?**

When accessing an SSH service a user identifier must be provided and generally
production services require clients to satisfy a challenge to authenticate
using private information known only to the two parties when the user account
is created. Both anonymous and time of first use account creation cannot require
private information for authentication, especially without an agreed user
identifier since there is no side-band channel for that. SSH requires some kind
of user identifier to authenticate, even when using key based authentication.
The local OS username is often chosen as the default to satisfy the requirement,
which unfortunately leaks some potentially private information.

A variety internet services provide a level of access to anonymous users without
and operating system account for them. This is how people use their web browser
to read things on the web. Some also give them the ability to sign up with time
of first use account creation, often without creating heavy weight operating
system accounts for them. Often, these sorts of light-weight accounts are used
when someone wants to create or modify content with a service. In certain niche
cases accounts can be created automatically when a user presents a public
encryption key and can prove that they have access to its private key. See
[Project Gemini] for examples of this kind of super light account creation.

[Project Gemini](https://gemini.circumlunar.space/)

Unfortunately, these sorts of interactions are not yet in common use with
SSH, mostly because there aren't established conventions to permit someone
to connect anonymously at first use of the service. This tends to limit
use to backend interactions with high degrees of trust, such as interactive
shell access to manage low-level infrastructure, instead of high-level
services that we see with the world-wide web.

With a convention in place, users can access capsules as high-level services
with ease. So, how do they get set up?

## Client setup

Clients need these settings to the SSH client configuration when connecting
to a capsule. Note that SSH capsules are built on top of SSH so they require
some kind of user name. This convention specifies that the user name is always
"anonymous" instead of the default local user name to prevent leaking of local
information that could be used for tracking purposes.

```
User Name: anonymous
Port: 1966
Authentications: Public Key (not password)
Identity File: <key for identity with this host>
HOST: <hostname of the server>
LANG: <locale of the client>
TZ: <timezone of the client>
```

OpenSSH is likely one of the most popular SSH clients. It also has a great deal
of potential for customization using its configuration file and a number of
useful command-line utilities for setting it up. Luckily, most of settings
above can be made for capsule access like this.

```
.ssh/config:

Match user anonymous
  IdentitiesOnly yes
  PubkeyAuthentication yes
  PasswordAuthentication no
  PreferredAuthentications publickey
  Port 1966
  Include ~/.ssh/*_anon_config
```

In some capsule configurations this would be all that's needed for access. 
those capsules may produce some kind of greeting when invoked without
any command.

```
ssh anonymous@example.com
```

If you don't have a public key alredy this command will likely fail. You can
run the ssh-keygen tool to generate a default key (identity) to use with all
sites. Note that unless you want to be prompted for your passphrase each time
or you have an SSH integration to your OS keyvault you might want to leave it
empty. The keygen tool will attempt to protect your key by setting OS security
permissions on the files.

```
$ ssh-keygen
Generating public/private rsa key pair.
Your identification has been saved in id_rsa.
Your public key has been saved in id_rsa.pub.
...
```

If you want to use the same identity for all of your capsules then this is
nearly sufficient. However, if you want a separate identity for each site
then you can configure OpenSSH to do that too. This is a strength of SSH as
a protocol since it permits different kinds of identity management depending
on your needs. OpenSSH is so popular that a wide variety of SSH clients and
tools will work with its configuration files.

There is a special include at the end of the ssh configuration above that
include separate files with per-host configurations. These are split out
into separate files to make it easier to add/remove per-host configurations
because the file names are prefixed with the particular host. Here is an entry
for example.com

```
.ssh/example.com_anon_config:

Match user anonymous host example.com
  SetEnv HOST=example.com TZ=America/New_York
  IdentityFile ~/.ssh/example.com_anon_id_rsa 
```

You can see that a special environment variable HOST is set when connecting
to example.com. This will allow SSH servers to support virtual hosting, which
is a capability that other protocols have, such as http and is super useful for
service providers to combine or separate services for administration purposes.
SSH protocol doesn't send to the server the hostname that the client connected
and so this environment variable is provided as a convention to support virtual
hosting.

The final line points to a new SSH private key that doesn't exist yet. So, let's
use ssh-keygen tool to generate the key in there. Note that there are other
kinds of cryptographic algorithms that we can use with the key. RSA is very
commonly used and most likely to work with an SSH server that is configured for
anonymous access.

```
ssh-keygen -b 2048 -t rsa -f ~/.ssh/example.com_anon_id_rsa -q -N ""
```

Now, when you conect to anonymous@example.com it will use this key for that
site and none of the others restricting your identity to only that one site.
This can be useful to help and avoid certain user tracking mechanisms popular
on the net. If you ever want to use a new identity with this site then you can
delete the example.com_anon_id* files and regenerate them.

You may have noticed that we forgot about the LANG and TZ environment variables
described at the top of this section. It turns out that most Unix-based
OSes will usually send them with most OpenSSH client sessions to
the SSH server. Depending of your use of SSH and what tools are being provided
by the host the preferred timezone can be useful for presenting date information
in a form that is the most useful to you. You'll notice that we set the TZ
environment variable, but this can be changed to the preferred timezone string
to suit your needs.

It is expected that much of this configuration so far can be automated with
tools that are developed for SSH Capsules. Also, the idea will be that the
configuration is shared so that you can use different tools (ssh, scp, git, etc.)
with a capsule sharing your identity and trust preferences.

Whenever you first connect with an SSH server you get a prompt to add it
to the known hosts. If the key changes then you get a very loud error message
warning you about it, whether you're using git, rsync, scp or any tool that
there has been a breach in the trust of that site at any time. Most of these
tools will relay this message reliably and exit in error.

```
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
Someone could be eavesdropping on you right now (man-in-the-middle attack)!
It is also possible that the RSA host key has just been changed.
The fingerprint for the RSA key sent by the remote host is
dd:cf:55:31:7a:89:93:13:dd:99:67:c2:a2:19:22:13.
Please contact your system administrator.
Add correct host key in /home/user/.ssh/known_hosts to get rid of this message.
Offending key in /home/user/.ssh/known_hosts:7
RSA host key for 192.168.219.149 has changed and you have requested strict checking.
Host key verification failed.
```

Note that when using SSH Capsules there are no guarantees that the server
will support any particular command, or even interactive shell access. In this
framework it is left up to the user to discover hosts and side-band
communication channels via search engines, DNS, or even word of mouth to
know the kinds of services being offered by the host. One recommendation to
service owners is to have some kind of useful information available with a
bare ssh request to their site.

```
$ ssh anonymous@newhost.com

Welcome

We provide git, scp and rsync access for tracking your tee-off times at the
golf course.
```

## Capsule Addresses

Providing a [Gemini] face to your capsule might also serve as a friendly
introduction. Note that Gemini browser generally uses URL-based addresses
instead of SSH addresses. Here is an example of the same gemini address
expressed in both forms.

[Gemini](cmd/gemini/README.md)

```
gemini anonymous@example.com:/hello.gmi
gemcap://example.com/hello.gmi
```

Both addresses will lead to the same request on the standard capsule port
number over SSH. The first address defaults to port 1966, instead of 22 for
SSH, because of the client configuration entry based on the username of
"anonymous" as shown in the Client Setup. The second address defaults to
that port in the absence of a port in the URL by convention of the "gemcap"
protocol, which is a short easily typed form of (gemini + capsule). Similar
URL protocol name mappings and default ports can be used for other common
capsule services.

The first address represents a command that could be run in a UNIX shell,
much like other SSH-based commands, such as scp, rsync and git. Each tool
has their own method of starting an SSH request with the identity and trust
settings for the host and invoking themselves as a command passing context
information via command line arguments. The two sides generally pass data
through a pipe over SSH. The gemini command run in the capsule is very simple
and looks like this, which is a slight variant of the standard protocol
specification.

```
ssh anonymous@example.com gemini /hello.gmi
20 text/gemini; charset=utf-8<CR><LF>
# Welcome to Example
...
```

Describing the gemini interactions in this transparent way provides a great
deal of flexibility in the client and server implementations. A client could
be a simple command-line tool that supports the SSH-style address arguments and
is capable of making the SSH request. Alternatively, the client can be a Gemini
browser that performs the same actions, from the URL or even SSH addresses. 
The Gemini browser might internally run the gemini command-line tool to perform
the details of the transaction. It's possible that the command-line tool itself
might invoke the OS's SSH commands internally to manage keys and configuration.

## Capsule Setup

Now that clients have a method of making capsule requests with key-based
identification servers can be deployed to support them. In a previous section
you may have noticed that the port number is specified as 1966 and not the
default port 22. This choice was made for a number of reasons. Using a port above
1000 makes it unprivileged and gives administrators more flexibility to run
the server as a non-root user on Unix systems, which can be granted limited
permissions as part of a layered security approach.

OpenSSH itself does not allow easy access to clients without either a
password or knowledge of their public key before connecting. If someday it
were to support such a capability it could be configured to listen to both
the standard port 22 and 1966 at the same time. Until that time, it is most
likely that a separate SSH compatible server technology will be used in which
case it will need a separate port number so that it can run concurrently with
OpenSSH in some configurations. Virtual hosting and proxying is not currently
supported with OpenSSH, which limits some of the options.

An SSH server implementation that support capsules would need to be
capable of running SSH sessions as user "anonymous" that presents a public key
that is unknown to it. The [SSH Capsule Server] implementation here is an example of a
compliant Capsule server that can be flexibly deployed and run in any OS user
account in a variety of operating systems. Hopefully, there will be a variety
of implementations to suit different requirements.

[SSH Capsule Server](cmd/ssh-capsule-server)

### Activity monitoring

Since SSH is often, but not always, used for interactive shells, timeouts are
usually made very large, some implementations will attempt to automatically
re-connect. Generally speaking, anonymous access to an SSH server is likely
to be non-interactive since interactive sessions could become costly as there
are more concurrent sessions happening. It can also permit more snooping of the
server. Most anonymous SSH servers will probably adopt some kind of active
session termination policies based on inactivity. Servers should probably block
direct shell access or exclude them from the lists of allowed commands,
which is covered in the next section.

```
$ ssh anonymous@someplace.com bash
bash: command not found

$ ssh anonymous@someplace.com some_long_running_command
...
Connection to someplace.com closed by remote host.
```

### Allowed command list

Since the recommended policy is to block any interactive shell access to the
anonymous SSH server, it becomes much easier to implement a list of
allowed commands that can run. Many of the tools that run on top of SSH rely on
a version of that same tool to be installed on the server so that they can
call themselves and pass information via command-line parameters and pipes
to stdin/stdout for data transfer. In order for an SSH server to avoid having
to re-implement much of this functionality themselves, they will generally
allow the client to invoke those tools remotely and facilitate the information
passing and pipes. Luckily, many of the tools have predictable commands that
they invoke, such as git-receive-pack, git-upload-pack, scp, etc. Servers can
have configurable lists of the commands and arguments that are permitted.
For some parameters there will need to be some flexibility to provide different
file paths though.

It turns out that allow lists are also a good security practice along with
layers. With anonymous access these kinds of techniques become more critical
as there is less trust of clients and there are fewer ways to block them from
the system.

```
commands:

scp -t <path>
git-receive-pack <path>
git-upload-pack <path>
rsync --server -logDtpr <path> <path>
ls <path>
cat <path>
```

### Path Binding

The paths themselves form a virtual representation of the host to the outside
world. It is unlikely that it would be permissable to access /etc or other
users' home directories as examples. Resources near the root are going to be
the most easily discoverable to outside users and they are probably not
interested in trying to discover the correct path in /var/srv/... to where
relevant content will reside.

Ultimately the virtual paths exposed to the outside world need to be bound to
physical paths in the server. An existing precedent is HTTP, where most servers
provide mechanisms to map out the virtual paths to physical ones. A similar
approach can work here too, except that this will work with multiple protocols
and tools.

```
capsule1/content-location:

/var/srv/content
```

The mapping also serves as a layer of security preventing access to physical
paths that are internal. The SSH server will convert virtual paths to physical
ones before invoking a command and passing in the path parameter(s) with
special care taken to try and avoid maliciously constructed paths from escaping
the virtual directory structure.

### Time of First Use (TOFU) Accounts

Time of first use accounts represent accounts that are generated on-the-fly
when a public key first passes the authentication challenge with SSH, which
verifies that the entity posesses the matching private key, and accessing a
service. These are different than the operating system accounts of the server
because no side-channel passing of information, such as user names, passwords
or even public keys are required. Instead, if a service supports them there
will be some record, storage location allocated for that public key so that
this information can be used when the user interacts with it in the future.
For example, a user's public key might permit them access to a group with
additional comands that are permitted for users of that group.

### Virtual hosting

As mentioned earlier, virtual hosting adds a tremendous amount of flexibility
for managing the infrastructure of deployments. This is why the concept has
been introduced here for anonymous SSH access where the underlying protocol
does not have support for it, which is why it is considered optional, but
strongly recommended in the client configuration. Service implementers may
plan ahead and refuse anonymous SSH connections if the HOST environment
variable is not provided to give themselves the needed flexibility from the
beginning. Others may be more confident in the stability of their deployment
and might not require it.

A virtual host deployed on one server may appear entirely different in terms
of path structure than another one. The same virtual path could point to an
entirely different physical path based on the host name.

```
capsule2/content-location:

/var/srv/soccer
```

### Additional security measures

Throughout this framework it has been mentioned a few times that layered
security is a good idea and there have been measures taken in general to
protect against malicious attacks. It's ultimately up to those who manage
the service infrastructure to decide what security measures are needed based on
the threat model that they face. Sometimes additional measures will be needed.

It is possible that the allowed command list and paths are not sufficient to
protect against certain types of intrusion, even when the server runs as a
restricted OS user account. There can be weaknesses in the server implementation
or the OS level permissions, groups and file-level accesses. Additional
isolation can be achieved with containers, such as Docker, where only the
paths that are permitted will be mounted into the container. If an intrusion
were possible then the filesystem view would only be to the container and not
the parent OS. Virtual machines could also function in the same way.

There are measures recommended above that would limit the amount of time that
a command may be allowed to run without any network activity before it is
forcibly closed. This is an effort to help prevent excessive resource usage on
the server. However, it may be possible that there are other critical things
running there. It depends on how the deployment is being managed. Docker
containers and VM's would offer a way to limit CPU and memory consumption to
help and improve isolation from other critical processes.

As anonymous SSH services become more widely deployed other security problems 
and solutions may appear over time. It is conceivable that capsule
implementations could opt to simulate the protocols for scp, cat, ls, git, etc.
without any direct command invocation of those tools in the server. Since each
request will send fairly predictable command line arguments these could be
detected, parsed and implemented with some level of virtualized storage
instead of the file system. Also, for these tools there are a variety of API
libraries available in different programming languages that might be invoked
directly by the capsule.

## Summary

A convention has been developed to facility robust access to SSH capsules from
both client and server perspectives in the hopes that more services will be deployed
on the internet. While other anonymous access services, such as HTTP exist out
there, many of them only support a narrow set of protocols and tools that work
with them instead of supporting a number of different ones that can use a single
set of trust and identity settings transparently. The hope is that this might
help to foster more of a Unix philosophy in hosted internet services where
each tool can be more focused on its task, but also combined to support a
wealth of flexibility and functionality.
