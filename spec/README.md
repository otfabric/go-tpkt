# Reference specifications

The two RFC texts in this directory are the wire specification the package is checked against. They are copies for reference; the authoritative texts are the RFC Editor's.

| File | Document | Source | SHA-256 of the file in this directory |
|---|---|---|---|
| `rfc1006.txt` | RFC 1006, "ISO Transport Service on top of the TCP", Version: 3, May 1987 | <https://www.rfc-editor.org/rfc/rfc1006.txt> | `d0f610d05a8e1d3f402d1113d80132ffae81decc9e092796f4b8d03425328e98` |
| `rfc2126.txt` | RFC 2126, "ISO Transport Service on top of TCP (ITOT)", March 1997 | <https://www.rfc-editor.org/rfc/rfc2126.txt> | `07ac95a13c9d0af4ede6ff2f5fc0cbafcb959993c4c1a95f59c5e5a1bdbb85d6` |

## Verification

The dates on which the texts were retrieved were not recorded. `rfc1006.txt` has been in this repository since 2026-03-13 and `rfc2126.txt` since 2026-07-16.

On 2026-11-14 both files were compared with the texts at the source URLs above. They differ from those texts in blank lines and form feeds only: after trailing whitespace is stripped, no non-blank line differs.

Because of that difference in blank lines, the checksums above are those of the files in this directory, not of the texts as the RFC Editor serves them. `SHA256SUMS` holds the same checksums; verify the files with:

```sh
cd spec && shasum -a 256 -c SHA256SUMS
```

To check a file against its source again, download the text and compare the two with blank lines and form feeds removed.

## Licence

RFCs are published by the RFC Editor. Both documents state "Distribution of this memo is unlimited." Their copying and redistribution fall under the IETF Trust's terms for RFCs (<https://trustee.ietf.org/license-info>), not under the MIT licence of this repository.
