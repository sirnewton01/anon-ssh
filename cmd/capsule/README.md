# Capsule Helper

The capsule helper command takes as input a host of a capsule that will be
accessed and ensures that an SSH configuration is in place to connect to
the host as a capsule. This involves checking that the general capsule
configuration is in place for the SSH client. When that is in place it will
check that there is the HOST environment variable and generate a set of
RSA encryption keys to use with that one host. The command will return
quickly if there's nothing to do. It also outputs the SSH address to be
used with the capsule. This makes it useful in UNIX shells as a subcommand
to SSH tools that are not capsule aware. Here are some example usages.

```
$ capsule somehost
capsule@somehost
$ ssh $(capsule somehost)
Welcome to somehost...
$ git clone $(capsule somegitrepo):/mysrc
Cloning mysrc...
$ rsync -a $(capsule mybackup):/backup1 .
```

Ideally, someday this command will be no longer needed when either the
popular SSH tools support capsules or if SSH itself supports them. Meanwhile,
this tool is in place as a convenience.
