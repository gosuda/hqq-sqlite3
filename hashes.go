package sqlite

import "gosuda.org/hqq/secureexec"

// Version follows the v<WrapperMajor>.<WrapperMinor>.<NormalizedCVersion> scheme,
// where 3540100 corresponds to SQLITE_VERSION_NUMBER (SQLite 3.54.1).
const (
	Version             = "v0.1.3540100"
	SQLiteVersionNumber = 3540100
)

// ReleaseHashes maps "<GOOS>/<GOARCH>" to the SHA-256 digest of the prebuilt
// hqq-sqlite-server binary published with this module version. The release
// workflow populates these entries alongside SLSA build provenance attestations.
var ReleaseHashes = map[string]secureexec.Hash{}

// ExpectedHash returns the pinned SHA-256 digest of hqq-sqlite-server for goos/goarch.
func ExpectedHash(goos, goarch string) (secureexec.Hash, bool) {
	h, ok := ReleaseHashes[goos+"/"+goarch]
	return h, ok
}

func mustHash(s string) secureexec.Hash {
	h, err := secureexec.ParseHash(s)
	if err != nil {
		panic(err)
	}
	return h
}
