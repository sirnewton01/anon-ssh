Anonymous SSH
=============

This project creates a framework for SSH access to hosts without a pre-created
account along with a reference service implementation.

What's so special about SSH?

SSH is a service that provides more than remote interactive shells over an
encrypted channel. It is also has a distributed trust mechanism that helps to
prevent "man in the middle" attacks by caching public keys of hosts at the time
of first use and producing very loud errors if the key changes in the future.
The client's identity is configurable, flexible and can be shared between
different protocols, such as remote shell, git or scp. This makes it largely
transparent to the user once it has been configured and allows them to tailor
their identity for use with select or all hosts.

What's the current state?

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
[Project Gemini](https://gemini.circumlunar.space/) for examples of this kind
of very light account creation.

Unfortunately, these sorts of interactions are not yet practical with SSH,
mostly because there aren't established conventions to permit someone to connect
anonymously at first use of the service. This tends to limit its use to
backend interactions with high degrees of trust.

With a protocol in place an SSH session could be made entirely anonymous
and secure using a combination of client and server settings.

## Client setup

What is needed is to set the following
settings whenever accessing a host anonymously.

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
above can be made for all hosts when connecting as the anonymous user like this.

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

In most configurations this would be all that's needed to access a host name
with a simple SSH command. Some anonymous SSH servers may produce some kind of
greeting when invoked without any command.

```
ssh anonymous@example.com
```

If you don't have a public key for SSH this command will likely fail. You can
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

If you want to use the same identity for all of your SSH hosts then this is
nearly sufficient. However, if you want a separate identity for each site
then you can configure OpenSSH to do that too. This is a strength of SSH as
a protocol since it permits different kinds of identity management depending
on your needs. OpenSSH is so popular that a wide variety of SSH clients and
tools will work with its configuration files.

There is a special include at the end of the ssh configuration above that
will allow separate files with per-host configurations in them. These are
split out into separate files to make it easier to add/remove per-host
configurations because the file names are prefixed with the particular host.
Here is an entry for example.com

```
.ssh/example.com_anon_config:

Match user anonymous host example.com
  SetEnv HOST=example.com
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
create a directory for it ~/.ssh/newhost.com and use ssh-keygen tool to generate
the key in there. Note that there are other kinds of cryptographic algorithms
that we can use with the key. RSA is very commonly used and most likely to work
with an SSH server that is configured for anonymous access.

```
ssh-keygen -b 2048 -t rsa -f ~/.ssh/example.com_anon_id_rsa -q -N ""
```

Now, when you conect to anonymous@example.com it will use this key for that
site and none of the others restricting your identity to only that one site.
This can be useful to help and avoid certain user tracking mechanisms popular
on the web. If you ever want to use a new identity with this site then you can
delete the example.com_anon_id* files and regenerate them.

You may have noticed that we forgot about the LANG and TZ environment variables
described at the top of this section. It turns out that most Unix-based
OSes will usually set this and send it with most OpenSSH client sessions to
the SSH server. Depending of your use of ssh and what tools are being provided
by the host the preferred timezone can be useful for presenting date information
in a form that is the most useful to you. You'll notice that we set the TZ
environment variable with a SetEnv like in your .ssh/config file, but this can
be changed to the preferred timezone string to suit the user's needs.

It is expected that much of this configuration can be automated (or not) with
special tools that are developed for anonymous access on behalf of the user
and their preferences. Also, the idea will be that the configuration is shared
so that you you can different tools hosted by the SSH servers and different
clients sharing your preferences for how your identity is managed and the
trust relationships that you've established.

Whenever you first connect with an SSH server you get a prompt to add it
to the known hosts. If the key changes then you get a very loud error message
warning you about it, whether you're using git, rsync, scp or any tool that
there has been a breach in the trust of that site at any time. Most of these
tools will relay that message reliably.

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

Note that when SSH anonymously there are no guarantees that the server will
support any particular command, or even interactive shell access. In this
framework it is left up to the user to discover hosts and side-band
communication channels via search engines, DNS, or even word of mouth to
know the kinds of services being offered by the host. One recommendation to
service owners is to have some kind of useful information available with a
bare ssh request to their site.

```
$ ssh anonymous@newhost.com

Welcome

We provide git, scp and rsync access for tracking your tee-off times
at the golf course.
```

## Servers

Now that clients have a method of making anonymous requests with key-based
identification servers can be deployed to support them. In the previous section
you may have noticed that the port number is specified as 1966 and not the
usual port 22. This choice was made for a number of reasons. Using a port above
1000 makes it unprivileged and gives administrators more flexibility to run
the server as a non-root user on Unix systems, which can be granted limited
permissions as part of a layered security approach.

OpenSSH itself does not easily allow access to clients without either a
password or knowledge of their private key before connecting. If someday it
were to support such a capability it could be configured to listen to both
the standard port 22 and 1966 at the same time. Until that time, it is most
likely that a separate SSH compatible server technology will be used in which
case it will need a separate port number so that it can run concurrently with
OpenSSH in some configurations. Virtual hosting and proxying is not currently
supported with OpenSSH, which limits some of the options.

An SSH server implementation that support anonymous access would need to be
capable of running SSH sessions as user "anonymous" that present a public key
that is unknown to it. The reference implementation here is an example of a
compliant SSH server that can be flexibly deployed and run in any OS user
account in a variety of operating systems. Hopefully, there will be a variety
of implementations to suit different requirements.

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
Connection to someplace.com closed by remote host.
```

### Allowed command list

Since the recommended policy is to block any interactive shell access to the
anonymous SSH server, it becomes much easier to implement a list of
allowed commands to run. Many of the tools that run on top of SSH rely on
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
command-allow-list:

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
bindings/myservice.com:

/=/var/srv/content
/tmp=/var/srv/tmp
```

The bindings also serve as a kind of list of allowed locations, which helps
with layered security. The SSH server will convert virtual paths to physical
ones before invoking a command and passing in the path parameter(s) with
special care taken to avoid maliciously constructed paths from escaping the
virtual directory structure.

### Time of First Use (TOFU) Accounts

Time of first use accounts represent accounts that are generated on-the-fly
when a public key first passes the authentication challenge with SSH, which
verifies that the entity posesses the matching private key, and accessing a
service. These are different than the operating system accounts of the server
because no side-channel passing of information, such as user names, passwords
or even public keys are required. Instead, if a service supports them there
will be some record, storage location allocated for that public key so that
this information can be used when the user interacts with it in the future.

If there are special files, directories, or other resources available for
a TOFU user for discoverability purposes these are made available in a special
path ~, which both matches existing Unix conventions and is typical for
regular SSH accounts too. Note that in URL's the path should be interpreted
as /~ as the URL specification requires a / as a delimiter between the host
(and port) and the path.

Access to other TOFU users' data is left out of scope for the purposes of this
framework. The recommendation is that such sharing should probably require
non-anonymous access so that more details about the user can be known, such as
some kind of unique, but memorable user name.

It is expected that for most server implementations that are file system
based the TOFU user's private information will be stored in a directory with
a name matching the fingerprint of the public key. After authentication paths
prefixed with a ~ will bind to that location.

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
of path structure than another one. This is why the bindings above are
in a file that matches the name of the virtual host where they apply. For
another virtual host things could look much different and user may never
notice, except if they look at the IP address.

```
bindings/soccer-times.com:

/=/var/srv/soccer
/fifa-schedules=/var/srv/fifa
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
isolation can be achieved with containers, such as Docker where only the
paths that are permitted will be mounted into the container. If an intrusion
were possible then the filesystem view would only be to the container and not
the parent OS. Virtual machines could also function in the same way.

There are measures recommended above that would limit the amount of time that
a command may be allowed to run without any network activity before it is
forcibly closed. This is an effort to help prevent excessive CPU usage on
the server. However, it may be possible that there are other critical things
running there. It depends on how the deployment is being managed. Docker
containers and VM's would offer a way to limit CPU and memory consumption to
help and improve isolation from other critical processes.

As anonymous SSH services become more widely deployed other security concerns
and solutions may appear over time.

## Summary

A framework has been developed to facility robust anonymous SSH access from both
client and server perspectives in the hopes that more services will be deployed
on the internet. While other anonymous access services, such as HTTP exist out
there, many of them only support a narrow set of protocols and tools that work
with them instead of supporting a number of different ones that can use a single
set of trust and identity settings transparently. The hope is that this might
help to foster more of a Unix philosophy in hosted internet services where
each tool can be more focused on its task, but also combined to support a
wealth of flexibility and functionality.
