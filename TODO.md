# Scrutineer — Deferred Features

Features deferred beyond v0.0.1, tracked for future implementation.

## HTTP/3 (QUIC)
- Full QUIC transport protocol (RFC 9000, 9001)
- HTTP/3 framing (RFC 9114)
- Blocked on Go stdlib adoption or from-scratch implementation
- Revisit when `golang.org/x/net/quic` is promoted to stdlib or stable

## Video Recording
- Record browser test runs as video files
- Requires frame capture from CDP and video encoding

## SMTP
- Send, auth, envelope validation
- Implement from scratch or evaluate `golang.org/x/` options at that time

## IMAP
- Mailbox access, search, fetch
- Implement from scratch or evaluate `golang.org/x/` options at that time

## POP
- Mailbox retrieval
- Implement from scratch or evaluate `golang.org/x/` options at that time

## Additional Output Formats
- JUnit XML — for CI integration (Jenkins, GitHub Actions, GitLab)
- TAP (Test Anything Protocol)
