# Gemini Command

The gemini command invokes a Gemini transaction with either on either a Gemini
URL, Gemini SSH  or a local file path. In the case of the gemcap URL it will
initiate the connection over SSH and invoke the transaction on the remote host using
this command, but with a file path. Also, in the latter case the gemini command
will set up any extra SSH configuration for the remote host to set up capsule
SSH access, such as a public key if none exists. Note that SSH-style addresses
can also be used instead of URL's.

```
gemini [gemcap|gemini]://[username@]somehost/some/path
gemini [username@]somehost:/some/path
```

Note that if no username is provided then it will be assumed that it is
capsule, following the SSH Capsule framework. It will run the gemini command
on the remote system substituting the URL for just the remote file path. This
command is analogous to the request portion of the Gemini Protocol with some
notable differences. The full URL is not provided here and there is no CRLF
terminator. Note that the path is expected to be UTF-8 encoded. This command
can produce error messages on standard error and exit codes related to transport
level errors in either SSH or TLS. Note that exit code will be zero for success
2x responses and will be the status code itself for error statuses.

```
gemini /some/path
```

When the command is invoked on a local file path it will return a well-formed
Gemini protocol response, except that the status line is sent to stderr and
the content is sent to stdout. This is done to allow independent redirection
of the content from status and warning messages. Also, it follows established
C and UNIX standards so that this command works more like many command-line
tools. The well-defined status line allows gemini-aware clients to parse it
and react in different ways depending on the status code.

If the local file path exists and can be read by the current user then a 20
status is returned with the media type along with the contents of the file.
Otherwise, a 51 (Not Found) status is returned. Success statuses will exit
with code 0, any non-success statuses exit with the code
set to the status integer.

```
20 text/plain
This is the contents of the file.
```

If the path is a directory then a check will be performed whether there is an
index.gmi file in that location that is readable by the current user in which
case the status is 20 and the file contents will be sent.

## Server mode

When the gemini command is invoked with the server flag it starts a gemini
server and supports simple read access to the filesystem with the capsule
content path as the root.

```
gemini --server [--listen-address <address>] <cert> <key> <capsule-content-path>
```

