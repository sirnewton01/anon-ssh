# Gemini Command

The gemini command invokes a Gemini transaction with either on either a Gemini
URL, Gemini SSH  or a local file path. In the case of the SSH URL it will initiate
the connection over SSH and invoke the transaction on the remote host using
this command, but with a file path. Also, in the latter case the gemini command
will set up any extra SSH configuration for the remote host to set up anonymous
SSH access, such as a public key if none exists.

```
gemini [gemssh|gemini]://[username@]somehost/some/path
```

Note that if no username is provided then it will be assumed that it is
anonymous, following the Anonymous SSH framework. It will run the gemini command
on the remote system substituting the URL for just the remote file path. This
command is analogous to the request portion of the Gemini Protocol with some
notable differences. The full URL is not provided here and there is no CRLF
terminator. Note that the path is expected to be UTF-8 encoded.

```
gemini /some/path
```

When the command is invoked on a local file path it will return a well-formed
Gemini protocol response. If the local file path exists and can be read by the
current user then a 20 status is returned with the media type along with the
contents of the file. Otherwise, a 51 (Not Found) status is returned.

```
20 text/plain
This is the contents of the file.
```

If the path is a directory then a check will be performed whether there is an
index.gmi file in that location that is readable by the current user in which
case the status is 20 and the file contents will be sent.

When the gemini command is invoked with the server flag it starts a gemini
server for a local content path.

```
gemini --server [--listen-address <address>] <cert> <key> <capsule-content-path>
```

