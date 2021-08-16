# Docker Quick Start Guide

There is a Docker image you can use to set up a capsule and connect to
it associated with this project. This can help you get up and running
quickly so that you can try out SSH capsules and see how they work. This
guide assumes that you have Docker already set up on your computer and are
familiar with it.

First, let's create a directory for our capsule and generate an SSH host
key.

```
% mkdir srv
% ssh-keygen -m PEM -f srv/hostkey -N ""
```

Now, we can pull the docker image that will serve both as a server container
and a client container.

```
% docker pull ghcr.io/sirnewton01/ssh-capsules
```

The server container will be given access to our local srv directory and
port 1966 forwarded to our host machine. Note that on macOS you may need
to find the VirtualBox VM that docker is using and manually forward port
1966 from the VM to the host for this to work.

```
% docker run -it --rm --user $(id -u):$(id -g) -v $(pwd)/srv:srv -p 1966:1966 ghcr.io/sirnewton01/ssh-capsules # Linux will match the user ID and group ID to the provided ID's of the host
% docker run -it --rm --user 1000:$(id -g) -v $(pwd)/srv:/srv -p 1966:1966 ghcr.io/sirnewton01/ssh-capsules # macOS volumes seem to be set always to uid 1000

2021/08/16 01:27:23 Server started on addresss :1966
```

You'll notice that the srv/ directory has some new content in it under
srv/capsule. The top-level folder contains configuration files that you can
change, such as assigning the host name of the capsule and the allowed commands.
If you modify the commands file and uncomment the gemini command then we can
use gemini transactions with this capsule. The same applies to scp, git or even
cat commands or any other shell commands that you want in the future. This
is a measure of security against clients running commands that could have
unintended consequences. Uncomment the gemini command in the commands file and
save it.

```
srv/capsule/commands:
...
gemini <path>
...
```

The capsule has files that form the content of the capsule. You can see that
there's an index.gmi at the top of srv/capsule/content that you can modify to
have what content you want. You can add any other files that you want in there
to form the content of your capsule: images, markdown, html, even git
repositories. Now you have a working SSH capsule running, let's connect to
it with a docker client that will be anonymous and have a new identity (key)
each time that we run it.

```
% docker run -i --rm ghcr.io/sirnewton01/ssh-capsules gemini capsule@mycomputer:/ 2>/dev/null # change mycomputer to be your computer's name
Welcome to my capsule!
```

We redirected the stderr to /dev/null because it contains informational
messages from SSH indicating that it added the IP address of your computer to
the known hosts with the host key. SSH keeps track of this in case of an
attempt to spoof the capsule in the future. Also, stderr has metadata coming
from gemini indicating the success/failure status of the transaction and media
type. As an exercise, you can try this again without the ```2>/dev/null``` and
see these messages.

Our capsule implementation has a greeting message if you attempt to connect to
it with ssh directly. Since the ssh command does not yet have the logic to
connect to SSH capsules, there is a helper tool called "capsule" that will
help us to use tools that work with SSH, but don't yet have capsule capability.

```
% docker run -i --rm ghcr.io/sirnewton01/ssh-capsules bash -c 'ssh $(capsule mycomputer)'
Pseudo-terminal will not be allocated because stdin is not a terminal.
Warning: Permanently added the RSA host key for IP address '[192.168.1.174]:1966' to the list of known hosts.
Welcome to mycomputer, user capsule
Your public key is ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC2x0Eg53pLhTXqhB1yqf0qiQL6SsRZ5j6c51RJEL1sVnX2WSGSPGeETxTQ5B77cTYD4+/koNTaa8562FlpTQ0BgBzBvcbE/tptrGsHBgnsgv/HGRYYCr2os3tNJ7IndVPdjUKZ2hY66G7X9bYDQXcYug4ZrnYusaQB3HajbFBrJDn9N3OcgCwATzuGHEPYKpHcs9rcLw6hgpBH359X5zLX1rUzh68N20FREXrJTf922qygwsvvUTS0G7/C8LJ6qhotqYb26UajYtLERybZOg8FjYbZ3N/e9vfoPm3QNzyqDPwCp6c6hsGTgtfB+9jNNjQJKsccN2x9so7/k0MEtICX
Your environment: [HOST=mycomputer]
```

There's a little extra information on stderr at the beginning for anyone that's
curious about what happened in the transaction. The capsule echos back the
virtual host, the user name (usually capsule, when using SSH capsule), the
public key and environment. This capsule implementation provides this
information as a way for you to verify what information you are giving to the
capsule with each transaction.

If you try this same command above again you'll notice
that the public key will be different each time. Using docker in this way
ensures that the SSH key is regenerated on each request, which can be a good
way to avoid certain types of tracking. Depending on your needs you can use
a sandbox, such as docker to throw away your key with every transaction. The
SSH capsule tools, such as gemini and capsule will generally generate a reusable
key for each host in cases where you want to be able to use your key to get
additional privileges with a capsule based on your identity through your key.
In this case, you would probably want to install the gemini and capsule tools
into your host environment outside of docker. It's ultimately up to you the
level of anonymity that you want to achieve.

That's about it for this quickstart guide. In summary, SSH capsules are
straight forward to set up in a Docker environment and offer an extra level
of sandboxing as part of a layered security approach. Also, client tools such
as gemini and capsule (the helper tool) can be used in a Docker environment to
demonstrate the capabilities as well as randomize the key identity on each
transaction. For additional experimentation you can look at enabling the cat,
scp and git commands on your capsule and experiment with them, or possibly even
other commands that this tutorial hasn't considered.
