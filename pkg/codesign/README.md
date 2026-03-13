# Codesign
Pure Go implementation of Mach-O ad-hoc signing (ldid style).

## Features
-   Pseudo-signing (ad-hoc) for Mach-O binaries.
-   Entitlement injection from XML.
-   SHA-256 page hashing.
-   Support for single-architecture binaries.
-   Automatic update of `__LINKEDIT` segment size and `LC_CODE_SIGNATURE` load command.

## To Do
-   [ ] Universal (FAT) binary support.
-   [ ] CMS/Real signing with certificates (not needed for ldid replacement).
-   [ ] Adding `LC_CODE_SIGNATURE` if not present in the original binary.
